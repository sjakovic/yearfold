// Package organize plans rule-based moves, such as sorting media into
// per-year folders. Planning never touches the disk.
package organize

import (
	"fmt"
	"time"

	"github.com/sjakovic/yearfold/internal/store"
)

type Move struct {
	ID   int64  `json:"id"`
	From string `json:"from"`
	To   string `json:"to"` // destination directory
}

type Plan struct {
	Moves []Move `json:"moves"`
	// NoDate counts media skipped because it has no reliable capture date.
	NoDate int `json:"noDate"`
}

// ByDate plans moving every dated media file into "YYYY" or, with byMonth,
// "YYYY/MM" at the top of the library.
func ByDate(st *store.Store, byMonth bool) (Plan, error) {
	plan := Plan{Moves: []Move{}}
	rows, err := st.DB.Query(`SELECT id, rel_path, dir, taken_at, taken_src FROM files
		WHERE status = 'present' AND kind IN ('image', 'video') ORDER BY rel_path`)
	if err != nil {
		return plan, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id            int64
			rel, dir, src string
			takenAt       int64
		)
		if err := rows.Scan(&id, &rel, &dir, &takenAt, &src); err != nil {
			return plan, err
		}
		if src == "" || src == store.SrcMtime {
			plan.NoDate++
			continue
		}
		t := time.Unix(takenAt, 0).UTC()
		dest := fmt.Sprintf("%04d", t.Year())
		if byMonth {
			dest = fmt.Sprintf("%04d/%02d", t.Year(), int(t.Month()))
		}
		if dir != dest {
			plan.Moves = append(plan.Moves, Move{ID: id, From: rel, To: dest})
		}
	}
	return plan, rows.Err()
}
