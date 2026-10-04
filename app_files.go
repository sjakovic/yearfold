package main

import (
	"time"

	"github.com/sjakovic/yearfold/internal/fileops"
	"github.com/sjakovic/yearfold/internal/library"
	"github.com/sjakovic/yearfold/internal/organize"
	"github.com/sjakovic/yearfold/internal/store"
)

// MoveFiles physically moves files into destDir inside the root.
func (a *App) MoveFiles(ids []int64, destDir string) (int, error) {
	lib, err := a.library()
	if err != nil {
		return 0, err
	}
	targets := make([]fileops.Target, len(ids))
	for i, id := range ids {
		targets[i] = fileops.Target{ID: id, DestDir: destDir}
	}
	return lib.Ops.Move(targets)
}

// SetDate gives the selected photos and videos a new capture date (Unix
// seconds, taken as wall-clock time), inside the file where possible.
func (a *App) SetDate(ids []int64, unix int64) (library.DateResult, error) {
	lib, err := a.library()
	if err != nil {
		return library.DateResult{}, err
	}
	res, err := lib.SetDate(ids, time.Unix(unix, 0).UTC())
	if res.Written > 0 {
		go a.process()
	}
	return res, err
}

func (a *App) TrashFiles(ids []int64) (int, error) {
	lib, err := a.library()
	if err != nil {
		return 0, err
	}
	return lib.Ops.Trash(ids)
}

func (a *App) RestoreFiles(ids []int64) (int, error) {
	lib, err := a.library()
	if err != nil {
		return 0, err
	}
	return lib.Ops.Restore(ids)
}

func (a *App) EmptyTrash() (int, error) {
	lib, err := a.library()
	if err != nil {
		return 0, err
	}
	return lib.Ops.EmptyTrash()
}

// Undo reverts the last move or trash operation.
func (a *App) Undo() (int, error) {
	lib, err := a.library()
	if err != nil {
		return 0, err
	}
	return lib.Ops.Undo()
}

type OrganizePreview struct {
	Count  int             `json:"count"`
	NoDate int             `json:"noDate"`
	Moves  []organize.Move `json:"moves"` // sample
}

// PlanOrganize previews sorting media into year (or year/month) folders.
func (a *App) PlanOrganize(byMonth bool) (OrganizePreview, error) {
	p := OrganizePreview{Moves: []organize.Move{}}
	lib, err := a.library()
	if err != nil {
		return p, err
	}
	plan, err := organize.ByDate(lib.St, byMonth)
	if err != nil {
		return p, err
	}
	p.Count, p.NoDate = len(plan.Moves), plan.NoDate
	p.Moves = plan.Moves
	if len(p.Moves) > sampleLimit {
		p.Moves = p.Moves[:sampleLimit]
	}
	return p, nil
}

// ApplyOrganize performs the moves PlanOrganize previews, as one undoable batch.
func (a *App) ApplyOrganize(byMonth bool) (int, error) {
	lib, err := a.library()
	if err != nil {
		return 0, err
	}
	plan, err := organize.ByDate(lib.St, byMonth)
	if err != nil {
		return 0, err
	}
	targets := make([]fileops.Target, len(plan.Moves))
	for i, m := range plan.Moves {
		targets[i] = fileops.Target{ID: m.ID, DestDir: m.To}
	}
	return lib.Ops.Move(targets)
}

func (a *App) Duplicates() ([][]store.Item, error) {
	lib, err := a.library()
	if err != nil {
		return [][]store.Item{}, err
	}
	return lib.St.Duplicates()
}

func (a *App) ClearThumbs() error {
	lib, err := a.library()
	if err != nil {
		return err
	}
	return lib.Thumbs.Clear()
}
