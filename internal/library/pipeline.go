package library

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"sync"

	"github.com/sjakovic/yearfold/internal/meta"
	"github.com/sjakovic/yearfold/internal/scanner"
	"github.com/sjakovic/yearfold/internal/store"
)

const metaBatchSize = 200

func (l *Library) ExtractMeta(ctx context.Context, progress func(done, total int)) error {
	files, err := l.St.PendingMeta()
	if err != nil || len(files) == 0 {
		return err
	}

	jobs := make(chan store.File)
	results := make(chan store.MetaUpdate)
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
	go func() {
		wg.Wait()
		close(results)
	}()

	done := 0
	batch := make([]store.MetaUpdate, 0, metaBatchSize)
	var saveErr error
	flush := func() {
		if saveErr == nil {
			saveErr = l.St.SaveMeta(batch)
		}
		batch = batch[:0]
		if progress != nil {
			progress(done, len(files))
		}
	}
	for r := range results {
		done++
		batch = append(batch, r)
		if len(batch) == metaBatchSize {
			flush()
		}
	}
	flush()
	if saveErr != nil {
		return saveErr
	}
	return ctx.Err()
}

func (l *Library) readMeta(f store.File) store.MetaUpdate {
	info := meta.Info{Orientation: 1}
	if f.Kind == store.KindImage {
		if got, err := meta.Extract(l.Abs(f), f.Ext); err == nil {
			info = got
		}
	}
	takeout := l.takeoutFor(f)

	u := store.MetaUpdate{
		ID:          f.ID,
		TakenAt:     f.Mtime,
		TakenSrc:    store.SrcMtime,
		Width:       info.Width,
		Height:      info.Height,
		Orientation: info.Orientation,
		Camera:      info.Camera,
		HasGPS:      info.HasGPS,
		Lat:         info.Lat,
		Lon:         info.Lon,
	}
	doc := map[string]any{}
	if len(info.Tags) > 0 {
		doc["tags"] = info.Tags
	}
	if takeout != nil {
		doc["takeout"] = takeout
		if takeout.TakenAt > 0 {
			u.TakenAt, u.TakenSrc = takeout.TakenAt, store.SrcTakeout
		}
		if !u.HasGPS && (takeout.Lat != 0 || takeout.Lon != 0) {
			u.HasGPS, u.Lat, u.Lon = true, takeout.Lat, takeout.Lon
		}
		if b, err := json.Marshal(takeout); err == nil {
			u.TakeoutJSON = string(b)
		}
	}
	if info.TakenAt > 0 {
		u.TakenAt, u.TakenSrc = info.TakenAt, store.SrcExif
	}
	if f.DateOverride > 0 {
		u.TakenAt, u.TakenSrc = f.DateOverride, store.SrcManual
	}
	if len(doc) > 0 {
		if b, err := json.Marshal(doc); err == nil {
			u.MetaJSON = string(b)
		}
	}
	return u
}

func (l *Library) takeoutFor(f store.File) *meta.Takeout {
	sidecars, _ := l.St.Sidecars(f.ID)
	if len(sidecars) == 0 {
		if original, ok := meta.EditedOriginal(f.Name); ok {
			if id, found, _ := l.St.PresentID(f.Dir, original); found {
				sidecars, _ = l.St.Sidecars(id)
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

func (l *Library) storedTakeout(id int64) *meta.Takeout {
	raw, err := l.St.TakeoutJSON(id)
	if err != nil || raw == "" {
		return nil
	}
	var t meta.Takeout
	if json.Unmarshal([]byte(raw), &t) != nil {
		return nil
	}
	return &t
}

func (l *Library) HashPending(ctx context.Context, progress func(done, total int)) error {
	files, err := l.St.PendingHash()
	if err != nil {
		return err
	}
	for i, f := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if hash, err := scanner.HashFile(l.Abs(f)); err == nil {
			if err := l.St.SetHash(f.ID, hash, f.Size, f.Mtime); err != nil {
				return err
			}
		}
		if progress != nil && (i%20 == 0 || i == len(files)-1) {
			progress(i+1, len(files))
		}
	}
	return nil
}

func workers() int {
	return max(runtime.NumCPU()-1, 1)
}
