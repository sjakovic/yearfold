package library

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/sjakovic/yearfold/internal/meta"
	"github.com/sjakovic/yearfold/internal/store"
)

type DateResult struct {
	Written int `json:"written"` // date stored inside the file
	Indexed int `json:"indexed"` // date kept in the index only
}

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
			if err := l.St.SetDateOverride(id, t.Unix()); err != nil {
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

func (l *Library) fileRewritten(f store.File) error {
	info, err := os.Stat(l.Abs(f))
	if err != nil {
		return err
	}
	return l.St.MarkRewritten(f.ID, info.Size(), info.ModTime().Unix())
}
