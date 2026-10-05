// Package thumbs generates and caches grid thumbnails on demand.
package thumbs

import (
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/gen2brain/heic"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"github.com/sjakovic/yearfold/internal/store"
)

const Size = 360

var ErrUnsupported = errors.New("thumbnail not supported")

type Manager struct {
	dir string
	sem chan struct{}

	mu      sync.Mutex
	pending map[int64]*sync.Mutex
}

func New(dir string) *Manager {
	n := runtime.NumCPU() - 1
	if n < 1 {
		n = 1
	}
	return &Manager{dir: dir, sem: make(chan struct{}, n), pending: map[int64]*sync.Mutex{}}
}

func (m *Manager) path(id int64) string {
	return filepath.Join(m.dir, fmt.Sprintf("%02x", id&0xff), fmt.Sprintf("%d.jpg", id))
}

func (m *Manager) Remove(id int64) { os.Remove(m.path(id)) }

func (m *Manager) Clear() error {
	if err := os.RemoveAll(m.dir); err != nil {
		return err
	}
	return os.MkdirAll(m.dir, 0o755)
}

func (m *Manager) Get(id int64, src, ext, kind string, orientation int) (string, error) {
	out := m.path(id)
	if _, err := os.Stat(out); err == nil {
		return out, nil
	}

	m.mu.Lock()
	l := m.pending[id]
	if l == nil {
		l = &sync.Mutex{}
		m.pending[id] = l
	}
	m.mu.Unlock()
	l.Lock()
	defer func() {
		l.Unlock()
		m.mu.Lock()
		delete(m.pending, id)
		m.mu.Unlock()
	}()
	if _, err := os.Stat(out); err == nil {
		return out, nil
	}

	m.sem <- struct{}{}
	defer func() { <-m.sem }()

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}
	tmp := out + ".tmp"
	var err error
	if kind == store.KindVideo {
		err = videoFrame(src, tmp)
	} else {
		err = imageThumb(src, ext, orientation, tmp)
	}
	if err != nil {
		os.Remove(tmp)
		return "", err
	}
	return out, os.Rename(tmp, out)
}

func imageThumb(src, ext string, orientation int, out string) error {
	img, err := Decode(src, ext)
	if err != nil {
		return err
	}
	return writeJPEG(out, Orient(Resize(img, Size), orientation))
}

func writeJPEG(out string, img image.Image) error {
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 82}); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func Decode(src, ext string) (image.Image, error) {
	f, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return decodeReader(f, ext)
}

func decodeReader(r io.Reader, ext string) (image.Image, error) {
	switch ext {
	case "heic", "heif":
		return heic.Decode(r)
	case "jpg", "jpeg", "png", "gif", "webp", "bmp", "tif", "tiff":
		img, _, err := image.Decode(r)
		return img, err
	}
	return nil, ErrUnsupported
}

func Resize(img image.Image, max int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= max && h <= max {
		return img
	}
	nw, nh := max, h*max/w
	if h > w {
		nw, nh = w*max/h, max
	}
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}

func Orient(img image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	swap := orientation >= 5
	dw, dh := w, h
	if swap {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch orientation {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func videoFrame(src, out string) error {
	if ffmpegFrame(src, out) == nil {
		return nil
	}
	if runtime.GOOS == "darwin" {
		return quickLookFrame(src, out)
	}
	return ErrUnsupported
}

func ffmpegFrame(src, out string) error {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return ErrUnsupported
	}
	scale := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", Size, Size)
	cmd := exec.Command(ffmpeg, "-y", "-loglevel", "error", "-ss", "1", "-i", src,
		"-frames:v", "1", "-vf", scale, "-f", "image2", "-c:v", "mjpeg", out)
	if err := cmd.Run(); err != nil {
		return ErrUnsupported
	}
	if st, err := os.Stat(out); err != nil || st.Size() == 0 {
		return ErrUnsupported
	}
	return nil
}

const quickLookTimeout = 30 * time.Second

func quickLookFrame(src, out string) error {
	tmp, err := os.MkdirTemp("", "yearfold-ql-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	ctx, cancel := context.WithTimeout(context.Background(), quickLookTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "qlmanage", "-t", "-s", fmt.Sprint(Size), "-o", tmp, src)
	if err := cmd.Run(); err != nil {
		return ErrUnsupported
	}
	img, err := Decode(filepath.Join(tmp, filepath.Base(src)+".png"), "png")
	if err != nil {
		return ErrUnsupported
	}
	return writeJPEG(out, Resize(img, Size))
}
