// Package app is the API the user interface calls.
package app

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/sjakovic/yearfold/internal/library"
	"github.com/sjakovic/yearfold/internal/scanner"
	"github.com/sjakovic/yearfold/internal/settings"
)

const sampleLimit = 300

var errNoLibrary = errors.New("no library is open")

type App struct {
	ctx     context.Context
	version string
	cfg     settings.Settings

	mu      sync.Mutex
	lib     *library.Library
	libCtx  context.Context
	cancel  context.CancelFunc
	pending *scanner.Changes

	seq     atomic.Int64
	procMu  sync.Mutex
	running bool
	again   bool
}

func New(version string) *App {
	return &App{version: version}
}

func Startup(a *App) func(context.Context) {
	return func(ctx context.Context) {
		a.ctx = ctx
		a.cfg = settings.Load()
	}
}

func Shutdown(a *App) func(context.Context) {
	return func(context.Context) { a.CloseLibrary() }
}

func CurrentLibrary(a *App) func() *library.Library {
	return a.current
}

func (a *App) current() *library.Library {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lib
}

func (a *App) library() (*library.Library, error) {
	if l := a.current(); l != nil {
		return l, nil
	}
	return nil, errNoLibrary
}

func (a *App) libraryContext() context.Context {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.libCtx == nil {
		return context.Background()
	}
	return a.libCtx
}

type Progress struct {
	Phase string `json:"phase"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
	Seq   int64  `json:"seq"`
}

func (a *App) progress(phase string, done, total int) {
	runtime.EventsEmit(a.ctx, "progress", Progress{Phase: phase, Done: done, Total: total, Seq: a.seq.Add(1)})
}

func (a *App) idle() {
	a.progress("", 0, 0)
}

func (a *App) changed() {
	runtime.EventsEmit(a.ctx, "changed")
}

func (a *App) text(en, sr string) string {
	if a.cfg.Language == "sr" {
		return sr
	}
	return en
}

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
			_ = lib.ExtractMeta(ctx, func(done, total int) { a.progress("meta", done, total) })
			a.changed()
			_ = lib.HashPending(ctx, func(done, total int) { a.progress("hash", done, total) })
		}
		a.idle()
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
