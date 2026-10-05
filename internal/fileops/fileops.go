// Package fileops moves, trashes and restores files on disk and keeps the
// index and the undo journal in step.
package fileops

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sjakovic/yearfold/internal/scanner"
	"github.com/sjakovic/yearfold/internal/store"
)

type Ops struct {
	Root     string
	TrashDir string
	St       *store.Store
	OnDelete func(id int64)
}

type Target struct {
	ID      int64
	DestDir string
}

func (o *Ops) abs(rel string) string { return filepath.Join(o.Root, filepath.FromSlash(rel)) }

func CleanDir(dir string) (string, error) {
	dir = strings.ReplaceAll(strings.TrimSpace(dir), `\`, "/")
	for _, seg := range strings.Split(dir, "/") {
		if seg == ".." {
			return "", errors.New("destination must stay inside the library folder")
		}
	}
	clean := strings.Trim(path.Clean("/"+dir), "/")
	if clean == scanner.MetaDir || strings.HasPrefix(clean, scanner.MetaDir+"/") {
		return "", errors.New("destination is reserved for library data")
	}
	return clean, nil
}

func (o *Ops) freePath(rel string) (string, error) {
	dir, name, _ := store.SplitPath(rel)
	stem, ext := name, ""
	if i := strings.LastIndex(name, "."); i > 0 {
		stem, ext = name[:i], name[i:]
	}
	for n := 0; n < 10000; n++ {
		candidate := name
		if n > 0 {
			candidate = fmt.Sprintf("%s_%d%s", stem, n, ext)
		}
		candidateRel := path.Join(dir, candidate)
		taken, err := o.St.PathTaken(candidateRel)
		if err != nil {
			return "", err
		}
		if taken {
			continue
		}
		if _, err := os.Lstat(o.abs(candidateRel)); errors.Is(err, os.ErrNotExist) {
			return candidateRel, nil
		}
	}
	return "", fmt.Errorf("no free name for %s", rel)
}

func (o *Ops) relocate(f store.File, newRel string, record func() error) error {
	dst := o.abs(newRel)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(o.abs(f.RelPath), dst); err != nil {
		return err
	}
	if err := record(); err != nil {
		_ = os.Rename(dst, o.abs(f.RelPath))
		return err
	}
	o.pruneEmpty(f.Dir)
	return nil
}

func (o *Ops) pruneEmpty(dir string) {
	for dir != "" && dir != "." {
		if os.Remove(o.abs(dir)) != nil {
			return
		}
		dir = path.Dir(dir)
	}
}

func (o *Ops) withSidecars(id int64, status string) ([]store.File, error) {
	f, err := o.St.GetFile(id)
	if err != nil {
		return nil, err
	}
	if f.Status != status {
		return nil, nil
	}
	files := []store.File{f}
	if status == store.StatusPresent {
		sidecars, err := o.St.Sidecars(id)
		if err != nil {
			return nil, err
		}
		files = append(files, sidecars...)
	}
	return files, nil
}

func (o *Ops) Move(targets []Target) (int, error) {
	batch, err := o.St.NextBatch()
	if err != nil {
		return 0, err
	}
	done := map[int64]bool{}
	moved := 0
	for _, t := range targets {
		dest, err := CleanDir(t.DestDir)
		if err != nil {
			return moved, err
		}
		files, err := o.withSidecars(t.ID, store.StatusPresent)
		if err != nil {
			return moved, err
		}
		for _, f := range files {
			if done[f.ID] || f.Dir == dest {
				continue
			}
			done[f.ID] = true
			newRel, err := o.freePath(path.Join(dest, f.Name))
			if err != nil {
				return moved, err
			}
			err = o.relocate(f, newRel, func() error {
				return o.St.RecordMove(batch, f.ID, f.RelPath, newRel)
			})
			if err != nil {
				return moved, fmt.Errorf("%s: %w", f.RelPath, err)
			}
			moved++
		}
	}
	return moved, nil
}

func (o *Ops) Trash(ids []int64) (int, error) {
	batch, err := o.St.NextBatch()
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(o.TrashDir, 0o755); err != nil {
		return 0, err
	}
	n := 0
	for _, id := range ids {
		files, err := o.withSidecars(id, store.StatusPresent)
		if err != nil {
			return n, err
		}
		for _, f := range files {
			if err := o.trash(f, batch); err != nil {
				return n, fmt.Errorf("%s: %w", f.RelPath, err)
			}
			n++
		}
	}
	return n, nil
}

func (o *Ops) trash(f store.File, batch int64) error {
	trashName := fmt.Sprintf("%d_%s", f.ID, f.Name)
	dst := filepath.Join(o.TrashDir, trashName)
	if err := os.Rename(o.abs(f.RelPath), dst); err != nil {
		return err
	}
	if err := o.St.RecordTrash(batch, f.ID, f.RelPath, trashName); err != nil {
		_ = os.Rename(dst, o.abs(f.RelPath))
		return err
	}
	return nil
}

func (o *Ops) Restore(ids []int64) (int, error) {
	n := 0
	for _, id := range ids {
		sidecars, err := o.St.TrashedSidecars(id)
		if err != nil {
			return n, err
		}
		for _, fileID := range append([]int64{id}, sidecars...) {
			f, err := o.St.GetFile(fileID)
			if err != nil {
				return n, err
			}
			if f.Status != store.StatusTrashed {
				continue
			}
			if err := o.restore(f); err != nil {
				return n, fmt.Errorf("%s: %w", f.RelPath, err)
			}
			n++
		}
	}
	return n, nil
}

func (o *Ops) restore(f store.File) error {
	rel, err := o.freePath(f.RelPath)
	if err != nil {
		return err
	}
	src := filepath.Join(o.TrashDir, f.TrashPath)
	dst := o.abs(rel)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	if err := o.St.RecordRestore(f.ID, rel); err != nil {
		_ = os.Rename(dst, src)
		return err
	}
	return nil
}

func (o *Ops) EmptyTrash() (int, error) {
	trashed, err := o.St.Trashed()
	if err != nil {
		return 0, err
	}
	n := 0
	for _, f := range trashed {
		if f.TrashPath != "" {
			err := os.Remove(filepath.Join(o.TrashDir, f.TrashPath))
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return n, err
			}
		}
		if err := o.St.DeleteFile(f.ID); err != nil {
			return n, err
		}
		if o.OnDelete != nil {
			o.OnDelete(f.ID)
		}
		n++
	}
	return n, nil
}

func (o *Ops) Undo() (int, error) {
	ops, err := o.St.LastBatch()
	if err != nil {
		return 0, err
	}
	n := 0
	for _, op := range ops {
		undone, err := o.undo(op)
		if err != nil {
			return n, err
		}
		if undone {
			n++
		}
		if err := o.St.MarkUndone(op.ID); err != nil {
			return n, err
		}
	}
	return n, nil
}

func (o *Ops) undo(op store.Op) (bool, error) {
	f, err := o.St.GetFile(op.FileID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	switch {
	case op.Type == store.OpMove && f.Status == store.StatusPresent && f.RelPath == op.To:
		back, err := o.freePath(op.From)
		if err != nil {
			return false, err
		}
		err = o.relocate(f, back, func() error { return o.St.Relocate(f.ID, back) })
		if err != nil {
			return false, fmt.Errorf("%s: %w", f.RelPath, err)
		}
		return true, nil
	case op.Type == store.OpTrash && f.Status == store.StatusTrashed:
		if err := o.restore(f); err != nil {
			return false, fmt.Errorf("%s: %w", f.RelPath, err)
		}
		return true, nil
	}
	return false, nil
}
