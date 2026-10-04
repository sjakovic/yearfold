package main

import (
	"errors"

	"github.com/sjakovic/yearfold/internal/scanner"
)

type MovedPath struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ChangeReport summarises what CheckChanges found. Path lists are samples.
type ChangeReport struct {
	NewCount      int         `json:"newCount"`
	ModifiedCount int         `json:"modifiedCount"`
	MissingCount  int         `json:"missingCount"`
	MovedCount    int         `json:"movedCount"`
	New           []string    `json:"new"`
	Modified      []string    `json:"modified"`
	Missing       []string    `json:"missing"`
	Moved         []MovedPath `json:"moved"`
}

// CheckChanges compares the disk with the index without changing the index.
func (a *App) CheckChanges() (ChangeReport, error) {
	rep := ChangeReport{New: []string{}, Modified: []string{}, Missing: []string{}, Moved: []MovedPath{}}
	lib, err := a.library()
	if err != nil {
		return rep, err
	}
	ch, err := scanner.Diff(lib.Root, lib.St, func(seen int) { a.progress("scan", seen, 0) })
	a.progress("", 0, 0)
	if err != nil {
		return rep, err
	}
	a.mu.Lock()
	a.pending = ch
	a.mu.Unlock()

	rep.NewCount = len(ch.New)
	rep.ModifiedCount = len(ch.Modified) + len(ch.Revived)
	rep.MissingCount = len(ch.Missing)
	rep.MovedCount = len(ch.Moved)
	for _, e := range ch.New {
		if len(rep.New) < sampleLimit {
			rep.New = append(rep.New, e.RelPath)
		}
	}
	for _, k := range append(append([]scanner.Known{}, ch.Modified...), ch.Revived...) {
		if len(rep.Modified) < sampleLimit {
			rep.Modified = append(rep.Modified, k.RelPath)
		}
	}
	for _, k := range ch.Missing {
		if len(rep.Missing) < sampleLimit {
			rep.Missing = append(rep.Missing, k.RelPath)
		}
	}
	for _, m := range ch.Moved {
		if len(rep.Moved) < sampleLimit {
			rep.Moved = append(rep.Moved, MovedPath{From: m.From, To: m.To.RelPath})
		}
	}
	return rep, nil
}

// ApplyChanges writes the result of the last CheckChanges to the index.
func (a *App) ApplyChanges() error {
	lib, err := a.library()
	if err != nil {
		return err
	}
	a.mu.Lock()
	ch := a.pending
	a.pending = nil
	a.mu.Unlock()
	if ch == nil {
		return errors.New("nothing to apply; check for changes first")
	}
	if err := lib.Apply(ch); err != nil {
		return err
	}
	go a.process()
	return nil
}
