package meta

import "testing"

func TestMatchSidecar(t *testing.T) {
	media := []string{
		"IMG_1234.jpg", "IMG_1234(1).jpg", "IMG_1234-edited.jpg", "VID_0001.mp4",
		"a_very_long_file_name_that_takeout_truncates_here.jpg", "other.png",
	}
	tests := []struct {
		json, want string
	}{
		{"IMG_1234.jpg.json", "IMG_1234.jpg"},
		{"IMG_1234.jpg.supplemental-metadata.json", "IMG_1234.jpg"},
		{"IMG_1234.jpg.supplemental-me.json", "IMG_1234.jpg"},
		{"IMG_1234.jpg.s.json", "IMG_1234.jpg"},
		{"IMG_1234.jpg(1).json", "IMG_1234(1).jpg"},
		{"IMG_1234.jpg.supplemental-metadata(1).json", "IMG_1234(1).jpg"},
		{"VID_0001.mp4.supplemental-metadata.json", "VID_0001.mp4"},
		{"a_very_long_file_name_that_takeout_truncates_h.json", "a_very_long_file_name_that_takeout_truncates_here.jpg"},
		{"metadata.json", ""},
		{"IMG_9999.jpg.json", ""},
	}
	for _, tt := range tests {
		got, ok := MatchSidecar(tt.json, media)
		if got != tt.want || ok != (tt.want != "") {
			t.Errorf("MatchSidecar(%q) = %q, %v; want %q", tt.json, got, ok, tt.want)
		}
	}
}

func TestParseTakeout(t *testing.T) {
	tk, ok := ParseTakeout([]byte(`{"title":"a.jpg","description":"more",
		"photoTakenTime":{"timestamp":"1560600000"},
		"geoData":{"latitude":0,"longitude":0},"geoDataExif":{"latitude":44.8,"longitude":20.4},
		"people":[{"name":"Ana"}]}`))
	if !ok || tk.TakenAt != 1560600000 || tk.Lat != 44.8 || tk.Description != "more" || len(tk.People) != 1 {
		t.Errorf("ParseTakeout = %+v, %v", tk, ok)
	}
	if _, ok := ParseTakeout([]byte(`{"albumData":{"title":"x"}}`)); ok {
		t.Error("album metadata parsed as a photo sidecar")
	}
}

func TestEditedOriginal(t *testing.T) {
	if got, ok := EditedOriginal("IMG_1-edited.jpg"); !ok || got != "IMG_1.jpg" {
		t.Errorf("EditedOriginal = %q, %v", got, ok)
	}
	if _, ok := EditedOriginal("IMG_1.jpg"); ok {
		t.Error("plain file reported as edited")
	}
}
