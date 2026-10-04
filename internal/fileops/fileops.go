// Package fileops performs the operations that touch files on disk: moving
// inside the root, trashing, restoring and undoing. Every change is recorded
// in the ops journal.
package fileops

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/sjakovic/yearfold/internal/scanner"
	"github.com/sjakovic/yearfold/internal/store"
)

const (
	opMove  = "move"
	opTrash = "trash"
)

type Ops struct {
	Root     string
	TrashDir string
	St       *store.Store
	// OnDelete is called for files removed for good.
	OnDelete func(id int64)
}

// Target asks for a file to be moved into DestDir (relative to the root).
type Target struct {
	ID      int64
	DestDir string
}

func (o *Ops) abs(rel string) string { return filepath.Join(o.Root, filepath.FromSlash(rel)) }

// CleanDir normalises a destination directory and rejects anything outside
// the root or inside the library's own data folder.
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

func (o *Ops) nextBatch() (int64, error) {
	var b int64
	err := o.St.DB.QueryRow(`SELECT COALESCE(MAX(batch), 0) + 1 FROM ops`).Scan(&b)
	return b, err
}

// freePath returns rel, or rel with a _N suffix when the name is taken.
func (o *Ops) freePath(rel string) (string, error) {
	dir, name, _ := store.SplitPath(rel)
	stem, ext := name, ""
	if i := strings.LastIndex(name, "."); i > 0 {
		stem, ext = name[:i], name[i:]
	}
	for n := 0; n < 10000; n++ {
		cand := name
		if n > 0 {
			cand = fmt.Sprintf("%s_%d%s", stem, n, ext)
		}
		candRel := path.Join(dir, cand)
		taken, err := o.St.PathTaken(candRel)
		if err != nil {
			return "", err
		}
		if !taken {
			if _, err := os.Lstat(o.abs(candRel)); errors.Is(err, os.ErrNotExist) {
				return candRel, nil
			}
		}
	}
	return "", fmt.Errorf("no free name for %s", rel)
}

// relocate renames a present file to newRel on disk and in the index.
func (o *Ops) relocate(f store.File, newRel string, batch int64, journal bool) error {
	dst := o.abs(newRel)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(o.abs(f.RelPath), dst); err != nil {
		return err
	}
	err := o.St.InTx(func(tx *sql.Tx) error {
		if err := store.SetPath(tx, f.ID, newRel); err != nil {
			return err
		}
		if !journal {
			return nil
		}
		_, err := tx.Exec(`INSERT INTO ops (batch, type, file_id, from_path, to_path, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
			batch, opMove, f.ID, f.RelPath, newRel, time.Now().Unix())
		return err
	})
	if err != nil {
		os.Rename(dst, o.abs(f.RelPath))
		return err
	}
	o.pruneEmpty(f.Dir)
	return nil
}

// pruneEmpty removes dir and its parents while they are empty.
func (o *Ops) pruneEmpty(dir string) {
	for dir != "" && dir != "." {
		if os.Remove(o.abs(dir)) != nil {
			return
		}
		dir = path.Dir(dir)
	}
}

// withSidecars expands ids so that sidecars travel with their media file.
func (o *Ops) withSidecars(id int64, status string) ([]store.File, error) {
	f, err := o.St.GetFile(id)
	if err != nil {
		return nil, err
	}
	if f.Status != status {
		return nil, nil
	}
	out := []store.File{f}
	if status == store.StatusPresent {
		sc, err := o.St.Sidecars(id)
		if err != nil {
			return nil, err
		}
		out = append(out, sc...)
	}
	return out, nil
}

// Move moves files (and their sidecars) and returns how many files moved.
func (o *Ops) Move(targets []Target) (int, error) {
	batch, err := o.nextBatch()
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
			if err := o.relocate(f, newRel, batch, true); err != nil {
				return moved, fmt.Errorf("%s: %w", f.RelPath, err)
			}
			moved++
		}
	}
	return moved, nil
}

// Trash moves files (and their sidecars) into the library trash.
func (o *Ops) Trash(ids []int64) (int, error) {
	batch, err := o.nextBatch()
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
			trashName := fmt.Sprintf("%d_%s", f.ID, f.Name)
			dst := filepath.Join(o.TrashDir, trashName)
			if err := os.Rename(o.abs(f.RelPath), dst); err != nil {
				return n, fmt.Errorf("%s: %w", f.RelPath, err)
			}
			err := o.St.InTx(func(tx *sql.Tx) error {
				if _, err := tx.Exec(`UPDATE files SET status = 'trashed', trash_path = ? WHERE id = ?`, trashName, f.ID); err != nil {
					return err
				}
				_, err := tx.Exec(`INSERT INTO ops (batch, type, file_id, from_path, to_path, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
					batch, opTrash, f.ID, f.RelPath, trashName, time.Now().Unix())
				return err
			})
			if err != nil {
				os.Rename(dst, o.abs(f.RelPath))
				return n, err
			}
			n++
		}
	}
	return n, nil
}

// Restore brings trashed files back to where they were.
func (o *Ops) Restore(ids []int64) (int, error) {
	n := 0
	for _, id := range ids {
		f, err := o.St.GetFile(id)
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
		// Sidecars trashed together with the media file come back with it.
		rows, err := o.St.DB.Query(`SELECT id FROM files WHERE sidecar_of = ? AND status = 'trashed'`, id)
		if err != nil {
			return n, err
		}
		var scIDs []int64
		for rows.Next() {
			var sid int64
			if err := rows.Scan(&sid); err == nil {
				scIDs = append(scIDs, sid)
			}
		}
		rows.Close()
		for _, sid := range scIDs {
			sc, err := o.St.GetFile(sid)
			if err != nil {
				return n, err
			}
			if err := o.restore(sc); err != nil {
				return n, fmt.Errorf("%s: %w", sc.RelPath, err)
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
	err = o.St.InTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`UPDATE files SET status = 'present', trash_path = '' WHERE id = ?`, f.ID); err != nil {
			return err
		}
		return store.SetPath(tx, f.ID, rel)
	})
	if err != nil {
		os.Rename(dst, src)
	}
	return err
}

// EmptyTrash permanently deletes every trashed file.
func (o *Ops) EmptyTrash() (int, error) {
	rows, err := o.St.DB.Query(`SELECT id, trash_path FROM files WHERE status = 'trashed'`)
	if err != nil {
		return 0, err
	}
	type item struct {
		id   int64
		path string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.path); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, it)
	}
	rows.Close()
	n := 0
	for _, it := range items {
		if it.path != "" {
			if err := os.Remove(filepath.Join(o.TrashDir, it.path)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return n, err
			}
		}
		if _, err := o.St.DB.Exec(`DELETE FROM files WHERE id = ?`, it.id); err != nil {
			return n, err
		}
		if o.OnDelete != nil {
			o.OnDelete(it.id)
		}
		n++
	}
	return n, nil
}

// Undo reverts the most recent move or trash batch and returns how many
// files were put back. It returns 0 when there is nothing to undo.
func (o *Ops) Undo() (int, error) {
	var batch int64
	err := o.St.DB.QueryRow(`SELECT batch FROM ops WHERE undone = 0 ORDER BY id DESC LIMIT 1`).Scan(&batch)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	rows, err := o.St.DB.Query(`SELECT id, type, file_id, from_path, to_path FROM ops
		WHERE batch = ? AND undone = 0 ORDER BY id DESC`, batch)
	if err != nil {
		return 0, err
	}
	type op struct {
		id, fileID    int64
		typ, from, to string
	}
	var ops []op
	for rows.Next() {
		var p op
		if err := rows.Scan(&p.id, &p.typ, &p.fileID, &p.from, &p.to); err != nil {
			rows.Close()
			return 0, err
		}
		ops = append(ops, p)
	}
	rows.Close()

	n := 0
	for _, p := range ops {
		f, err := o.St.GetFile(p.fileID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			// Deleted for good since; nothing to put back.
		case err != nil:
			return n, err
		case p.typ == opMove && f.Status == store.StatusPresent && f.RelPath == p.to:
			back, err := o.freePath(p.from)
			if err != nil {
				return n, err
			}
			if err := o.relocate(f, back, 0, false); err != nil {
				return n, fmt.Errorf("%s: %w", f.RelPath, err)
			}
			n++
		case p.typ == opTrash && f.Status == store.StatusTrashed:
			if err := o.restore(f); err != nil {
				return n, fmt.Errorf("%s: %w", f.RelPath, err)
			}
			n++
		}
		if _, err := o.St.DB.Exec(`UPDATE ops SET undone = 1 WHERE id = ?`, p.id); err != nil {
			return n, err
		}
	}
	return n, nil
}
