package app

import (
	"fmt"
	"testing"

	"github.com/sjakovic/yearfold/internal/scanner"
	"github.com/sjakovic/yearfold/internal/store"
)

func TestChangeReportCountsEverythingButSamplesPaths(t *testing.T) {
	ch := &scanner.Changes{
		Modified: []store.KnownStat{{ID: 1, FileStat: store.FileStat{RelPath: "changed.jpg"}}},
		Revived:  []store.KnownStat{{ID: 2, FileStat: store.FileStat{RelPath: "back.jpg"}}},
		Missing:  []store.KnownStat{{ID: 3, FileStat: store.FileStat{RelPath: "gone.jpg"}}},
		Moved:    []scanner.Moved{{ID: 4, From: "a.jpg", To: store.FileStat{RelPath: "b.jpg"}}},
	}
	for i := 0; i < sampleLimit+50; i++ {
		ch.New = append(ch.New, store.FileStat{RelPath: fmt.Sprintf("new%d.jpg", i)})
	}

	r := newChangeReport(ch)
	if r.NewCount != sampleLimit+50 || len(r.New) != sampleLimit {
		t.Errorf("new: count %d, sample %d", r.NewCount, len(r.New))
	}
	if r.ModifiedCount != 2 || r.Modified[1] != "back.jpg" {
		t.Errorf("modified = %v", r.Modified)
	}
	if r.MissingCount != 1 || r.MovedCount != 1 || r.Moved[0] != (MovedPath{From: "a.jpg", To: "b.jpg"}) {
		t.Errorf("report = %+v", r)
	}
}

func TestEmptyChangeReportHasNoNilLists(t *testing.T) {
	r := newChangeReport(&scanner.Changes{})
	if r.New == nil || r.Modified == nil || r.Missing == nil || r.Moved == nil {
		t.Errorf("nil list in %+v", r)
	}
}
