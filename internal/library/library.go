// Package library ties together the on-disk layout of a library
// (<root>/.yearfold) and the background processing of its files.
package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/sjakovic/yearfold/internal/fileops"
	"github.com/sjakovic/yearfold/internal/meta"
	"github.com/sjakovic/yearfold/internal/scanner"
	"github.com/sjakovic/yearfold/internal/store"
	"github.com/sjakovic/yearfold/internal/thumbs"
)

type Library struct {
	Root   string
	Dir    string // <root>/.yearfold
	St     *store.Store
	Thumbs *thumbs.Manager
	Ops    *fileops.Ops
}

// Open opens the library rooted at root, creating its data folder if needed.
func Open(root string) (*Library, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if st, err := os.Stat(root); err != nil {
		return nil, err
	} else if !st.IsDir() {
		return nil, &os.PathError{Op: "open", Path: root, Err: os.ErrInvalid}
	}
	dir := filepath.Join(root, scanner.MetaDir)
	if err := adoptLegacyLibrary(root, dir); err != nil {
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

// Abs returns the absolute path of a file, wherever its status puts it.
func (l *Library) Abs(f store.File) string {
	if f.Status == store.StatusTrashed {
		return filepath.Join(l.Ops.TrashDir, f.TrashPath)
	}
	return filepath.Join(l.Root, filepath.FromSlash(f.RelPath))
}

// Scan reconciles the index with the disk right away.
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

// LinkSidecars pairs Takeout JSON files with the media they describe.
func (l *Library) LinkSidecars() error {
	rows, err := l.St.DB.Query(`SELECT id, dir, name FROM files
		WHERE status = 'present' AND ext = 'json' AND sidecar_of IS NULL ORDER BY dir`)
	if err != nil {
		return err
	}
	type js struct {
		id   int64
		name string
	}
	byDir := map[string][]js{}
	for rows.Next() {
		var j js
		var dir string
		if err := rows.Scan(&j.id, &dir, &j.name); err != nil {
			rows.Close()
			return err
		}
		byDir[dir] = append(byDir[dir], j)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for dir, jsons := range byDir {
		mrows, err := l.St.DB.Query(`SELECT id, name FROM files
			WHERE dir = ? AND status = 'present' AND kind IN ('image', 'video')`, dir)
		if err != nil {
			return err
		}
		ids := map[string]int64{}
		var names []string
		for mrows.Next() {
			var id int64
			var name string
			if err := mrows.Scan(&id, &name); err != nil {
				mrows.Close()
				return err
			}
			ids[name] = id
			names = append(names, name)
		}
		mrows.Close()
		if len(names) == 0 {
			continue
		}
		err = l.St.InTx(func(tx *sql.Tx) error {
			for _, j := range jsons {
				name, ok := meta.MatchSidecar(j.name, names)
				if !ok {
					continue
				}
				if _, err := tx.Exec(`UPDATE files SET sidecar_of = ?, kind = 'sidecar' WHERE id = ?`, ids[name], j.id); err != nil {
					return err
				}
				if _, err := tx.Exec(`UPDATE files SET meta_done = 0 WHERE id = ?`, ids[name]); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

type metaResult struct {
	id       int64
	info     meta.Info
	takeout  *meta.Takeout
	mtime    int64
	override int64
	metaJSON string
}

// ExtractMeta reads embedded and sidecar metadata for media not processed yet.
func (l *Library) ExtractMeta(ctx context.Context, progress func(done, total int)) error {
	files, err := l.pending(`meta_done = 0 AND kind IN ('image', 'video')`)
	if err != nil || len(files) == 0 {
		return err
	}
	jobs := make(chan store.File)
	results := make(chan metaResult)
	var wg sync.WaitGroup
	for i := 0; i < workers(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				results <- l.readMeta(f)
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, f := range files {
			select {
			case jobs <- f:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() { wg.Wait(); close(results) }()

	done := 0
	batch := make([]metaResult, 0, 200)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		err := l.St.InTx(func(tx *sql.Tx) error {
			for _, r := range batch {
				if err := saveMeta(tx, r); err != nil {
					return err
				}
			}
			return nil
		})
		batch = batch[:0]
		if progress != nil {
			progress(done, len(files))
		}
		return err
	}
	var firstErr error
	for r := range results {
		done++
		batch = append(batch, r)
		if len(batch) == cap(batch) && firstErr == nil {
			firstErr = flush()
		}
	}
	if firstErr == nil {
		firstErr = flush()
	}
	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}

func (l *Library) readMeta(f store.File) metaResult {
	r := metaResult{id: f.ID, mtime: f.Mtime, override: f.DateOverride, info: meta.Info{Orientation: 1}}
	if f.Kind == store.KindImage {
		if info, err := meta.Extract(l.Abs(f), f.Ext); err == nil {
			r.info = info
		}
	}
	r.takeout = l.takeoutFor(f)

	doc := map[string]any{}
	if len(r.info.Tags) > 0 {
		doc["tags"] = r.info.Tags
	}
	if r.takeout != nil {
		doc["takeout"] = r.takeout
	}
	if len(doc) > 0 {
		if b, err := json.Marshal(doc); err == nil {
			r.metaJSON = string(b)
		}
	}
	return r
}

// takeoutFor loads the Takeout sidecar of a file; "-edited" variants borrow
// the sidecar of their original. Once the sidecar is gone, the copy kept in
// the index is used.
func (l *Library) takeoutFor(f store.File) *meta.Takeout {
	sidecars, _ := l.St.Sidecars(f.ID)
	if len(sidecars) == 0 {
		if orig, ok := meta.EditedOriginal(f.Name); ok {
			var origID int64
			err := l.St.DB.QueryRow(`SELECT id FROM files WHERE dir = ? AND name = ? AND status = 'present'`,
				f.Dir, orig).Scan(&origID)
			if err == nil {
				sidecars, _ = l.St.Sidecars(origID)
			}
		}
	}
	for _, sc := range sidecars {
		data, err := os.ReadFile(l.Abs(sc))
		if err != nil {
			continue
		}
		if t, ok := meta.ParseTakeout(data); ok {
			return &t
		}
	}
	return l.storedTakeout(f.ID)
}

func saveMeta(tx *sql.Tx, r metaResult) error {
	takenAt, src := r.mtime, store.SrcMtime
	lat, lon, hasGPS := r.info.Lat, r.info.Lon, r.info.HasGPS
	if r.takeout != nil {
		if r.takeout.TakenAt > 0 {
			takenAt, src = r.takeout.TakenAt, store.SrcTakeout
		}
		if !hasGPS && (r.takeout.Lat != 0 || r.takeout.Lon != 0) {
			lat, lon, hasGPS = r.takeout.Lat, r.takeout.Lon, true
		}
	}
	if r.info.TakenAt > 0 {
		takenAt, src = r.info.TakenAt, store.SrcExif
	}
	if r.override > 0 {
		takenAt, src = r.override, store.SrcManual
	}
	// An empty value keeps the copy already stored for the file.
	takeoutJSON := ""
	if r.takeout != nil {
		if b, err := json.Marshal(r.takeout); err == nil {
			takeoutJSON = string(b)
		}
	}
	_, err := tx.Exec(`UPDATE files SET taken_at = ?, taken_src = ?, width = ?, height = ?, orientation = ?,
		camera = ?, has_gps = ?, lat = ?, lon = ?, meta_json = ?, meta_done = 1,
		takeout_json = CASE WHEN ? != '' THEN ? ELSE takeout_json END WHERE id = ?`,
		takenAt, src, r.info.Width, r.info.Height, r.info.Orientation, r.info.Camera, hasGPS, lat, lon, r.metaJSON,
		takeoutJSON, takeoutJSON, r.id)
	return err
}

// DateResult reports where SetDate stored the new date.
type DateResult struct {
	// Written counts files that now carry the date inside the file.
	Written int `json:"written"`
	// Indexed counts files whose format cannot hold it, so the date is kept
	// in the index only.
	Indexed int `json:"indexed"`
}

// SetDate gives photos and videos a new capture date. The date is written
// into the file itself when the format allows it; otherwise it is stored in
// the index, where it takes precedence over the file's metadata.
func (l *Library) SetDate(ids []int64, t time.Time) (DateResult, error) {
	var res DateResult
	for _, id := range ids {
		f, err := l.St.GetFile(id)
		if err != nil {
			return res, err
		}
		if f.Status != store.StatusPresent || (f.Kind != store.KindImage && f.Kind != store.KindVideo) {
			return res, fmt.Errorf("%s: only photos and videos in the library can get a new date", f.RelPath)
		}

		err = meta.ErrCannotEmbed
		if f.Kind == store.KindImage {
			err = meta.WriteDate(l.Abs(f), f.Ext, t)
		}
		switch {
		case errors.Is(err, meta.ErrCannotEmbed):
			if _, err := l.St.DB.Exec(`UPDATE files SET date_override = ?, taken_at = ?, taken_src = ? WHERE id = ?`,
				t.Unix(), t.Unix(), store.SrcManual, id); err != nil {
				return res, err
			}
			res.Indexed++
		case err != nil:
			return res, fmt.Errorf("%s: %w", f.RelPath, err)
		default:
			if err := l.fileRewritten(f); err != nil {
				return res, err
			}
			res.Written++
		}
	}
	return res, nil
}

// fileRewritten records that the content of f changed on disk: size and
// mtime are refreshed so the next scan does not report it, and the hash and
// metadata are computed again from the file, which now holds the date.
func (l *Library) fileRewritten(f store.File) error {
	info, err := os.Stat(l.Abs(f))
	if err != nil {
		return err
	}
	_, err = l.St.DB.Exec(`UPDATE files SET size = ?, mtime = ?, hash = '', meta_done = 0, date_override = 0
		WHERE id = ?`, info.Size(), info.ModTime().Unix(), f.ID)
	return err
}

// HashPending computes content hashes for files that do not have one yet.
func (l *Library) HashPending(ctx context.Context, progress func(done, total int)) error {
	files, err := l.pending(`hash = ''`)
	if err != nil {
		return err
	}
	for i, f := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if h, err := scanner.HashFile(l.Abs(f)); err == nil {
			// Guard on size and mtime so a file changed meanwhile is hashed again later.
			if _, err := l.St.DB.Exec(`UPDATE files SET hash = ? WHERE id = ? AND size = ? AND mtime = ?`,
				h, f.ID, f.Size, f.Mtime); err != nil {
				return err
			}
		}
		if progress != nil && (i%20 == 0 || i == len(files)-1) {
			progress(i+1, len(files))
		}
	}
	return nil
}

func (l *Library) pending(cond string) ([]store.File, error) {
	rows, err := l.St.DB.Query(`SELECT id, rel_path, dir, name, ext, kind, size, mtime, date_override FROM files
		WHERE status = 'present' AND ` + cond + ` ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.File
	for rows.Next() {
		f := store.File{Status: store.StatusPresent}
		if err := rows.Scan(&f.ID, &f.RelPath, &f.Dir, &f.Name, &f.Ext, &f.Kind, &f.Size, &f.Mtime, &f.DateOverride); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func workers() int {
	n := runtime.NumCPU() - 1
	if n < 1 {
		n = 1
	}
	return n
}
