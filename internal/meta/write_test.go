package meta

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var makerData = []byte("MAKERNOTE-PAYLOAD-0123456789")

func cameraTIFF() []byte {
	bo := binary.LittleEndian
	const (
		ifd0Offset  = 8
		ifd0Size    = 2 + 3*ifdEntrySize + 4
		makeOffset  = ifd0Offset + ifd0Size
		makerOffset = makeOffset + 6
		exifOffset  = makerOffset + 28
		exifSize    = 2 + 2*ifdEntrySize + 4
		rationalOff = exifOffset + exifSize
		dateOffset  = rationalOff + 8
	)
	b := []byte{'I', 'I', 42, 0}
	b = bo.AppendUint32(b, ifd0Offset)

	b = bo.AppendUint16(b, 3)
	for _, e := range []ifdEntry{
		newEntry(bo, 0x010F, typeASCII, 6, makeOffset), // Make
		newEntry(bo, tagExifIFD, typeLong, 1, exifOffset),
		newEntry(bo, 0xC4A5, 7, uint32(len(makerData)), makerOffset), // vendor tag
	} {
		b = append(b, e[:]...)
	}
	b = bo.AppendUint32(b, 0)
	b = append(b, "Canon\x00"...)
	b = append(b, makerData...)

	b = bo.AppendUint16(b, 2)
	for _, e := range []ifdEntry{
		newEntry(bo, 0x829A, 5, 1, rationalOff), // ExposureTime
		newEntry(bo, tagDateTimeOriginal, typeASCII, stampLen+1, dateOffset),
	} {
		b = append(b, e[:]...)
	}
	b = bo.AppendUint32(b, 0)
	b = bo.AppendUint32(b, 1)
	b = bo.AppendUint32(b, 250)
	return append(b, "1999:01:01 00:00:00\x00"...)
}

func testImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 32, 24))
	for i := range img.Pix {
		img.Pix[i] = uint8(i)
	}
	return img
}

func writeTemp(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestWriteDateAddsEXIFToPlainFiles(t *testing.T) {
	var jpg, pngData bytes.Buffer
	if err := jpeg.Encode(&jpg, testImage(), nil); err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(&pngData, testImage()); err != nil {
		t.Fatal(err)
	}
	first := time.Date(2004, 6, 15, 12, 30, 0, 0, time.UTC)
	second := time.Date(2010, 12, 31, 23, 59, 58, 0, time.UTC)

	for _, tt := range []struct {
		name, ext string
		data      []byte
	}{
		{"a.jpg", "jpg", jpg.Bytes()},
		{"a.png", "png", pngData.Bytes()},
	} {
		p := writeTemp(t, tt.name, tt.data)
		for _, want := range []time.Time{first, second} {
			if err := WriteDate(p, tt.ext, want); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			info, err := Extract(p, tt.ext)
			if err != nil {
				t.Fatal(err)
			}
			if info.TakenAt != want.Unix() {
				t.Errorf("%s: read back %d, want %d", tt.name, info.TakenAt, want.Unix())
			}
			if info.Width != 32 || info.Height != 24 {
				t.Errorf("%s: image damaged, size %dx%d", tt.name, info.Width, info.Height)
			}
		}
		f, err := os.Open(p)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = image.Decode(f)
		f.Close()
		if err != nil {
			t.Errorf("%s no longer decodes: %v", tt.name, err)
		}
		if st, _ := os.Stat(p); st.Mode().Perm() != 0o600 {
			t.Errorf("%s: permissions changed to %v", tt.name, st.Mode().Perm())
		}
	}
}

func TestWriteDateKeepsExistingTags(t *testing.T) {
	var plain bytes.Buffer
	if err := jpeg.Encode(&plain, testImage(), nil); err != nil {
		t.Fatal(err)
	}
	segment, err := exifSegment(cameraTIFF())
	if err != nil {
		t.Fatal(err)
	}
	p := writeTemp(t, "camera.jpg", splice(plain.Bytes(), 2, 2, segment))

	before, _ := Extract(p, "jpg")
	if before.Camera != "Canon" || before.TakenAt != time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC).Unix() {
		t.Fatalf("fixture not readable: %+v", before)
	}

	want := time.Date(2004, 6, 15, 12, 30, 0, 0, time.UTC)
	if err := WriteDate(p, "jpg", want); err != nil {
		t.Fatal(err)
	}
	after, _ := Extract(p, "jpg")
	if after.TakenAt != want.Unix() {
		t.Errorf("date = %d, want %d", after.TakenAt, want.Unix())
	}
	if after.Camera != "Canon" {
		t.Errorf("Make lost: %q", after.Camera)
	}
	if after.Tags["EXIF:ExposureTime"] != before.Tags["EXIF:ExposureTime"] || after.Tags["EXIF:ExposureTime"] == "" {
		t.Errorf("ExposureTime changed: %q -> %q", before.Tags["EXIF:ExposureTime"], after.Tags["EXIF:ExposureTime"])
	}
	data, _ := os.ReadFile(p)
	if !bytes.Contains(data, makerData) {
		t.Error("vendor tag data was dropped")
	}
	for name := range before.Tags {
		if _, ok := after.Tags[name]; !ok {
			t.Errorf("tag %s disappeared", name)
		}
	}
}

func TestWriteDateUnsupportedFormat(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // make sure exiftool is not found
	p := writeTemp(t, "a.heic", []byte("not really heic"))
	if err := WriteDate(p, "heic", time.Now()); !errors.Is(err, ErrCannotEmbed) {
		t.Errorf("err = %v, want ErrCannotEmbed", err)
	}
	if data, _ := os.ReadFile(p); string(data) != "not really heic" {
		t.Error("unsupported file was modified")
	}
}

func TestSetTIFFDateRejectsGarbage(t *testing.T) {
	for _, in := range [][]byte{nil, []byte("II*"), []byte("XX*\x00\x08\x00\x00\x00"), {'I', 'I', 42, 0, 200, 0, 0, 0}} {
		if _, err := setTIFFDate(in, "2004:06:15 12:30:00"); err == nil {
			t.Errorf("setTIFFDate(%q) accepted malformed input", in)
		}
	}
}
