package store

import (
	"database/sql"
	"errors"
	"time"
)

func (s *Store) NextBatch() (int64, error) {
	var b int64
	err := s.db.QueryRow(`SELECT COALESCE(MAX(batch), 0) + 1 FROM ops`).Scan(&b)
	return b, err
}

func journal(tx *sql.Tx, batch int64, typ string, fileID int64, from, to string) error {
	_, err := tx.Exec(`INSERT INTO ops (batch, type, file_id, from_path, to_path, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		batch, typ, fileID, from, to, time.Now().Unix())
	return err
}

func (s *Store) RecordMove(batch, fileID int64, from, to string) error {
	return s.inTx(func(tx *sql.Tx) error {
		if err := setPath(tx, fileID, to); err != nil {
			return err
		}
		return journal(tx, batch, OpMove, fileID, from, to)
	})
}

func (s *Store) Relocate(fileID int64, to string) error {
	return s.inTx(func(tx *sql.Tx) error { return setPath(tx, fileID, to) })
}

func (s *Store) RecordTrash(batch, fileID int64, relPath, trashName string) error {
	return s.inTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`UPDATE files SET status = 'trashed', trash_path = ? WHERE id = ?`,
			trashName, fileID); err != nil {
			return err
		}
		return journal(tx, batch, OpTrash, fileID, relPath, trashName)
	})
}

func (s *Store) RecordRestore(fileID int64, relPath string) error {
	return s.inTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`UPDATE files SET status = 'present', trash_path = '' WHERE id = ?`, fileID); err != nil {
			return err
		}
		return setPath(tx, fileID, relPath)
	})
}

func (s *Store) TrashedSidecars(mediaID int64) ([]int64, error) {
	rows, err := s.db.Query(`SELECT id FROM files WHERE sidecar_of = ? AND status = 'trashed'`, mediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

type TrashedFile struct {
	ID        int64
	TrashPath string
}

func (s *Store) Trashed() ([]TrashedFile, error) {
	rows, err := s.db.Query(`SELECT id, trash_path FROM files WHERE status = 'trashed'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrashedFile
	for rows.Next() {
		var f TrashedFile
		if err := rows.Scan(&f.ID, &f.TrashPath); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) DeleteFile(id int64) error {
	_, err := s.db.Exec(`DELETE FROM files WHERE id = ?`, id)
	return err
}

type Op struct {
	ID     int64
	FileID int64
	Type   string
	From   string
	To     string
}

func (s *Store) LastBatch() ([]Op, error) {
	var batch int64
	err := s.db.QueryRow(`SELECT batch FROM ops WHERE undone = 0 ORDER BY id DESC LIMIT 1`).Scan(&batch)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT id, type, file_id, from_path, to_path FROM ops
		WHERE batch = ? AND undone = 0 ORDER BY id DESC`, batch)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ops []Op
	for rows.Next() {
		var o Op
		if err := rows.Scan(&o.ID, &o.Type, &o.FileID, &o.From, &o.To); err != nil {
			return nil, err
		}
		ops = append(ops, o)
	}
	return ops, rows.Err()
}

func (s *Store) MarkUndone(opID int64) error {
	_, err := s.db.Exec(`UPDATE ops SET undone = 1 WHERE id = ?`, opID)
	return err
}

type PathPair struct {
	From string
	To   string
}

func (s *Store) MoveHistory() ([]PathPair, error) {
	rows, err := s.db.Query(`SELECT from_path, to_path FROM ops WHERE type = 'move' AND undone = 0 ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PathPair
	for rows.Next() {
		var p PathPair
		if err := rows.Scan(&p.From, &p.To); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type MovedFile struct {
	ID   int64
	From string
}

func (s *Store) MovedMedia() ([]MovedFile, error) {
	rows, err := s.db.Query(`SELECT o.file_id, o.from_path FROM ops o JOIN files f ON f.id = o.file_id
		WHERE o.type = 'move' AND o.undone = 0 AND f.status = 'present' AND f.kind IN ('image', 'video')
		ORDER BY o.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MovedFile
	for rows.Next() {
		var m MovedFile
		if err := rows.Scan(&m.ID, &m.From); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) RenameDir(batch int64, oldDir, newDir string) (int, error) {
	n := 0
	err := s.inTx(func(tx *sql.Tx) error {
		rows, err := tx.Query(`SELECT id, rel_path FROM files
			WHERE status = 'present' AND (dir = ? OR dir LIKE ? ESCAPE '\')`, oldDir, escapeLike(oldDir)+"/%")
		if err != nil {
			return err
		}
		type entry struct {
			id  int64
			rel string
		}
		var files []entry
		for rows.Next() {
			var e entry
			if err := rows.Scan(&e.id, &e.rel); err != nil {
				rows.Close()
				return err
			}
			files = append(files, e)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for _, f := range files {
			to := newDir + f.rel[len(oldDir):]
			if err := setPath(tx, f.id, to); err != nil {
				return err
			}
			if err := journal(tx, batch, OpMove, f.id, f.rel, to); err != nil {
				return err
			}
		}
		n = len(files)
		return nil
	})
	return n, err
}

func (s *Store) DirExists(dir string) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM files WHERE status = 'present' AND (dir = ? OR dir LIKE ? ESCAPE '\')`,
		dir, escapeLike(dir)+"/%").Scan(&n)
	return n > 0, err
}
