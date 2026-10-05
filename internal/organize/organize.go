// Package organize plans moves of media into year folders.
package organize

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/sjakovic/yearfold/internal/store"
)

type Layout string

const (
	LayoutYear   Layout = "year"   // 2005
	LayoutMonth  Layout = "month"  // 2005/03
	LayoutFolder Layout = "folder" // 2005/Trip to Rome
)

type Move struct {
	ID   int64  `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

type Plan struct {
	Moves  []Move `json:"moves"`
	NoDate int    `json:"noDate"`
}

func ByDate(st *store.Store, root string, layout Layout) (Plan, error) {
	plan := Plan{Moves: []Move{}}
	switch layout {
	case LayoutYear, LayoutMonth, LayoutFolder:
	default:
		return plan, fmt.Errorf("unknown layout %q", layout)
	}
	media, err := st.Media()
	if err != nil {
		return plan, err
	}
	var folders *folderNamer
	if layout == LayoutFolder {
		if folders, err = newFolderNamer(st, root); err != nil {
			return plan, err
		}
	}

	for _, m := range media {
		if !m.HasDate() {
			plan.NoDate++
			continue
		}
		t := time.Unix(m.TakenAt, 0).UTC()
		year := fmt.Sprintf("%04d", t.Year())

		var dest string
		switch layout {
		case LayoutYear:
			dest = year
		case LayoutMonth:
			dest = fmt.Sprintf("%s/%02d", year, int(t.Month()))
		case LayoutFolder:
			if inYearFolder(m.Dir) {
				continue
			}
			dest = folders.destination(m.Dir, year)
		}
		if m.Dir != dest {
			plan.Moves = append(plan.Moves, Move{ID: m.ID, From: m.RelPath, To: dest})
		}
	}
	return plan, nil
}

func isYear(name string) bool {
	if len(name) != 4 {
		return false
	}
	for _, c := range name {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func inYearFolder(dir string) bool {
	top, _, _ := strings.Cut(dir, "/")
	return isYear(top)
}

type source struct {
	dir, year string
}

type folderNamer struct {
	root     string
	existing map[string]bool
	assigned map[source]string
	owned    map[string]bool
}

func newFolderNamer(st *store.Store, root string) (*folderNamer, error) {
	n := &folderNamer{root: root, existing: map[string]bool{}, assigned: map[source]string{}, owned: map[string]bool{}}

	dirs, err := st.Dirs()
	if err != nil {
		return nil, err
	}
	for _, d := range dirs {
		for dir := d.Dir; dir != "" && dir != "."; dir = path.Dir(dir) {
			n.existing[dir] = true
		}
	}

	history, err := st.MoveHistory()
	if err != nil {
		return nil, err
	}
	for _, h := range history {
		fromDir, _, _ := store.SplitPath(h.From)
		toDir, _, _ := store.SplitPath(h.To)
		year, name, _ := strings.Cut(toDir, "/")
		if isYear(year) && name != "" && !strings.Contains(name, "/") {
			n.assigned[source{dir: fromDir, year: year}] = toDir
			n.owned[toDir] = true
		}
	}
	return n, nil
}

func (n *folderNamer) destination(dir, year string) string {
	src := source{dir: dir, year: year}
	if dest, ok := n.assigned[src]; ok {
		return dest
	}
	name := path.Base(dir)
	if dir == "" || name == year {
		return year
	}
	for i := 1; ; i++ {
		dest := year + "/" + name
		if i > 1 {
			dest = fmt.Sprintf("%s %d", dest, i)
		}
		if !n.taken(dest) {
			n.assigned[src] = dest
			n.owned[dest] = true
			return dest
		}
	}
}

func (n *folderNamer) taken(dest string) bool {
	if n.owned[dest] || n.existing[dest] {
		return true
	}
	_, err := os.Lstat(filepath.Join(n.root, filepath.FromSlash(dest)))
	return !errors.Is(err, os.ErrNotExist)
}
