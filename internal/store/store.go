// Package store is the SQLite-backed index of a library: files, tags, albums
// and the operations journal. All paths are slash-separated and relative to
// the library root.
package store

import (
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const (
	StatusPresent = "present"
	StatusMissing = "missing"
	StatusTrashed = "trashed"

	KindImage   = "image"
	KindVideo   = "video"
	KindSidecar = "sidecar"
	KindOther   = "other"

	// SrcMtime marks a taken_at that is only the file modification time.
	SrcMtime   = "mtime"
	SrcExif    = "exif"
	SrcTakeout = "takeout"
	// SrcManual marks a date the user set that lives only in the index,
	// because the file format cannot hold it.
	SrcManual = "manual"
)

type Store struct {
	DB *sql.DB
}

var migrations = []string{
	`CREATE TABLE files (
		id          INTEGER PRIMARY KEY,
		rel_path    TEXT NOT NULL,
		dir         TEXT NOT NULL,
		name        TEXT NOT NULL,
		ext         TEXT NOT NULL,
		kind        TEXT NOT NULL,
		size        INTEGER NOT NULL,
		mtime       INTEGER NOT NULL,
		hash        TEXT NOT NULL DEFAULT '',
		taken_at    INTEGER NOT NULL DEFAULT 0,
		taken_src   TEXT NOT NULL DEFAULT '',
		width       INTEGER NOT NULL DEFAULT 0,
		height      INTEGER NOT NULL DEFAULT 0,
		orientation INTEGER NOT NULL DEFAULT 1,
		camera      TEXT NOT NULL DEFAULT '',
		has_gps     INTEGER NOT NULL DEFAULT 0,
		lat         REAL NOT NULL DEFAULT 0,
		lon         REAL NOT NULL DEFAULT 0,
		meta_json   TEXT NOT NULL DEFAULT '',
		meta_done   INTEGER NOT NULL DEFAULT 0,
		sidecar_of  INTEGER REFERENCES files(id) ON DELETE SET NULL,
		status      TEXT NOT NULL DEFAULT 'present',
		trash_path  TEXT NOT NULL DEFAULT ''
	);
	CREATE UNIQUE INDEX files_path ON files(rel_path) WHERE status = 'present';
	CREATE INDEX files_dir ON files(dir);
	CREATE INDEX files_hash ON files(hash);
	CREATE INDEX files_taken ON files(taken_at);
	CREATE INDEX files_sidecar ON files(sidecar_of);
	CREATE INDEX files_status ON files(status);

	CREATE TABLE tags (
		id   INTEGER PRIMARY KEY,
		name TEXT NOT NULL UNIQUE COLLATE NOCASE
	);
	CREATE TABLE file_tags (
		file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
		tag_id  INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
		PRIMARY KEY (file_id, tag_id)
	) WITHOUT ROWID;

	CREATE TABLE albums (
		id         INTEGER PRIMARY KEY,
		name       TEXT NOT NULL,
		created_at INTEGER NOT NULL
	);
	CREATE TABLE album_files (
		album_id INTEGER NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
		file_id  INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
		position INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (album_id, file_id)
	) WITHOUT ROWID;

	CREATE TABLE ops (
		id         INTEGER PRIMARY KEY,
		batch      INTEGER NOT NULL,
		type       TEXT NOT NULL,
		file_id    INTEGER NOT NULL,
		from_path  TEXT NOT NULL,
		to_path    TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		undone     INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX ops_batch ON ops(batch);`,

	// EXIF dates used to be read in the computer's time zone; read them again
	// so they are stored as the camera's wall-clock time.
	`UPDATE files SET meta_done = 0 WHERE taken_src = 'exif';`,

	`ALTER TABLE files ADD COLUMN date_override INTEGER NOT NULL DEFAULT 0;`,

	// Keep a copy of the Takeout sidecar data in the index, and read the files
	// that have a sidecar again so the copy gets filled in.
	`ALTER TABLE files ADD COLUMN takeout_json TEXT NOT NULL DEFAULT '';
	UPDATE files SET meta_done = 0 WHERE taken_src = 'takeout'
		OR id IN (SELECT sidecar_of FROM files WHERE sidecar_of IS NOT NULL);`,
}

// Open opens (creating and migrating if needed) the database at path.
func Open(path string) (*Store, error) {
	p := filepath.ToSlash(path)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	q := url.Values{}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "busy_timeout(10000)")
	q.Add("_pragma", "synchronous(NORMAL)")
	dsn := (&url.URL{Scheme: "file", Path: p, RawQuery: q.Encode()}).String()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	s := &Store{DB: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func (s *Store) migrate() error {
	var v int
	if err := s.DB.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return err
	}
	for ; v < len(migrations); v++ {
		tx, err := s.DB.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migrations[v]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", v+1, err)
		}
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, v+1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// InTx runs fn inside a transaction.
func (s *Store) InTx(fn func(tx *sql.Tx) error) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// SplitPath splits a slash-separated relative path into dir, name and
// lower-cased extension (without the dot).
func SplitPath(rel string) (dir, name, ext string) {
	name = rel
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		dir, name = rel[:i], rel[i+1:]
	}
	if i := strings.LastIndex(name, "."); i > 0 {
		ext = strings.ToLower(name[i+1:])
	}
	return dir, name, ext
}

var imageExts = map[string]bool{
	"jpg": true, "jpeg": true, "png": true, "gif": true, "webp": true, "heic": true,
	"heif": true, "bmp": true, "tif": true, "tiff": true, "avif": true, "dng": true,
	"cr2": true, "nef": true, "arw": true, "raw": true,
}

var videoExts = map[string]bool{
	"mp4": true, "mov": true, "avi": true, "mkv": true, "m4v": true, "3gp": true,
	"webm": true, "mpg": true, "mpeg": true, "mts": true, "wmv": true,
}

// KindOf classifies a file by its extension.
func KindOf(ext string) string {
	switch {
	case imageExts[ext]:
		return KindImage
	case videoExts[ext]:
		return KindVideo
	}
	return KindOther
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
