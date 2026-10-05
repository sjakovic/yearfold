package store

type DatedMedia struct {
	ID       int64
	RelPath  string
	Dir      string
	TakenAt  int64
	TakenSrc string
}

func (m DatedMedia) HasDate() bool {
	return m.TakenSrc != "" && m.TakenSrc != SrcMtime
}

func (s *Store) Media() ([]DatedMedia, error) {
	rows, err := s.db.Query(`SELECT id, rel_path, dir, taken_at, taken_src FROM files
		WHERE status = 'present' AND kind IN ('image', 'video') ORDER BY rel_path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DatedMedia
	for rows.Next() {
		var m DatedMedia
		if err := rows.Scan(&m.ID, &m.RelPath, &m.Dir, &m.TakenAt, &m.TakenSrc); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

type ExportFile struct {
	Path    string   `json:"path"`
	TakenAt int64    `json:"takenAt,omitempty"`
	Hash    string   `json:"hash,omitempty"`
	Tags    []string `json:"tags,omitempty"`
	Albums  []string `json:"albums,omitempty"`
}

func (s *Store) Export() ([]ExportFile, error) {
	tags, err := s.labelsByFile(`SELECT ft.file_id, t.name FROM file_tags ft JOIN tags t ON t.id = ft.tag_id
		ORDER BY t.name`)
	if err != nil {
		return nil, err
	}
	albums, err := s.labelsByFile(`SELECT af.file_id, a.name FROM album_files af JOIN albums a ON a.id = af.album_id
		ORDER BY a.name`)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(`SELECT id, rel_path, taken_at, taken_src, hash FROM files
		WHERE status = 'present' ORDER BY rel_path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	files := []ExportFile{}
	for rows.Next() {
		var (
			id  int64
			src string
			f   ExportFile
		)
		if err := rows.Scan(&id, &f.Path, &f.TakenAt, &src, &f.Hash); err != nil {
			return nil, err
		}
		if src == "" || src == SrcMtime {
			f.TakenAt = 0
		}
		f.Tags, f.Albums = tags[id], albums[id]
		files = append(files, f)
	}
	return files, rows.Err()
}

func (s *Store) labelsByFile(query string) (map[int64][]string, error) {
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][]string{}
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[id] = append(out[id], name)
	}
	return out, rows.Err()
}
