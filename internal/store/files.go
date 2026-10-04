package store

import (
	"database/sql"
	"strings"
	"time"
)

// File is a full row of the files table.
type File struct {
	ID          int64   `json:"id"`
	RelPath     string  `json:"relPath"`
	Dir         string  `json:"dir"`
	Name        string  `json:"name"`
	Ext         string  `json:"ext"`
	Kind        string  `json:"kind"`
	Size        int64   `json:"size"`
	Mtime       int64   `json:"mtime"`
	Hash        string  `json:"hash"`
	TakenAt     int64   `json:"takenAt"`
	TakenSrc    string  `json:"takenSrc"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	Orientation int     `json:"orientation"`
	Camera      string  `json:"camera"`
	HasGPS      bool    `json:"hasGps"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	MetaJSON    string  `json:"metaJson"`
	SidecarOf   int64   `json:"sidecarOf"`
	Status      string  `json:"status"`
	TrashPath   string  `json:"trashPath"`
	// DateOverride is a user-set capture date (Unix seconds) that takes
	// precedence over metadata; 0 when there is none.
	DateOverride int64 `json:"dateOverride"`
}

const fileCols = `id, rel_path, dir, name, ext, kind, size, mtime, hash, taken_at, taken_src,
	width, height, orientation, camera, has_gps, lat, lon, meta_json, COALESCE(sidecar_of, 0), status, trash_path, date_override`

type scanner interface{ Scan(dest ...any) error }

func scanFile(r scanner) (File, error) {
	var f File
	err := r.Scan(&f.ID, &f.RelPath, &f.Dir, &f.Name, &f.Ext, &f.Kind, &f.Size, &f.Mtime, &f.Hash,
		&f.TakenAt, &f.TakenSrc, &f.Width, &f.Height, &f.Orientation, &f.Camera, &f.HasGPS,
		&f.Lat, &f.Lon, &f.MetaJSON, &f.SidecarOf, &f.Status, &f.TrashPath, &f.DateOverride)
	return f, err
}

func (s *Store) queryFiles(q string, args ...any) ([]File, error) {
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []File
	for rows.Next() {
		f, err := scanFile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) GetFile(id int64) (File, error) {
	return scanFile(s.DB.QueryRow(`SELECT `+fileCols+` FROM files WHERE id = ?`, id))
}

// Sidecars returns the present sidecar files attached to a media file.
func (s *Store) Sidecars(id int64) ([]File, error) {
	return s.queryFiles(`SELECT `+fileCols+` FROM files WHERE sidecar_of = ? AND status = 'present'`, id)
}

// InsertFile adds a newly discovered file and returns its id.
func InsertFile(tx *sql.Tx, rel string, size, mtime int64) (int64, error) {
	dir, name, ext := SplitPath(rel)
	res, err := tx.Exec(`INSERT INTO files (rel_path, dir, name, ext, kind, size, mtime, taken_at, taken_src)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, rel, dir, name, ext, KindOf(ext), size, mtime, mtime, SrcMtime)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// SetPath points a file row at a new relative path.
func SetPath(tx *sql.Tx, id int64, rel string) error {
	dir, name, ext := SplitPath(rel)
	_, err := tx.Exec(`UPDATE files SET rel_path = ?, dir = ?, name = ?, ext = ? WHERE id = ?`, rel, dir, name, ext, id)
	return err
}

// PathTaken reports whether a present file already occupies rel.
func (s *Store) PathTaken(rel string) (bool, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM files WHERE rel_path = ? AND status = 'present'`, rel).Scan(&n)
	return n > 0, err
}

// Item is the light representation used by the grid.
type Item struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	RelPath  string `json:"relPath"`
	Kind     string `json:"kind"`
	Size     int64  `json:"size"`
	TakenAt  int64  `json:"takenAt"`
	TakenSrc string `json:"takenSrc"`
	Hash     string `json:"hash"`
}

// Filter selects files for the grid.
type Filter struct {
	InDir     bool   `json:"inDir"`
	Dir       string `json:"dir"`
	Recursive bool   `json:"recursive"`
	Kind      string `json:"kind"` // media, image, video, other or "" for all
	TagID     int64  `json:"tagId"`
	AlbumID   int64  `json:"albumId"`
	Search    string `json:"search"`
	Status    string `json:"status"` // defaults to present
	NoDate    bool   `json:"noDate"`
	Year      int    `json:"year"`
	Sort      string `json:"sort"` // date (default) or name
	Offset    int    `json:"offset"`
	Limit     int    `json:"limit"`
}

type Page struct {
	Items []Item `json:"items"`
	Total int    `json:"total"`
}

func (s *Store) List(f Filter) (Page, error) {
	var (
		join  string
		where []string
		args  []any
	)
	if f.AlbumID > 0 {
		join = ` JOIN album_files af ON af.file_id = f.id AND af.album_id = ?`
		args = append(args, f.AlbumID)
	}
	status := f.Status
	if status == "" {
		status = StatusPresent
	}
	where = append(where, `f.status = ?`)
	args = append(args, status)

	if f.InDir {
		if !f.Recursive {
			where = append(where, `f.dir = ?`)
			args = append(args, f.Dir)
		} else if f.Dir != "" {
			where = append(where, `(f.dir = ? OR f.dir LIKE ? ESCAPE '\')`)
			args = append(args, f.Dir, escapeLike(f.Dir)+"/%")
		}
	}
	switch f.Kind {
	case "media":
		where = append(where, `f.kind IN ('image', 'video')`)
	case KindImage, KindVideo:
		where = append(where, `f.kind = ?`)
		args = append(args, f.Kind)
	case KindOther:
		where = append(where, `f.kind IN ('other', 'sidecar')`)
	}
	if f.TagID > 0 {
		where = append(where, `EXISTS (SELECT 1 FROM file_tags ft WHERE ft.file_id = f.id AND ft.tag_id = ?)`)
		args = append(args, f.TagID)
	}
	if q := strings.TrimSpace(f.Search); q != "" {
		where = append(where, `f.rel_path LIKE ? ESCAPE '\'`)
		args = append(args, "%"+escapeLike(q)+"%")
	}
	if f.NoDate {
		where = append(where, `f.taken_src IN ('', 'mtime') AND f.kind IN ('image', 'video')`)
	}
	if f.Year > 0 {
		from := time.Date(f.Year, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
		to := time.Date(f.Year+1, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
		where = append(where, `f.taken_src NOT IN ('', 'mtime') AND f.taken_at >= ? AND f.taken_at < ?`)
		args = append(args, from, to)
	}
	base := ` FROM files f` + join + ` WHERE ` + strings.Join(where, " AND ")

	var p Page
	if err := s.DB.QueryRow(`SELECT COUNT(*)`+base, args...).Scan(&p.Total); err != nil {
		return p, err
	}

	order := ` ORDER BY f.taken_at DESC, f.rel_path`
	switch {
	case f.Sort == "name":
		order = ` ORDER BY f.rel_path`
	case f.AlbumID > 0:
		order = ` ORDER BY af.position, f.taken_at`
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.DB.Query(`SELECT f.id, f.name, f.rel_path, f.kind, f.size, f.taken_at, f.taken_src, f.hash`+
		base+order+` LIMIT ? OFFSET ?`, append(args, limit, f.Offset)...)
	if err != nil {
		return p, err
	}
	defer rows.Close()
	p.Items = []Item{}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Name, &it.RelPath, &it.Kind, &it.Size, &it.TakenAt, &it.TakenSrc, &it.Hash); err != nil {
			return p, err
		}
		p.Items = append(p.Items, it)
	}
	return p, rows.Err()
}

type DirCount struct {
	Dir   string `json:"dir"`
	Count int    `json:"count"`
}

// Dirs returns every directory that directly contains present files.
func (s *Store) Dirs() ([]DirCount, error) {
	rows, err := s.DB.Query(`SELECT dir, COUNT(*) FROM files WHERE status = 'present' GROUP BY dir ORDER BY dir`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DirCount{}
	for rows.Next() {
		var d DirCount
		if err := rows.Scan(&d.Dir, &d.Count); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

type YearCount struct {
	Year  int `json:"year"`
	Count int `json:"count"`
}

// Years returns media counts per year for files with a real capture date.
func (s *Store) Years() ([]YearCount, error) {
	rows, err := s.DB.Query(`SELECT CAST(strftime('%Y', taken_at, 'unixepoch') AS INTEGER) y, COUNT(*)
		FROM files WHERE status = 'present' AND kind IN ('image', 'video') AND taken_src NOT IN ('', 'mtime')
		GROUP BY y ORDER BY y DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []YearCount{}
	for rows.Next() {
		var y YearCount
		if err := rows.Scan(&y.Year, &y.Count); err != nil {
			return nil, err
		}
		out = append(out, y)
	}
	return out, rows.Err()
}

type Stats struct {
	Files   int `json:"files"`
	Media   int `json:"media"`
	Other   int `json:"other"`
	NoDate  int `json:"noDate"`
	Trashed int `json:"trashed"`
	Missing int `json:"missing"`
}

func (s *Store) Stats() (Stats, error) {
	var st Stats
	err := s.DB.QueryRow(`SELECT
		COALESCE(SUM(status = 'present'), 0),
		COALESCE(SUM(status = 'present' AND kind IN ('image', 'video')), 0),
		COALESCE(SUM(status = 'present' AND kind IN ('other', 'sidecar')), 0),
		COALESCE(SUM(status = 'present' AND kind IN ('image', 'video') AND taken_src IN ('', 'mtime')), 0),
		COALESCE(SUM(status = 'trashed'), 0),
		COALESCE(SUM(status = 'missing'), 0)
		FROM files`).Scan(&st.Files, &st.Media, &st.Other, &st.NoDate, &st.Trashed, &st.Missing)
	return st, err
}

// Duplicates returns groups of present media files sharing the same content hash.
func (s *Store) Duplicates() ([][]Item, error) {
	rows, err := s.DB.Query(`SELECT id, name, rel_path, kind, size, taken_at, taken_src, hash FROM files
		WHERE status = 'present' AND kind IN ('image', 'video') AND hash != '' AND hash IN (
			SELECT hash FROM files WHERE status = 'present' AND kind IN ('image', 'video') AND hash != ''
			GROUP BY hash HAVING COUNT(*) > 1)
		ORDER BY hash, rel_path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := [][]Item{}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Name, &it.RelPath, &it.Kind, &it.Size, &it.TakenAt, &it.TakenSrc, &it.Hash); err != nil {
			return nil, err
		}
		if n := len(groups); n > 0 && groups[n-1][0].Hash == it.Hash {
			groups[n-1] = append(groups[n-1], it)
		} else {
			groups = append(groups, []Item{it})
		}
	}
	return groups, rows.Err()
}
