package main

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/sjakovic/yearfold/internal/library"
	"github.com/sjakovic/yearfold/internal/scanner"
)

// sampleLimit caps how many paths of each kind a report sends to the UI.
const sampleLimit = 300

var errNoLibrary = errors.New("no library is open")

// App is the API exposed to the frontend.
type App struct {
	ctx     context.Context
	version string
	cfg     library.Config

	mu      sync.Mutex
	lib     *library.Library
	libCtx  context.Context
	cancel  context.CancelFunc
	pending *scanner.Changes // result of the last CheckChanges, waiting for ApplyChanges

	seq     atomic.Int64
	procMu  sync.Mutex
	running bool
	again   bool
}

func NewApp(version string) *App {
	return &App{version: version}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.cfg = library.LoadConfig()
}

func (a *App) shutdown(context.Context) {
	a.CloseLibrary()
}

func (a *App) current() *library.Library {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lib
}

// libraryContext is cancelled when the open library is closed.
func (a *App) libraryContext() context.Context {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.libCtx == nil {
		return context.Background()
	}
	return a.libCtx
}

func (a *App) library() (*library.Library, error) {
	if l := a.current(); l != nil {
		return l, nil
	}
	return nil, errNoLibrary
}

type Progress struct {
	Phase string `json:"phase"` // scan, meta, hash or "" when idle
	Done  int    `json:"done"`
	Total int    `json:"total"`
	// Seq orders events; the runtime may deliver them out of order.
	Seq int64 `json:"seq"`
}

func (a *App) progress(phase string, done, total int) {
	runtime.EventsEmit(a.ctx, "progress", Progress{Phase: phase, Done: done, Total: total, Seq: a.seq.Add(1)})
}

func (a *App) changed() {
	runtime.EventsEmit(a.ctx, "changed")
}

type AppState struct {
	Root     string   `json:"root"`
	Recent   []string `json:"recent"`
	Language string   `json:"language"`
	Version  string   `json:"version"`
}

// text picks the string for the current UI language.
func (a *App) text(en, sr string) string {
	if a.cfg.Language == "sr" {
		return sr
	}
	return en
}

// SetLanguage switches the UI language: "en" or "sr" (Serbian Cyrillic).
func (a *App) SetLanguage(lang string) (AppState, error) {
	if lang != "en" && lang != "sr" {
		return a.GetState(), errors.New("unsupported language: " + lang)
	}
	a.cfg.Language = lang
	return a.GetState(), a.cfg.Save()
}

func (a *App) GetState() AppState {
	st := AppState{Recent: a.cfg.Recent, Language: a.cfg.Language, Version: a.version}
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

// PickFolder shows the native folder dialog and opens the chosen library.
// It returns an empty root when the dialog is cancelled.
func (a *App) PickFolder() (AppState, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: a.text("Choose a folder with photos", "Изабери фолдер са сликама"),
	})
	if err != nil || dir == "" {
		return a.GetState(), err
	}
	return a.OpenLibrary(dir)
}

// OpenLibrary opens the library at root and brings its index up to date
// when it is new.
func (a *App) OpenLibrary(root string) (AppState, error) {
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
	a.cfg.Save()

	stats, err := lib.St.Stats()
	if err != nil {
		return a.GetState(), err
	}
	if stats.Files+stats.Trashed+stats.Missing == 0 {
		// A brand new library: index everything without asking.
		go func() {
			_, err := lib.Scan(func(seen int) { a.progress("scan", seen, 0) })
			a.progress("", 0, 0)
			a.changed()
			if err == nil {
				a.process()
			}
		}()
	} else {
		go a.process()
	}
	return a.GetState(), nil
}

func (a *App) CloseLibrary() {
	a.mu.Lock()
	lib, cancel := a.lib, a.cancel
	a.lib, a.cancel, a.pending = nil, nil, nil
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if lib != nil {
		lib.Close()
	}
}

// process runs the background pipeline for the open library: pair sidecars,
// read metadata, hash. A call made while it is running queues one more pass.
func (a *App) process() {
	a.procMu.Lock()
	if a.running {
		a.again = true
		a.procMu.Unlock()
		return
	}
	a.running = true
	a.procMu.Unlock()

	for {
		a.mu.Lock()
		lib, ctx := a.lib, a.libCtx
		a.mu.Unlock()
		if lib != nil && lib.LinkSidecars() == nil {
			lib.ExtractMeta(ctx, func(done, total int) { a.progress("meta", done, total) })
			a.changed()
			lib.HashPending(ctx, func(done, total int) { a.progress("hash", done, total) })
		}
		a.progress("", 0, 0)
		a.changed()

		a.procMu.Lock()
		if !a.again {
			a.running = false
			a.procMu.Unlock()
			return
		}
		a.again = false
		a.procMu.Unlock()
	}
}
