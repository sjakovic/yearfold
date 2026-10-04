package meta

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"sort"
)

// This file sets the capture date inside an EXIF block without disturbing
// anything else in it. Existing bytes are never moved: the date and a new
// copy of the IFD that references it are appended to the end of the block
// and the pointer to that IFD is updated. Every other tag, including maker
// notes and tags this code knows nothing about, therefore stays valid.

const (
	tagExifIFD           = 0x8769
	tagDateTimeOriginal  = 0x9003
	tagDateTimeDigitized = 0x9004

	typeASCII = 2
	typeLong  = 4

	ifdEntrySize = 12
	// stampLen is "YYYY:MM:DD HH:MM:SS".
	stampLen = 19
)

var (
	errBadEXIF  = errors.New("malformed EXIF data")
	errEXIFFull = errors.New("no room left in the EXIF block")

	// emptyTIFF is a little-endian EXIF block with an empty first IFD.
	emptyTIFF = []byte{'I', 'I', 42, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0}
)

// byteOrder can both decode and append integers in a TIFF's byte order.
type byteOrder interface {
	binary.ByteOrder
	binary.AppendByteOrder
}

type ifdEntry [ifdEntrySize]byte

type ifd struct {
	offset  uint32
	entries []ifdEntry
	next    uint32
}

func tiffByteOrder(tiff []byte) (byteOrder, error) {
	if len(tiff) < 8 {
		return nil, errBadEXIF
	}
	var bo byteOrder
	switch string(tiff[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return nil, errBadEXIF
	}
	if bo.Uint16(tiff[2:4]) != 42 {
		return nil, errBadEXIF
	}
	return bo, nil
}

func readIFD(tiff []byte, bo byteOrder, offset uint32) (ifd, error) {
	d := ifd{offset: offset}
	start := int64(offset)
	if start+2 > int64(len(tiff)) {
		return d, errBadEXIF
	}
	count := int64(bo.Uint16(tiff[start:]))
	end := start + 2 + count*ifdEntrySize
	if end+4 > int64(len(tiff)) {
		return d, errBadEXIF
	}
	for p := start + 2; p < end; p += ifdEntrySize {
		var e ifdEntry
		copy(e[:], tiff[p:p+ifdEntrySize])
		d.entries = append(d.entries, e)
	}
	d.next = bo.Uint32(tiff[end:])
	return d, nil
}

func newEntry(bo byteOrder, tag, typ uint16, count, value uint32) ifdEntry {
	var e ifdEntry
	bo.PutUint16(e[0:], tag)
	bo.PutUint16(e[2:], typ)
	bo.PutUint32(e[4:], count)
	bo.PutUint32(e[8:], value)
	return e
}

// appendIFD writes an IFD at the end of tiff and returns its offset.
func appendIFD(tiff []byte, bo byteOrder, entries []ifdEntry, next uint32) ([]byte, uint32) {
	sort.SliceStable(entries, func(i, j int) bool {
		return bo.Uint16(entries[i][:2]) < bo.Uint16(entries[j][:2])
	})
	tiff = padEven(tiff)
	offset := uint32(len(tiff))
	tiff = bo.AppendUint16(tiff, uint16(len(entries)))
	for _, e := range entries {
		tiff = append(tiff, e[:]...)
	}
	return bo.AppendUint32(tiff, next), offset
}

func padEven(b []byte) []byte {
	if len(b)%2 == 1 {
		return append(b, 0)
	}
	return b
}

// setTIFFDate returns a copy of an EXIF (TIFF) block whose DateTimeOriginal
// and DateTimeDigitized are stamp, in the form "YYYY:MM:DD HH:MM:SS".
func setTIFFDate(tiff []byte, stamp string) ([]byte, error) {
	if len(stamp) != stampLen {
		return nil, errors.New("invalid date stamp")
	}
	bo, err := tiffByteOrder(tiff)
	if err != nil {
		return nil, err
	}
	out := append([]byte(nil), tiff...)
	root, err := readIFD(out, bo, bo.Uint32(out[4:8]))
	if err != nil {
		return nil, err
	}

	// Carry over every tag of the existing Exif IFD except the dates.
	var exifEntries []ifdEntry
	pointer := -1
	for i, e := range root.entries {
		if bo.Uint16(e[:2]) == tagExifIFD {
			pointer = i
		}
	}
	if pointer >= 0 {
		old, err := readIFD(out, bo, bo.Uint32(root.entries[pointer][8:]))
		if err != nil {
			return nil, err
		}
		for _, e := range old.entries {
			if tag := bo.Uint16(e[:2]); tag != tagDateTimeOriginal && tag != tagDateTimeDigitized {
				exifEntries = append(exifEntries, e)
			}
		}
	}

	out = padEven(out)
	stampOffset := uint32(len(out))
	out = append(append(out, stamp...), 0)
	for _, tag := range []uint16{tagDateTimeOriginal, tagDateTimeDigitized} {
		exifEntries = append(exifEntries, newEntry(bo, tag, typeASCII, stampLen+1, stampOffset))
	}
	out, exifOffset := appendIFD(out, bo, exifEntries, 0)

	if pointer >= 0 {
		// Repoint the existing ExifIFD entry in place.
		pos := int(root.offset) + 2 + pointer*ifdEntrySize + 8
		bo.PutUint32(out[pos:], exifOffset)
		return out, nil
	}
	// The first IFD has no Exif IFD yet: append a copy that points to it.
	entries := append(append([]ifdEntry(nil), root.entries...), newEntry(bo, tagExifIFD, typeLong, 1, exifOffset))
	out, rootOffset := appendIFD(out, bo, entries, root.next)
	bo.PutUint32(out[4:8], rootOffset)
	return out, nil
}

const (
	jpegMarkerSOI  = 0xD8
	jpegMarkerEOI  = 0xD9
	jpegMarkerSOS  = 0xDA
	jpegMarkerAPP0 = 0xE0
	jpegMarkerAPP1 = 0xE1
	// A JPEG segment length is 16 bits and includes its own two bytes.
	jpegMaxSegment = 0xFFFF
)

var exifHeader = []byte("Exif\x00\x00")

// setJPEGDate returns the JPEG with the capture date set, adding an EXIF
// segment when the file has none.
func setJPEGDate(data []byte, stamp string) ([]byte, error) {
	if len(data) < 4 || data[0] != 0xFF || data[1] != jpegMarkerSOI {
		return nil, errors.New("not a JPEG file")
	}
	// A new EXIF segment goes after SOI and the JFIF header, if present.
	insertAt := 2
	for pos := 2; ; {
		if pos+4 > len(data) || data[pos] != 0xFF {
			return nil, errors.New("malformed JPEG file")
		}
		marker := data[pos+1]
		if marker == jpegMarkerSOS || marker == jpegMarkerEOI {
			break
		}
		size := int(binary.BigEndian.Uint16(data[pos+2:]))
		end := pos + 2 + size
		if size < 2 || end > len(data) {
			return nil, errors.New("malformed JPEG file")
		}
		payload := data[pos+4 : end]
		if marker == jpegMarkerAPP1 && bytes.HasPrefix(payload, exifHeader) {
			tiff, err := setTIFFDate(payload[len(exifHeader):], stamp)
			if err != nil {
				return nil, err
			}
			segment, err := exifSegment(tiff)
			if err != nil {
				return nil, err
			}
			return splice(data, pos, end, segment), nil
		}
		if marker == jpegMarkerAPP0 && pos == insertAt {
			insertAt = end
		}
		pos = end
	}

	tiff, err := setTIFFDate(emptyTIFF, stamp)
	if err != nil {
		return nil, err
	}
	segment, err := exifSegment(tiff)
	if err != nil {
		return nil, err
	}
	return splice(data, insertAt, insertAt, segment), nil
}

func exifSegment(tiff []byte) ([]byte, error) {
	size := 2 + len(exifHeader) + len(tiff)
	if size > jpegMaxSegment {
		return nil, errEXIFFull
	}
	seg := []byte{0xFF, jpegMarkerAPP1}
	seg = binary.BigEndian.AppendUint16(seg, uint16(size))
	seg = append(seg, exifHeader...)
	return append(seg, tiff...), nil
}

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

// setPNGDate returns the PNG with the capture date set in its eXIf chunk,
// adding the chunk before the image data when the file has none.
func setPNGDate(data []byte, stamp string) ([]byte, error) {
	if !bytes.HasPrefix(data, pngSignature) {
		return nil, errors.New("not a PNG file")
	}
	insertAt := -1
	for pos := len(pngSignature); pos+12 <= len(data); {
		size := int(binary.BigEndian.Uint32(data[pos:]))
		end := pos + 12 + size
		if size < 0 || end > len(data) {
			return nil, errors.New("malformed PNG file")
		}
		switch string(data[pos+4 : pos+8]) {
		case "eXIf":
			tiff, err := setTIFFDate(data[pos+8:pos+8+size], stamp)
			if err != nil {
				return nil, err
			}
			return splice(data, pos, end, pngChunk("eXIf", tiff)), nil
		case "IDAT":
			if insertAt < 0 {
				insertAt = pos
			}
		}
		pos = end
	}
	if insertAt < 0 {
		return nil, errors.New("malformed PNG file")
	}
	tiff, err := setTIFFDate(emptyTIFF, stamp)
	if err != nil {
		return nil, err
	}
	return splice(data, insertAt, insertAt, pngChunk("eXIf", tiff)), nil
}

func pngChunk(kind string, payload []byte) []byte {
	chunk := binary.BigEndian.AppendUint32(nil, uint32(len(payload)))
	body := append([]byte(kind), payload...)
	chunk = append(chunk, body...)
	return binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(body))
}

// splice returns data with data[from:to] replaced by insert.
func splice(data []byte, from, to int, insert []byte) []byte {
	out := make([]byte, 0, len(data)-(to-from)+len(insert))
	out = append(out, data[:from]...)
	out = append(out, insert...)
	return append(out, data[to:]...)
}
