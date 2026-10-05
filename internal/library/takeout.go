package library

import (
	"context"
	"errors"
	"time"

	"github.com/sjakovic/yearfold/internal/meta"
	"github.com/sjakovic/yearfold/internal/store"
)

type TakeoutSummary struct {
	Sidecars int `json:"sidecars"`
	Dates    int `json:"dates"`
}

type TakeoutResult struct {
	Trashed int `json:"trashed"`
	Dates   int `json:"dates"`
	Skipped int `json:"skipped"`
}

type mergeCandidate struct {
	sidecarID int64
	media     store.File
}

func (l *Library) mergeCandidates(ctx context.Context) ([]mergeCandidate, error) {
	if err := l.LinkSidecars(); err != nil {
		return nil, err
	}
	if err := l.ExtractMeta(ctx, nil); err != nil {
		return nil, err
	}
	links, err := l.St.MergeableSidecars()
	if err != nil {
		return nil, err
	}
	out := make([]mergeCandidate, 0, len(links))
	for _, link := range links {
		m, err := l.St.GetFile(link.MediaID)
		if err != nil {
			return nil, err
		}
		out = append(out, mergeCandidate{sidecarID: link.SidecarID, media: m})
	}
	return out, nil
}

func needsDateInFile(f store.File) bool {
	return f.Kind == store.KindImage && f.TakenSrc == store.SrcTakeout && meta.CanEmbedDate(f.Ext)
}

func (l *Library) PlanTakeoutMerge(ctx context.Context) (TakeoutSummary, error) {
	var s TakeoutSummary
	candidates, err := l.mergeCandidates(ctx)
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

func (l *Library) MergeTakeout(ctx context.Context) (TakeoutResult, error) {
	var res TakeoutResult
	candidates, err := l.mergeCandidates(ctx)
	if err != nil {
		return res, err
	}

	failed := map[int64]bool{}
	handled := map[int64]bool{}
	var trash []int64
	for _, c := range candidates {
		m := c.media
		if !handled[m.ID] {
			handled[m.ID] = true
			if needsDateInFile(m) {
				if err := l.embedDate(m); err != nil {
					failed[m.ID] = true
				} else {
					res.Dates++
				}
			}
		}
		if failed[m.ID] {
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
