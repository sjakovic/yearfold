// Package organize plans rule-based moves, such as sorting media into
// per-year folders. Planning never changes anything on disk.
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

// Layout is the folder structure media is sorted into.
type Layout string

const (
	// LayoutYear puts everything directly into "2005".
	LayoutYear Layout = "year"
	// LayoutMonth puts everything into "2005/03".
	LayoutMonth Layout = "month"
	// LayoutFolder keeps the name of the folder a file is in: "2005/Trip to Rome".
	// Files that already are inside a year folder are left where they are,
	// whatever their date, so folders arranged by hand stay as arranged.
	LayoutFolder Layout = "folder"
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

type mediaFile struct {
	id       int64
	rel, dir string
	takenAt  int64
	src      string
}

// ByDate plans moving every dated media file of the library at root into
// year folders at the top of the library, following layout.
func ByDate(st *store.Store, root string, layout Layout) (Plan, error) {
	plan := Plan{Moves: []Move{}}
	switch layout {
	case LayoutYear, LayoutMonth, LayoutFolder:
	default:
		return plan, fmt.Errorf("unknown layout %q", layout)
	}
	files, err := mediaFiles(st)
	if err != nil {
		return plan, err
	}
	var folders *folderNamer
	if layout == LayoutFolder {
		if folders, err = newFolderNamer(st, root); err != nil {
			return plan, err
		}
	}

	for _, f := range files {
		if f.src == "" || f.src == store.SrcMtime {
			plan.NoDate++
			continue
		}
		t := time.Unix(f.takenAt, 0).UTC()
		year := fmt.Sprintf("%04d", t.Year())

		var dest string
		switch layout {
		case LayoutYear:
			dest = year
		case LayoutMonth:
			dest = fmt.Sprintf("%s/%02d", year, int(t.Month()))
		case LayoutFolder:
			if inYearFolder(f.dir) {
				continue
			}
			dest = folders.destination(f.dir, year)
		}
		if f.dir != dest {
			plan.Moves = append(plan.Moves, Move{ID: f.id, From: f.rel, To: dest})
		}
	}
	return plan, nil
}

func mediaFiles(st *store.Store) ([]mediaFile, error) {
	rows, err := st.DB.Query(`SELECT id, rel_path, dir, taken_at, taken_src FROM files
		WHERE status = 'present' AND kind IN ('image', 'video') ORDER BY rel_path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []mediaFile
	for rows.Next() {
		var f mediaFile
		if err := rows.Scan(&f.id, &f.rel, &f.dir, &f.takenAt, &f.src); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// isYear reports whether name looks like a year folder: four digits.
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

// inYearFolder reports whether dir is a year folder or anything below one.
func inYearFolder(dir string) bool {
	top, _, _ := strings.Cut(dir, "/")
	return isYear(top)
}

// source identifies the files of one original folder that belong to one year.
type source struct {
	dir, year string
}

// folderNamer picks the destination folder for LayoutFolder. All files of a
// source folder and year share one destination; different source folders
// that happen to have the same name get "Name", "Name 2", "Name 3", ...
type folderNamer struct {
	root string
	// existing holds every folder that already contains indexed files.
	existing map[string]bool
	// assigned maps a source to its destination, seeded from earlier moves
	// so that files organized later join the ones moved before.
	assigned map[source]string
	// owned holds the destinations that already belong to a source.
	owned map[string]bool
}

func newFolderNamer(st *store.Store, root string) (*folderNamer, error) {
	n := &folderNamer{root: root, existing: map[string]bool{}, assigned: map[source]string{}, owned: map[string]bool{}}

	dirs, err := st.Dirs()
	if err != nil {
		return nil, err
	}
	for _, d := range dirs {
		// A folder exists as soon as anything lives in it or below it.
		for dir := d.Dir; dir != "" && dir != "."; dir = path.Dir(dir) {
			n.existing[dir] = true
		}
	}

	rows, err := st.DB.Query(`SELECT from_path, to_path FROM ops WHERE type = 'move' AND undone = 0 ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var from, to string
		if err := rows.Scan(&from, &to); err != nil {
			return nil, err
		}
		fromDir, _, _ := store.SplitPath(from)
		toDir, _, _ := store.SplitPath(to)
		// Only moves into "<year>/<name>" tell where a source folder went.
		year, name, _ := strings.Cut(toDir, "/")
		if isYear(year) && name != "" && !strings.Contains(name, "/") {
			n.assigned[source{dir: fromDir, year: year}] = toDir
			n.owned[toDir] = true
		}
	}
	return n, rows.Err()
}

func (n *folderNamer) destination(dir, year string) string {
	src := source{dir: dir, year: year}
	if dest, ok := n.assigned[src]; ok {
		return dest
	}
	name := path.Base(dir)
	// Files in the library root, or in a folder already named after the year,
	// go straight into the year folder.
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

// taken reports whether dest already belongs to another folder: one claimed
// by a source, one holding indexed files, or one present on disk.
func (n *folderNamer) taken(dest string) bool {
	if n.owned[dest] || n.existing[dest] {
		return true
	}
	_, err := os.Lstat(filepath.Join(n.root, filepath.FromSlash(dest)))
	return !errors.Is(err, os.ErrNotExist)
}
