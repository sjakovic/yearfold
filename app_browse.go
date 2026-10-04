package main

import (
	"github.com/sjakovic/yearfold/internal/store"
)

func (a *App) List(f store.Filter) (store.Page, error) {
	lib, err := a.library()
	if err != nil {
		return store.Page{Items: []store.Item{}}, err
	}
	return lib.St.List(f)
}

type Overview struct {
	Stats  store.Stats       `json:"stats"`
	Dirs   []store.DirCount  `json:"dirs"`
	Years  []store.YearCount `json:"years"`
	Tags   []store.Tag       `json:"tags"`
	Albums []store.Album     `json:"albums"`
}

// GetOverview returns everything the sidebar shows.
func (a *App) GetOverview() (Overview, error) {
	var o Overview
	lib, err := a.library()
	if err != nil {
		return o, err
	}
	if o.Stats, err = lib.St.Stats(); err != nil {
		return o, err
	}
	if o.Dirs, err = lib.St.Dirs(); err != nil {
		return o, err
	}
	if o.Years, err = lib.St.Years(); err != nil {
		return o, err
	}
	if o.Tags, err = lib.St.Tags(); err != nil {
		return o, err
	}
	o.Albums, err = lib.St.Albums()
	return o, err
}

type Detail struct {
	File     store.File    `json:"file"`
	AbsPath  string        `json:"absPath"`
	Tags     []store.Tag   `json:"tags"`
	Albums   []store.Album `json:"albums"`
	Sidecars []string      `json:"sidecars"`
}

func (a *App) GetDetail(id int64) (Detail, error) {
	var d Detail
	lib, err := a.library()
	if err != nil {
		return d, err
	}
	if d.File, err = lib.St.GetFile(id); err != nil {
		return d, err
	}
	d.AbsPath = lib.Abs(d.File)
	if d.Tags, err = lib.St.FileTags(id); err != nil {
		return d, err
	}
	if d.Albums, err = lib.St.FileAlbums(id); err != nil {
		return d, err
	}
	d.Sidecars = []string{}
	sc, err := lib.St.Sidecars(id)
	for _, s := range sc {
		d.Sidecars = append(d.Sidecars, s.RelPath)
	}
	return d, err
}
