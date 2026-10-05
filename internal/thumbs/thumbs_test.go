package thumbs

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/sjakovic/yearfold/internal/store"
	"github.com/sjakovic/yearfold/internal/testutil"
)

func TestResizeKeepsProportions(t *testing.T) {
	tests := []struct{ w, h, max, wantW, wantH int }{
		{4000, 3000, 360, 360, 270},
		{3000, 4000, 360, 270, 360},
		{200, 100, 360, 200, 100},
		{5000, 2, 360, 360, 1},
	}
	for _, tt := range tests {
		got := Resize(image.NewRGBA(image.Rect(0, 0, tt.w, tt.h)), tt.max).Bounds()
		if got.Dx() != tt.wantW || got.Dy() != tt.wantH {
			t.Errorf("Resize(%dx%d) = %dx%d, want %dx%d", tt.w, tt.h, got.Dx(), got.Dy(), tt.wantW, tt.wantH)
		}
	}
}

func TestOrient(t *testing.T) {
	red := color.RGBA{R: 255, A: 255}
	src := image.NewRGBA(image.Rect(0, 0, 3, 2))
	src.Set(0, 0, red)

	want := map[int]image.Point{
		1: {X: 0, Y: 0},
		2: {X: 2, Y: 0},
		3: {X: 2, Y: 1},
		4: {X: 0, Y: 1},
		5: {X: 0, Y: 0},
		6: {X: 1, Y: 0},
		7: {X: 1, Y: 2},
		8: {X: 0, Y: 2},
	}
	for orientation, p := range want {
		got := Orient(src, orientation)
		b := got.Bounds()
		swapped := orientation >= 5
		if (b.Dx() == 2 && b.Dy() == 3) != swapped {
			t.Errorf("orientation %d: size %dx%d", orientation, b.Dx(), b.Dy())
		}
		if got.At(p.X, p.Y) != red {
			t.Errorf("orientation %d: marker not at %v", orientation, p)
		}
	}
}

func TestGetCachesAndRejectsOtherFiles(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.jpg")
	testutil.WriteJPEG(t, src, 100)
	m := New(filepath.Join(dir, "thumbs"))

	p, err := m.Get(7, src, "jpg", store.KindImage, 6)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := jpeg.DecodeConfig(f)
	f.Close()
	if err != nil || cfg.Width != 48 || cfg.Height != 64 {
		t.Errorf("thumbnail = %dx%d, %v", cfg.Width, cfg.Height, err)
	}

	if err := os.Remove(src); err != nil {
		t.Fatal(err)
	}
	if again, err := m.Get(7, src, "jpg", store.KindImage, 6); err != nil || again != p {
		t.Errorf("cached Get = %q, %v", again, err)
	}
	m.Remove(7)
	if _, err := m.Get(7, src, "jpg", store.KindImage, 6); err == nil {
		t.Error("Get succeeded after the thumbnail was removed and the source deleted")
	}

	text := filepath.Join(dir, "notes.txt")
	testutil.WriteFile(t, text, "hello")
	if _, err := m.Get(8, text, "txt", store.KindOther, 1); err == nil {
		t.Error("thumbnail made for a text file")
	}
	if err := m.Clear(); err != nil {
		t.Fatal(err)
	}
}
