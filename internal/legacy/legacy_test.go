package legacy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAdoptDir(t *testing.T) {
	base := t.TempDir()
	oldPath := filepath.Join(base, "old")
	newPath := filepath.Join(base, "new")

	if err := AdoptDir(oldPath, newPath); err != nil {
		t.Fatalf("nothing to adopt: %v", err)
	}
	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Fatal("new folder appeared from nowhere")
	}

	if err := os.MkdirAll(oldPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldPath, "data"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AdoptDir(oldPath, newPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(newPath, "data")); err != nil {
		t.Errorf("data not carried over: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("old folder still there")
	}

	if err := os.MkdirAll(oldPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := AdoptDir(oldPath, newPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Error("old folder removed although the new one already existed")
	}
}
