package app

import (
	"context"
	"errors"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/sjakovic/yearfold/internal/library"
)

type State struct {
	Root     string   `json:"root"`
	Recent   []string `json:"recent"`
	Language string   `json:"language"`
	Version  string   `json:"version"`
}

func (a *App) GetState() State {
	st := State{Recent: a.cfg.Recent, Language: a.cfg.Language, Version: a.version}
	if st.Language == "" {
		st.Language = "en"
	}
	if st.Recent == nil {
		st.Recent = []string{}
	}
	if l := a.current(); l != nil {
		st.Root = l.Root
	}
	return st
}

func (a *App) SetLanguage(lang string) (State, error) {
	if lang != "en" && lang != "sr" {
		return a.GetState(), errors.New("unsupported language: " + lang)
	}
	a.cfg.Language = lang
	return a.GetState(), a.cfg.Save()
}

func (a *App) PickFolder() (State, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: a.text("Choose a folder with photos", "Изабери фолдер са сликама"),
	})
	if err != nil || dir == "" {
		return a.GetState(), err
	}
	return a.OpenLibrary(dir)
}

func (a *App) OpenLibrary(root string) (State, error) {
	a.CloseLibrary()
	lib, err := library.Open(root)
	if err != nil {
		return a.GetState(), err
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.lib, a.libCtx, a.cancel = lib, ctx, cancel
	a.mu.Unlock()

	a.cfg.AddRecent(lib.Root)
	_ = a.cfg.Save()

	stats, err := lib.St.Stats()
	if err != nil {
		return a.GetState(), err
	}
	if stats.Files+stats.Trashed+stats.Missing == 0 {
		go a.firstScan(lib)
	} else {
		go a.process()
	}
	return a.GetState(), nil
}

func (a *App) firstScan(lib *library.Library) {
	_, err := lib.Scan(func(seen int) { a.progress("scan", seen, 0) })
	a.idle()
	a.changed()
	if err == nil {
		a.process()
	}
}

func (a *App) CloseLibrary() {
	a.mu.Lock()
	lib, cancel := a.lib, a.cancel
	a.lib, a.libCtx, a.cancel, a.pending = nil, nil, nil, nil
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if lib != nil {
		_ = lib.Close()
	}
}
