// Package scanner compares the files on disk with the index.
package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sjakovic/yearfold/internal/store"
)

const MetaDir = ".yearfold"

type Moved struct {
	ID   int64
	From string
	To   store.FileStat
}

type Changes struct {
	New      []store.FileStat
	Modified []store.KnownStat
	Missing  []store.KnownStat
	Moved    []Moved
	Revived  []store.KnownStat
}

func (c *Changes) Empty() bool {
	return len(c.New)+len(c.Modified)+len(c.Missing)+len(c.Moved)+len(c.Revived) == 0
}

func Diff(root string, st *store.Store, progress func(seen int)) (*Changes, error) {
	indexed, err := st.IndexedFiles()
	if err != nil {
		return nil, err
	}
	known := make(map[string]store.Indexed, len(indexed))
	for _, f := range indexed {
		known[f.RelPath] = f
	}

	ch := &Changes{}
	seen := map[string]bool{}
	err = walk(root, func(e store.FileStat) {
		seen[e.RelPath] = true
		f, ok := known[e.RelPath]
		switch {
		case !ok:
			ch.New = append(ch.New, e)
		case f.Status == store.StatusMissing:
			ch.Revived = append(ch.Revived, store.KnownStat{ID: f.ID, FileStat: e})
		case f.Size != e.Size || f.Mtime != e.Mtime:
			ch.Modified = append(ch.Modified, store.KnownStat{ID: f.ID, FileStat: e})
		}
		if progress != nil && len(seen)%500 == 0 {
			progress(len(seen))
		}
	})
	if err != nil {
		return nil, err
	}
	if progress != nil {
		progress(len(seen))
	}

	var gone []store.Indexed
	for p, f := range known {
		if seen[p] {
			continue
		}
		if f.Status == store.StatusPresent {
			ch.Missing = append(ch.Missing, store.KnownStat{
				ID:       f.ID,
				FileStat: store.FileStat{RelPath: p, Size: f.Size, Mtime: f.Mtime},
			})
		}
		if f.Hash != "" {
			gone = append(gone, f)
		}
	}
	detectMoves(root, ch, gone)
	return ch, nil
}

func walk(root string, visit func(store.FileStat)) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if d.Name() == MetaDir {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || d.Name() == ".DS_Store" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		visit(store.FileStat{RelPath: filepath.ToSlash(rel), Size: info.Size(), Mtime: info.ModTime().Unix()})
		return nil
	})
}

func detectMoves(root string, ch *Changes, gone []store.Indexed) {
	if len(gone) == 0 || len(ch.New) == 0 {
		return
	}
	bySize := map[int64][]int{}
	for i, g := range gone {
		bySize[g.Size] = append(bySize[g.Size], i)
	}
	used := map[int]bool{}
	movedIDs := map[int64]bool{}
	fresh := ch.New[:0]
	for _, e := range ch.New {
		match := -1
		if candidates := bySize[e.Size]; len(candidates) > 0 {
			if h, err := HashFile(filepath.Join(root, filepath.FromSlash(e.RelPath))); err == nil {
				for _, i := range candidates {
					if !used[i] && gone[i].Hash == h {
						match = i
						break
					}
				}
			}
		}
		if match < 0 {
			fresh = append(fresh, e)
			continue
		}
		used[match] = true
		movedIDs[gone[match].ID] = true
		ch.Moved = append(ch.Moved, Moved{ID: gone[match].ID, From: gone[match].RelPath, To: e})
	}
	ch.New = fresh

	missing := ch.Missing[:0]
	for _, m := range ch.Missing {
		if !movedIDs[m.ID] {
			missing = append(missing, m)
		}
	}
	ch.Missing = missing
}

func Apply(st *store.Store, ch *Changes, onInvalidate func(id int64)) error {
	update := store.ScanUpdate{New: ch.New}
	for _, m := range ch.Missing {
		update.Missing = append(update.Missing, m.ID)
	}
	for _, m := range ch.Moved {
		update.Moved = append(update.Moved, store.KnownStat{ID: m.ID, FileStat: m.To})
	}
	update.Changed = append(append(update.Changed, ch.Modified...), ch.Revived...)
	if err := st.ApplyScan(update); err != nil {
		return err
	}
	if onInvalidate != nil {
		for _, c := range update.Changed {
			onInvalidate(c.ID)
		}
	}
	return nil
}

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
