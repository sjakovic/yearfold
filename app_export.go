package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/sjakovic/yearfold/internal/store"
)

type exportFile struct {
	Path    string   `json:"path"`
	TakenAt int64    `json:"takenAt,omitempty"`
	Hash    string   `json:"hash,omitempty"`
	Tags    []string `json:"tags,omitempty"`
	Albums  []string `json:"albums,omitempty"`
}

// ExportJSON saves every file with its tags and albums to a JSON file chosen
// by the user and returns its path ("" when cancelled).
func (a *App) ExportJSON() (string, error) {
	lib, err := a.library()
	if err != nil {
		return "", err
	}
	out, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           a.text("Export library", "Извоз библиотеке"),
		DefaultFilename: "yearfold-export.json",
	})
	if err != nil || out == "" {
		return "", err
	}

	labels := func(q string) (map[int64][]string, error) {
		rows, err := lib.St.DB.Query(q)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		m := map[int64][]string{}
		for rows.Next() {
			var id int64
			var name string
			if err := rows.Scan(&id, &name); err != nil {
				return nil, err
			}
			m[id] = append(m[id], name)
		}
		return m, rows.Err()
	}
	tags, err := labels(`SELECT ft.file_id, t.name FROM file_tags ft JOIN tags t ON t.id = ft.tag_id ORDER BY t.name`)
	if err != nil {
		return "", err
	}
	albums, err := labels(`SELECT af.file_id, a.name FROM album_files af JOIN albums a ON a.id = af.album_id ORDER BY a.name`)
	if err != nil {
		return "", err
	}

	rows, err := lib.St.DB.Query(`SELECT id, rel_path, taken_at, taken_src, hash FROM files
		WHERE status = 'present' ORDER BY rel_path`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	files := []exportFile{}
	for rows.Next() {
		var id int64
		var src string
		var f exportFile
		if err := rows.Scan(&id, &f.Path, &f.TakenAt, &src, &f.Hash); err != nil {
			return "", err
		}
		if src == "" || src == store.SrcMtime {
			f.TakenAt = 0
		}
		f.Tags, f.Albums = tags[id], albums[id]
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(map[string]any{"root": lib.Root, "files": files}, "", "  ")
	if err != nil {
		return "", err
	}
	return out, os.WriteFile(out, data, 0o644)
}

// Reveal shows a file in the system file manager.
func (a *App) Reveal(id int64) error {
	lib, err := a.library()
	if err != nil {
		return err
	}
	f, err := lib.St.GetFile(id)
	if err != nil {
		return err
	}
	p := lib.Abs(f)
	switch goruntime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", p).Start()
	case "windows":
		return exec.Command("explorer", "/select,", p).Start()
	}
	return exec.Command("xdg-open", filepath.Dir(p)).Start()
}
