// Package scanner walks the library root and reconciles it with the index.
package scanner

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sjakovic/yearfold/internal/store"
)

// MetaDir is the hidden folder inside the root that holds the library data.
const MetaDir = ".yearfold"

type Entry struct {
	RelPath string `json:"relPath"`
	Size    int64  `json:"size"`
	Mtime   int64  `json:"mtime"`
}

type Known struct {
	ID int64 `json:"id"`
	Entry
}

type Moved struct {
	ID   int64  `json:"id"`
	From string `json:"from"`
	To   Entry  `json:"to"`
}

// Changes is the difference between the disk and the index.
type Changes struct {
	New      []Entry `json:"new"`
	Modified []Known `json:"modified"`
	Missing  []Known `json:"missing"`
	Moved    []Moved `json:"moved"`
	// Revived are files previously marked missing that are back at their old path.
	Revived []Known `json:"revived"`
}

func (c *Changes) Empty() bool {
	return len(c.New)+len(c.Modified)+len(c.Missing)+len(c.Moved)+len(c.Revived) == 0
}

type row struct {
	id          int64
	size, mtime int64
	hash        string
	status      string
}

// Diff walks root and compares it with the index without changing anything.
// progress is called periodically with the number of files seen so far.
func Diff(root string, st *store.Store, progress func(seen int)) (*Changes, error) {
	known := map[string]row{}
	// Rows that are gone from disk and could be the source of an external move.
	var gone []struct {
		path string
		row
	}
	rows, err := st.DB.Query(`SELECT id, rel_path, size, mtime, hash, status FROM files
		WHERE status IN ('present', 'missing') ORDER BY status`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var r row
		var p string
		if err := rows.Scan(&r.id, &p, &r.size, &r.mtime, &r.hash, &r.status); err != nil {
			rows.Close()
			return nil, err
		}
		// 'missing' sorts before 'present', so a present row wins a path clash.
		known[p] = r
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	ch := &Changes{}
	seen := map[string]bool{}
	n := 0
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
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
		e := Entry{RelPath: filepath.ToSlash(rel), Size: info.Size(), Mtime: info.ModTime().Unix()}
		seen[e.RelPath] = true
		r, ok := known[e.RelPath]
		switch {
		case !ok:
			ch.New = append(ch.New, e)
		case r.status == store.StatusMissing:
			ch.Revived = append(ch.Revived, Known{ID: r.id, Entry: e})
		case r.size != e.Size || r.mtime != e.Mtime:
			ch.Modified = append(ch.Modified, Known{ID: r.id, Entry: e})
		}
		if n++; progress != nil && n%500 == 0 {
			progress(n)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if progress != nil {
		progress(n)
	}

	for p, r := range known {
		if seen[p] {
			continue
		}
		if r.status == store.StatusPresent {
			ch.Missing = append(ch.Missing, Known{ID: r.id, Entry: Entry{RelPath: p, Size: r.size, Mtime: r.mtime}})
		}
		if r.hash != "" {
			gone = append(gone, struct {
				path string
				row
			}{p, r})
		}
	}

	// A new file with the same content as a vanished one was moved outside the app.
	if len(gone) > 0 && len(ch.New) > 0 {
		bySize := map[int64][]int{}
		for i, g := range gone {
			bySize[g.size] = append(bySize[g.size], i)
		}
		used := map[int]bool{}
		kept := ch.New[:0]
		for _, e := range ch.New {
			match := -1
			if cands := bySize[e.Size]; len(cands) > 0 {
				if h, err := HashFile(filepath.Join(root, filepath.FromSlash(e.RelPath))); err == nil {
					for _, i := range cands {
						if !used[i] && gone[i].hash == h {
							match = i
							break
						}
					}
				}
			}
			if match < 0 {
				kept = append(kept, e)
				continue
			}
			used[match] = true
			ch.Moved = append(ch.Moved, Moved{ID: gone[match].id, From: gone[match].path, To: e})
		}
		ch.New = kept
		movedIDs := map[int64]bool{}
		for _, m := range ch.Moved {
			movedIDs[m.ID] = true
		}
		missing := ch.Missing[:0]
		for _, m := range ch.Missing {
			if !movedIDs[m.ID] {
				missing = append(missing, m)
			}
		}
		ch.Missing = missing
	}
	return ch, nil
}

// Apply writes the changes to the index. onInvalidate is called for files
// whose content changed so cached thumbnails can be dropped.
func Apply(st *store.Store, ch *Changes, onInvalidate func(id int64)) error {
	err := st.InTx(func(tx *sql.Tx) error {
		// Missing first so their paths are free for new or moved files.
		for _, m := range ch.Missing {
			if _, err := tx.Exec(`UPDATE files SET status = 'missing' WHERE id = ?`, m.ID); err != nil {
				return err
			}
		}
		for _, m := range ch.Moved {
			if _, err := tx.Exec(`UPDATE files SET status = 'missing' WHERE id = ?`, m.ID); err != nil {
				return err
			}
		}
		for _, m := range ch.Moved {
			if err := store.SetPath(tx, m.ID, m.To.RelPath); err != nil {
				return err
			}
			if _, err := tx.Exec(`UPDATE files SET status = 'present', size = ?, mtime = ? WHERE id = ?`,
				m.To.Size, m.To.Mtime, m.ID); err != nil {
				return err
			}
		}
		for _, k := range append(append([]Known{}, ch.Modified...), ch.Revived...) {
			if _, err := tx.Exec(`UPDATE files SET status = 'present', size = ?, mtime = ?, hash = '', meta_done = 0
				WHERE id = ?`, k.Size, k.Mtime, k.ID); err != nil {
				return err
			}
		}
		for _, e := range ch.New {
			if _, err := store.InsertFile(tx, e.RelPath, e.Size, e.Mtime); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if onInvalidate != nil {
		for _, k := range ch.Modified {
			onInvalidate(k.ID)
		}
		for _, k := range ch.Revived {
			onInvalidate(k.ID)
		}
	}
	return nil
}

// HashFile returns the hex SHA-256 of a file's content.
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
