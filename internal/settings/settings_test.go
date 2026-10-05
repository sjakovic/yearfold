package settings

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	s := LoadFrom(path)
	if len(s.Recent) != 0 || s.Language != "" {
		t.Fatalf("fresh settings = %+v", s)
	}
	s.Language = "sr"
	s.AddRecent("/photos")
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	got := LoadFrom(path)
	if got.Language != "sr" || len(got.Recent) != 1 || got.Recent[0] != "/photos" {
		t.Errorf("loaded = %+v", got)
	}
}

func TestAddRecentMovesToFrontAndCaps(t *testing.T) {
	var s Settings
	for i := 0; i < maxRecent+3; i++ {
		s.AddRecent(fmt.Sprintf("/lib%d", i))
	}
	if len(s.Recent) != maxRecent || s.Recent[0] != fmt.Sprintf("/lib%d", maxRecent+2) {
		t.Fatalf("recent = %v", s.Recent)
	}
	again := s.Recent[3]
	s.AddRecent(again)
	if s.Recent[0] != again || len(s.Recent) != maxRecent {
		t.Errorf("recent = %v", s.Recent)
	}
	for _, r := range s.Recent[1:] {
		if r == again {
			t.Errorf("%s listed twice: %v", again, s.Recent)
		}
	}
}

func TestSaveWithoutPathIsNoOp(t *testing.T) {
	if err := (Settings{Language: "en"}).Save(); err != nil {
		t.Errorf("Save = %v", err)
	}
}
