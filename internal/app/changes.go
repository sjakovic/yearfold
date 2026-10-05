package app

import (
	"errors"

	"github.com/sjakovic/yearfold/internal/scanner"
	"github.com/sjakovic/yearfold/internal/store"
)

type MovedPath struct {
	From string `json:"from"`
	To   string `json:"to"`
}

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

func newChangeReport(ch *scanner.Changes) ChangeReport {
	added := make([]string, len(ch.New))
	for i, f := range ch.New {
		added[i] = f.RelPath
	}
	modified := append(knownPaths(ch.Modified), knownPaths(ch.Revived)...)
	missing := knownPaths(ch.Missing)
	moved := make([]MovedPath, len(ch.Moved))
	for i, m := range ch.Moved {
		moved[i] = MovedPath{From: m.From, To: m.To.RelPath}
	}
	return ChangeReport{
		NewCount:      len(added),
		ModifiedCount: len(modified),
		MissingCount:  len(missing),
		MovedCount:    len(moved),
		New:           sample(added),
		Modified:      sample(modified),
		Missing:       sample(missing),
		Moved:         sample(moved),
	}
}

func knownPaths(files []store.KnownStat) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.RelPath
	}
	return out
}

func sample[T any](items []T) []T {
	if len(items) > sampleLimit {
		return items[:sampleLimit]
	}
	return items
}

func (a *App) CheckChanges() (ChangeReport, error) {
	lib, err := a.library()
	if err != nil {
		return newChangeReport(&scanner.Changes{}), err
	}
	ch, err := scanner.Diff(lib.Root, lib.St, func(seen int) { a.progress("scan", seen, 0) })
	a.idle()
	if err != nil {
		return newChangeReport(&scanner.Changes{}), err
	}
	a.mu.Lock()
	a.pending = ch
	a.mu.Unlock()
	return newChangeReport(ch), nil
}

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
