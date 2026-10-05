package store

import (
	"database/sql"
	"errors"
)

type NamedFile struct {
	ID   int64
	Dir  string
	Name string
}

func (s *Store) UnlinkedJSON() ([]NamedFile, error) {
	return s.queryNamed(`SELECT id, dir, name FROM files
		WHERE status = 'present' AND ext = 'json' AND sidecar_of IS NULL ORDER BY dir`)
}

func (s *Store) MediaIn(dir string) ([]NamedFile, error) {
	return s.queryNamed(`SELECT id, dir, name FROM files
		WHERE dir = ? AND status = 'present' AND kind IN ('image', 'video')`, dir)
}

func (s *Store) queryNamed(q string, args ...any) ([]NamedFile, error) {
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NamedFile
	for rows.Next() {
		var f NamedFile
		if err := rows.Scan(&f.ID, &f.Dir, &f.Name); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) PresentID(dir, name string) (id int64, found bool, err error) {
	err = s.db.QueryRow(`SELECT id FROM files WHERE dir = ? AND name = ? AND status = 'present'`, dir, name).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return id, err == nil, err
}

type SidecarLink struct {
	SidecarID int64
	MediaID   int64
}

func (s *Store) LinkSidecars(links []SidecarLink) error {
	if len(links) == 0 {
		return nil
	}
	return s.inTx(func(tx *sql.Tx) error {
		for _, l := range links {
			if _, err := tx.Exec(`UPDATE files SET sidecar_of = ?, kind = 'sidecar' WHERE id = ?`,
				l.MediaID, l.SidecarID); err != nil {
				return err
			}
			if _, err := tx.Exec(`UPDATE files SET meta_done = 0 WHERE id = ?`, l.MediaID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) MergeableSidecars() ([]SidecarLink, error) {
	rows, err := s.db.Query(`SELECT s.id, m.id FROM files s JOIN files m ON m.id = s.sidecar_of
		WHERE s.status = 'present' AND m.status = 'present' AND m.meta_done = 1 AND m.takeout_json != ''
		ORDER BY m.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SidecarLink
	for rows.Next() {
		var l SidecarLink
		if err := rows.Scan(&l.SidecarID, &l.MediaID); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) PendingMeta() ([]File, error) {
	return s.queryPending(`meta_done = 0 AND kind IN ('image', 'video')`)
}

func (s *Store) PendingHash() ([]File, error) {
	return s.queryPending(`hash = ''`)
}

func (s *Store) queryPending(cond string) ([]File, error) {
	rows, err := s.db.Query(`SELECT id, rel_path, dir, name, ext, kind, size, mtime, date_override FROM files
		WHERE status = 'present' AND ` + cond + ` ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []File
	for rows.Next() {
		f := File{Status: StatusPresent}
		if err := rows.Scan(&f.ID, &f.RelPath, &f.Dir, &f.Name, &f.Ext, &f.Kind, &f.Size, &f.Mtime,
			&f.DateOverride); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

type MetaUpdate struct {
	ID          int64
	TakenAt     int64
	TakenSrc    string
	Width       int
	Height      int
	Orientation int
	Camera      string
	HasGPS      bool
	Lat, Lon    float64
	MetaJSON    string
	TakeoutJSON string
}

func (s *Store) SaveMeta(updates []MetaUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	return s.inTx(func(tx *sql.Tx) error {
		for _, u := range updates {
			if _, err := tx.Exec(`UPDATE files SET taken_at = ?, taken_src = ?, width = ?, height = ?,
				orientation = ?, camera = ?, has_gps = ?, lat = ?, lon = ?, meta_json = ?, meta_done = 1,
				takeout_json = CASE WHEN ? != '' THEN ? ELSE takeout_json END WHERE id = ?`,
				u.TakenAt, u.TakenSrc, u.Width, u.Height, u.Orientation, u.Camera, u.HasGPS, u.Lat, u.Lon,
				u.MetaJSON, u.TakeoutJSON, u.TakeoutJSON, u.ID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) RequeueMeta(ids ...int64) error {
	return s.inTx(func(tx *sql.Tx) error {
		for _, id := range ids {
			if _, err := tx.Exec(`UPDATE files SET meta_done = 0 WHERE id = ?`, id); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) TakeoutJSON(id int64) (string, error) {
	var raw string
	err := s.db.QueryRow(`SELECT takeout_json FROM files WHERE id = ?`, id).Scan(&raw)
	return raw, err
}

func (s *Store) SetHash(id int64, hash string, size, mtime int64) error {
	_, err := s.db.Exec(`UPDATE files SET hash = ? WHERE id = ? AND size = ? AND mtime = ?`, hash, id, size, mtime)
	return err
}

func (s *Store) SetDateOverride(id, unix int64) error {
	_, err := s.db.Exec(`UPDATE files SET date_override = ?, taken_at = ?, taken_src = ? WHERE id = ?`,
		unix, unix, SrcManual, id)
	return err
}

func (s *Store) MarkRewritten(id, size, mtime int64) error {
	_, err := s.db.Exec(`UPDATE files SET size = ?, mtime = ?, hash = '', meta_done = 0, date_override = 0
		WHERE id = ?`, size, mtime, id)
	return err
}
