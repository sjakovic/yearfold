package library

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sjakovic/yearfold/internal/meta"
	"github.com/sjakovic/yearfold/internal/store"
)

func (l *Library) LinkSidecars() error {
	jsons, err := l.St.UnlinkedJSON()
	if err != nil || len(jsons) == 0 {
		return err
	}
	byDir := map[string][]store.NamedFile{}
	for _, j := range jsons {
		byDir[j.Dir] = append(byDir[j.Dir], j)
	}

	var links []store.SidecarLink
	var unmatched []store.NamedFile
	for dir, files := range byDir {
		media, err := l.St.MediaIn(dir)
		if err != nil {
			return err
		}
		candidates := newCandidates(media)
		for _, j := range files {
			if id, ok := candidates.match(j.Name); ok {
				links = append(links, store.SidecarLink{SidecarID: j.ID, MediaID: id})
			} else {
				unmatched = append(unmatched, j)
			}
		}
	}

	across, err := l.linkAcrossParts(unmatched)
	if err != nil {
		return err
	}
	return l.St.LinkSidecars(append(links, across...))
}

// Google splits an export into parts (Takeout-1, Takeout-2, ...) and can put
// a photo and its JSON into different ones, under the same path inside.
func (l *Library) linkAcrossParts(jsons []store.NamedFile) ([]store.SidecarLink, error) {
	if len(jsons) == 0 {
		return nil, nil
	}
	byInnerDir, err := l.mediaByInnerDir()
	if err != nil {
		return nil, err
	}
	var links []store.SidecarLink
	for _, j := range jsons {
		inner := innerDir(j.Dir)
		if inner == "" {
			continue
		}
		id, ok := newCandidates(byInnerDir[inner]).match(j.Name)
		if !ok || !l.isTakeoutSidecar(j) {
			continue
		}
		links = append(links, store.SidecarLink{SidecarID: j.ID, MediaID: id})
	}
	return links, nil
}

// mediaByInnerDir groups media by the folder path below the first segment,
// using both where a file is now and where it was before it was moved.
func (l *Library) mediaByInnerDir() (map[string][]store.NamedFile, error) {
	out := map[string][]store.NamedFile{}
	add := func(id int64, rel string) {
		dir, name, _ := store.SplitPath(rel)
		if inner := innerDir(dir); inner != "" {
			out[inner] = append(out[inner], store.NamedFile{ID: id, Dir: dir, Name: name})
		}
	}
	media, err := l.St.Media()
	if err != nil {
		return nil, err
	}
	for _, m := range media {
		add(m.ID, m.RelPath)
	}
	moved, err := l.St.MovedMedia()
	if err != nil {
		return nil, err
	}
	for _, m := range moved {
		add(m.ID, m.From)
	}
	return out, nil
}

func innerDir(dir string) string {
	_, inner, _ := strings.Cut(dir, "/")
	return inner
}

func (l *Library) isTakeoutSidecar(j store.NamedFile) bool {
	data, err := os.ReadFile(filepath.Join(l.Root, filepath.FromSlash(j.Dir), j.Name))
	if err != nil {
		return false
	}
	_, ok := meta.ParseTakeout(data)
	return ok
}

type candidates struct {
	ids   map[string]int64
	names []string
}

func newCandidates(media []store.NamedFile) candidates {
	c := candidates{ids: make(map[string]int64, len(media))}
	for _, m := range media {
		if _, seen := c.ids[m.Name]; !seen {
			c.ids[m.Name] = m.ID
			c.names = append(c.names, m.Name)
		}
	}
	return c
}

func (c candidates) match(jsonName string) (int64, bool) {
	name, ok := meta.MatchSidecar(jsonName, c.names)
	if !ok {
		return 0, false
	}
	return c.ids[name], true
}
