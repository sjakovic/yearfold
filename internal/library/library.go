// Package library opens a photo library and does the work on its files.
package library

import (
	"os"
	"path/filepath"

	"github.com/sjakovic/yearfold/internal/fileops"
	"github.com/sjakovic/yearfold/internal/legacy"
	"github.com/sjakovic/yearfold/internal/scanner"
	"github.com/sjakovic/yearfold/internal/store"
	"github.com/sjakovic/yearfold/internal/thumbs"
)

type Library struct {
	Root   string
	Dir    string
	St     *store.Store
	Thumbs *thumbs.Manager
	Ops    *fileops.Ops
}

func Open(root string) (*Library, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "open", Path: root, Err: os.ErrInvalid}
	}

	dir := filepath.Join(root, scanner.MetaDir)
	if err := legacy.AdoptDir(filepath.Join(root, legacy.LibraryDir), dir); err != nil {
		return nil, err
	}
	thumbDir := filepath.Join(dir, "thumbs")
	trashDir := filepath.Join(dir, "trash")
	for _, d := range []string{thumbDir, trashDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	st, err := store.Open(filepath.Join(dir, "library.db"))
	if err != nil {
		return nil, err
	}

	l := &Library{Root: root, Dir: dir, St: st, Thumbs: thumbs.New(thumbDir)}
	l.Ops = &fileops.Ops{Root: root, TrashDir: trashDir, St: st, OnDelete: l.Thumbs.Remove}
	return l, nil
}

func (l *Library) Close() error { return l.St.Close() }

func (l *Library) Abs(f store.File) string {
	if f.Status == store.StatusTrashed {
		return filepath.Join(l.Ops.TrashDir, f.TrashPath)
	}
	return filepath.Join(l.Root, filepath.FromSlash(f.RelPath))
}

func (l *Library) Scan(progress func(seen int)) (*scanner.Changes, error) {
	ch, err := scanner.Diff(l.Root, l.St, progress)
	if err != nil {
		return nil, err
	}
	return ch, l.Apply(ch)
}

func (l *Library) Apply(ch *scanner.Changes) error {
	return scanner.Apply(l.St, ch, l.Thumbs.Remove)
}
