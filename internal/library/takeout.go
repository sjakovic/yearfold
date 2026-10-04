package library

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/sjakovic/yearfold/internal/meta"
	"github.com/sjakovic/yearfold/internal/store"
)

// The data of a Takeout sidecar is copied into the index (takeout_json) the
// first time a file's metadata is read. From then on the library no longer
// depends on the JSON file, which is what makes merging it away safe.

// storedTakeout returns the Takeout data kept in the index for a file.
func (l *Library) storedTakeout(id int64) *meta.Takeout {
	var raw string
	if err := l.St.DB.QueryRow(`SELECT takeout_json FROM files WHERE id = ?`, id).Scan(&raw); err != nil || raw == "" {
		return nil
	}
	var t meta.Takeout
	if json.Unmarshal([]byte(raw), &t) != nil {
		return nil
	}
	return &t
}

// TakeoutSummary describes what MergeTakeout would do.
type TakeoutSummary struct {
	// Sidecars is the number of Google JSON files attached to media.
	Sidecars int `json:"sidecars"`
	// Dates is the number of photos that would get the date written into the file.
	Dates int `json:"dates"`
}

// TakeoutResult reports what MergeTakeout did.
type TakeoutResult struct {
	// Trashed is the number of JSON files moved to the trash.
	Trashed int `json:"trashed"`
	// Dates is the number of photos that got the date written into the file.
	Dates int `json:"dates"`
	// Skipped is the number of JSON files left in place because writing the
	// date into their photo failed.
	Skipped int `json:"skipped"`
}

// mergeCandidate is a sidecar whose data is safely stored in the index.
type mergeCandidate struct {
	sidecarID int64
	media     store.File
}

func (l *Library) mergeCandidates() ([]mergeCandidate, error) {
	rows, err := l.St.DB.Query(`SELECT s.id, m.id FROM files s JOIN files m ON m.id = s.sidecar_of
		WHERE s.status = 'present' AND m.status = 'present' AND m.meta_done = 1 AND m.takeout_json != ''
		ORDER BY m.id`)
	if err != nil {
		return nil, err
	}
	var pairs [][2]int64
	for rows.Next() {
		var p [2]int64
		if err := rows.Scan(&p[0], &p[1]); err != nil {
			rows.Close()
			return nil, err
		}
		pairs = append(pairs, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]mergeCandidate, 0, len(pairs))
	for _, p := range pairs {
		m, err := l.St.GetFile(p[1])
		if err != nil {
			return nil, err
		}
		out = append(out, mergeCandidate{sidecarID: p[0], media: m})
	}
	return out, nil
}

// needsDateInFile reports whether a file's only date is the Takeout one and
// its format can hold a date.
func needsDateInFile(f store.File) bool {
	return f.Kind == store.KindImage && f.TakenSrc == store.SrcTakeout && meta.CanEmbedDate(f.Ext)
}

// PlanTakeoutMerge counts what MergeTakeout would do, after making sure all
// metadata has been read.
func (l *Library) PlanTakeoutMerge(ctx context.Context) (TakeoutSummary, error) {
	var s TakeoutSummary
	candidates, err := l.readyCandidates(ctx)
	if err != nil {
		return s, err
	}
	seen := map[int64]bool{}
	for _, c := range candidates {
		s.Sidecars++
		if !seen[c.media.ID] && needsDateInFile(c.media) {
			s.Dates++
		}
		seen[c.media.ID] = true
	}
	return s, nil
}

func (l *Library) readyCandidates(ctx context.Context) ([]mergeCandidate, error) {
	if err := l.LinkSidecars(); err != nil {
		return nil, err
	}
	if err := l.ExtractMeta(ctx, nil); err != nil {
		return nil, err
	}
	return l.mergeCandidates()
}

// MergeTakeout makes the Google JSON files unnecessary and moves them to the
// trash. Their data already lives in the index; in addition, photos whose
// only date comes from the JSON get it written into the file itself when
// the format allows. The JSON files can be restored from the trash or with
// Undo; dates written into photos stay.
func (l *Library) MergeTakeout(ctx context.Context) (TakeoutResult, error) {
	var res TakeoutResult
	candidates, err := l.readyCandidates(ctx)
	if err != nil {
		return res, err
	}

	dated := map[int64]error{}
	var trash []int64
	for _, c := range candidates {
		m := c.media
		if _, done := dated[m.ID]; !done {
			dated[m.ID] = nil
			if needsDateInFile(m) {
				dated[m.ID] = l.embedDate(m)
				if dated[m.ID] == nil {
					res.Dates++
				}
			}
		}
		if dated[m.ID] != nil {
			res.Skipped++
			continue
		}
		trash = append(trash, c.sidecarID)
	}
	if len(trash) > 0 {
		res.Trashed, err = l.Ops.Trash(trash)
	}
	return res, err
}

// embedDate writes a file's current capture date into the file itself.
func (l *Library) embedDate(f store.File) error {
	err := meta.WriteDate(l.Abs(f), f.Ext, time.Unix(f.TakenAt, 0).UTC())
	if errors.Is(err, meta.ErrCannotEmbed) {
		return nil
	}
	if err != nil {
		return err
	}
	return l.fileRewritten(f)
}
