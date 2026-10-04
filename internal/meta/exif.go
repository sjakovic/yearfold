// Package meta reads metadata embedded in media files and Google Takeout sidecars.
package meta

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
	"time"

	"github.com/bep/imagemeta"
	"github.com/gen2brain/heic"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// Info is the metadata embedded in an image file.
type Info struct {
	TakenAt     int64
	Width       int
	Height      int
	Orientation int
	Camera      string
	HasGPS      bool
	Lat, Lon    float64
	// Tags holds every decoded tag as "Source:Name" -> printable value.
	Tags map[string]string
}

var formats = map[string]imagemeta.ImageFormat{
	"jpg": imagemeta.JPEG, "jpeg": imagemeta.JPEG, "tif": imagemeta.TIFF, "tiff": imagemeta.TIFF,
	"png": imagemeta.PNG, "webp": imagemeta.WebP, "heic": imagemeta.HEIF, "heif": imagemeta.HEIF,
	"avif": imagemeta.AVIF, "dng": imagemeta.DNG,
}

const maxTagLen = 300

// Extract reads EXIF/IPTC/XMP and dimensions. Formats without embedded
// metadata support still get dimensions when the image can be decoded.
func Extract(path, ext string) (Info, error) {
	info := Info{Orientation: 1, Tags: map[string]string{}}
	f, err := os.Open(path)
	if err != nil {
		return info, err
	}
	defer f.Close()

	if format, ok := formats[ext]; ok {
		var tags imagemeta.Tags
		res, err := imagemeta.Decode(imagemeta.Options{
			R:               f,
			ImageFormat:     format,
			Sources:         imagemeta.EXIF | imagemeta.IPTC | imagemeta.XMP | imagemeta.CONFIG,
			ShouldHandleTag: func(imagemeta.TagInfo) bool { return true },
			Timeout:         15 * time.Second,
			HandleTag: func(ti imagemeta.TagInfo) error {
				tags.Add(ti)
				v := strings.TrimSpace(fmt.Sprint(ti.Value))
				if v == "" || len(v) > maxTagLen {
					return nil
				}
				info.Tags[sourceName(ti.Source)+":"+ti.Tag] = v
				return nil
			},
		})
		if err == nil {
			info.Width, info.Height = res.ImageConfig.Width, res.ImageConfig.Height
		}
		exif := tags.EXIF()
		if _, ok := exif["DateTimeOriginal"]; ok {
			if t, err := tags.GetDateTime(); err == nil && t.Year() > 1970 {
				info.TakenAt = wallClock(t).Unix()
			}
		}
		if lat, lon, err := tags.GetLatLong(); err == nil && (lat != 0 || lon != 0) {
			info.HasGPS, info.Lat, info.Lon = true, lat, lon
		}
		if ti, ok := exif["Orientation"]; ok {
			if o := toInt(ti.Value); o >= 1 && o <= 8 {
				info.Orientation = o
			}
		}
		var cam []string
		for _, k := range []string{"Make", "Model"} {
			if ti, ok := exif[k]; ok {
				if s := strings.TrimSpace(fmt.Sprint(ti.Value)); s != "" {
					cam = append(cam, s)
				}
			}
		}
		// Many cameras repeat the make inside the model name.
		if len(cam) == 2 && strings.HasPrefix(strings.ToLower(cam[1]), strings.ToLower(cam[0])) {
			cam = cam[1:]
		}
		info.Camera = strings.Join(cam, " ")
	}

	if info.Width == 0 {
		if _, err := f.Seek(0, 0); err == nil {
			var cfg image.Config
			var err error
			if ext == "heic" || ext == "heif" {
				cfg, err = heic.DecodeConfig(f)
			} else {
				cfg, _, err = image.DecodeConfig(f)
			}
			if err == nil {
				info.Width, info.Height = cfg.Width, cfg.Height
			}
		}
	}
	return info, nil
}

// wallClock re-expresses t's local date and time as UTC. EXIF dates carry no
// reliable time zone, so the index keeps what the camera's clock showed and
// the UI displays it unchanged, whatever zone the computer is in.
func wallClock(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
}

func sourceName(s imagemeta.Source) string {
	switch s {
	case imagemeta.EXIF:
		return "EXIF"
	case imagemeta.IPTC:
		return "IPTC"
	case imagemeta.XMP:
		return "XMP"
	}
	return "META"
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case uint16:
		return int(n)
	case uint32:
		return int(n)
	case int32:
		return int(n)
	case uint8:
		return int(n)
	case int64:
		return int(n)
	case uint64:
		return int(n)
	}
	var n int
	fmt.Sscan(fmt.Sprint(v), &n)
	return n
}
