package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

type Tag struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func (s *Store) Tags() ([]Tag, error) {
	return s.queryTags(`SELECT t.id, t.name, (SELECT COUNT(*) FROM file_tags ft JOIN files f ON f.id = ft.file_id
		WHERE ft.tag_id = t.id AND f.status = 'present') FROM tags t ORDER BY t.name COLLATE NOCASE`)
}

func (s *Store) FileTags(fileID int64) ([]Tag, error) {
	return s.queryTags(`SELECT t.id, t.name, 0 FROM tags t JOIN file_tags ft ON ft.tag_id = t.id
		WHERE ft.file_id = ? ORDER BY t.name COLLATE NOCASE`, fileID)
}

func (s *Store) queryTags(q string, args ...any) ([]Tag, error) {
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Tag{}
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// AddTag attaches the named tag (created if needed) to the given files.
func (s *Store) AddTag(fileIDs []int64, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("tag name is empty")
	}
	return s.InTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO tags (name) VALUES (?)`, name); err != nil {
			return err
		}
		var tagID int64
		if err := tx.QueryRow(`SELECT id FROM tags WHERE name = ?`, name).Scan(&tagID); err != nil {
			return err
		}
		for _, id := range fileIDs {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO file_tags (file_id, tag_id) VALUES (?, ?)`, id, tagID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) RemoveTag(fileIDs []int64, tagID int64) error {
	return s.InTx(func(tx *sql.Tx) error {
		for _, id := range fileIDs {
			if _, err := tx.Exec(`DELETE FROM file_tags WHERE file_id = ? AND tag_id = ?`, id, tagID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) DeleteTag(tagID int64) error {
	_, err := s.DB.Exec(`DELETE FROM tags WHERE id = ?`, tagID)
	return err
}

type Album struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func (s *Store) Albums() ([]Album, error) {
	return s.queryAlbums(`SELECT a.id, a.name, (SELECT COUNT(*) FROM album_files af JOIN files f ON f.id = af.file_id
		WHERE af.album_id = a.id AND f.status = 'present') FROM albums a ORDER BY a.name COLLATE NOCASE`)
}

func (s *Store) FileAlbums(fileID int64) ([]Album, error) {
	return s.queryAlbums(`SELECT a.id, a.name, 0 FROM albums a JOIN album_files af ON af.album_id = a.id
		WHERE af.file_id = ? ORDER BY a.name COLLATE NOCASE`, fileID)
}

func (s *Store) queryAlbums(q string, args ...any) ([]Album, error) {
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Album{}
	for rows.Next() {
		var a Album
		if err := rows.Scan(&a.ID, &a.Name, &a.Count); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) CreateAlbum(name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, errors.New("album name is empty")
	}
	res, err := s.DB.Exec(`INSERT INTO albums (name, created_at) VALUES (?, ?)`, name, time.Now().Unix())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) RenameAlbum(id int64, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("album name is empty")
	}
	_, err := s.DB.Exec(`UPDATE albums SET name = ? WHERE id = ?`, name, id)
	return err
}

func (s *Store) DeleteAlbum(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM albums WHERE id = ?`, id)
	return err
}

// AddToAlbum appends files to an album; files already in it keep their position.
func (s *Store) AddToAlbum(albumID int64, fileIDs []int64) error {
	return s.InTx(func(tx *sql.Tx) error {
		var pos int
		if err := tx.QueryRow(`SELECT COALESCE(MAX(position), 0) FROM album_files WHERE album_id = ?`, albumID).Scan(&pos); err != nil {
			return err
		}
		for _, id := range fileIDs {
			pos++
			if _, err := tx.Exec(`INSERT OR IGNORE INTO album_files (album_id, file_id, position) VALUES (?, ?, ?)`, albumID, id, pos); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) RemoveFromAlbum(albumID int64, fileIDs []int64) error {
	return s.InTx(func(tx *sql.Tx) error {
		for _, id := range fileIDs {
			if _, err := tx.Exec(`DELETE FROM album_files WHERE album_id = ? AND file_id = ?`, albumID, id); err != nil {
				return err
			}
		}
		return nil
	})
}

// MediaInDir returns ids of present media files directly inside dir.
func (s *Store) MediaInDir(dir string) ([]int64, error) {
	rows, err := s.DB.Query(`SELECT id FROM files WHERE dir = ? AND status = 'present' AND kind IN ('image', 'video')
		ORDER BY taken_at, name`, dir)
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
