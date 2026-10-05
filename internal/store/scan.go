package store

import "database/sql"

type Indexed struct {
	ID      int64
	RelPath string
	Size    int64
	Mtime   int64
	Hash    string
	Status  string
}

func (s *Store) IndexedFiles() ([]Indexed, error) {
	rows, err := s.db.Query(`SELECT id, rel_path, size, mtime, hash, status FROM files
		WHERE status IN ('present', 'missing') ORDER BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Indexed
	for rows.Next() {
		var f Indexed
		if err := rows.Scan(&f.ID, &f.RelPath, &f.Size, &f.Mtime, &f.Hash, &f.Status); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

type FileStat struct {
	RelPath string
	Size    int64
	Mtime   int64
}

type KnownStat struct {
	ID int64
	FileStat
}

type ScanUpdate struct {
	Missing []int64
	Moved   []KnownStat
	Changed []KnownStat
	New     []FileStat
}

func (s *Store) ApplyScan(u ScanUpdate) error {
	return s.inTx(func(tx *sql.Tx) error {
		for _, id := range u.Missing {
			if _, err := tx.Exec(`UPDATE files SET status = 'missing' WHERE id = ?`, id); err != nil {
				return err
			}
		}
		for _, m := range u.Moved {
			if _, err := tx.Exec(`UPDATE files SET status = 'missing' WHERE id = ?`, m.ID); err != nil {
				return err
			}
		}
		for _, m := range u.Moved {
			if err := setPath(tx, m.ID, m.RelPath); err != nil {
				return err
			}
			if _, err := tx.Exec(`UPDATE files SET status = 'present', size = ?, mtime = ? WHERE id = ?`,
				m.Size, m.Mtime, m.ID); err != nil {
				return err
			}
		}
		for _, c := range u.Changed {
			if _, err := tx.Exec(`UPDATE files SET status = 'present', size = ?, mtime = ?, hash = '', meta_done = 0
				WHERE id = ?`, c.Size, c.Mtime, c.ID); err != nil {
				return err
			}
		}
		for _, n := range u.New {
			if _, err := insertFile(tx, n.RelPath, n.Size, n.Mtime); err != nil {
				return err
			}
		}
		return nil
	})
}
