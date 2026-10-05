package app

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

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
	files, err := lib.St.Export()
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(map[string]any{"root": lib.Root, "files": files}, "", "  ")
	if err != nil {
		return "", err
	}
	return out, os.WriteFile(out, data, 0o644)
}

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
