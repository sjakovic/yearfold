package main

import (
	"errors"
	"path"
	"path/filepath"
)

func (a *App) AddTag(ids []int64, name string) error {
	lib, err := a.library()
	if err != nil {
		return err
	}
	return lib.St.AddTag(ids, name)
}

func (a *App) RemoveTag(ids []int64, tagID int64) error {
	lib, err := a.library()
	if err != nil {
		return err
	}
	return lib.St.RemoveTag(ids, tagID)
}

func (a *App) DeleteTag(tagID int64) error {
	lib, err := a.library()
	if err != nil {
		return err
	}
	return lib.St.DeleteTag(tagID)
}

func (a *App) CreateAlbum(name string) (int64, error) {
	lib, err := a.library()
	if err != nil {
		return 0, err
	}
	return lib.St.CreateAlbum(name)
}

func (a *App) RenameAlbum(id int64, name string) error {
	lib, err := a.library()
	if err != nil {
		return err
	}
	return lib.St.RenameAlbum(id, name)
}

func (a *App) DeleteAlbum(id int64) error {
	lib, err := a.library()
	if err != nil {
		return err
	}
	return lib.St.DeleteAlbum(id)
}

func (a *App) AddToAlbum(albumID int64, ids []int64) error {
	lib, err := a.library()
	if err != nil {
		return err
	}
	return lib.St.AddToAlbum(albumID, ids)
}

func (a *App) RemoveFromAlbum(albumID int64, ids []int64) error {
	lib, err := a.library()
	if err != nil {
		return err
	}
	return lib.St.RemoveFromAlbum(albumID, ids)
}

// AlbumFromDir creates an album named after dir holding its media, without
// touching the files. Handy for Takeout album folders.
func (a *App) AlbumFromDir(dir string) (int64, error) {
	lib, err := a.library()
	if err != nil {
		return 0, err
	}
	ids, err := lib.St.MediaInDir(dir)
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, errors.New("folder has no photos or videos")
	}
	name := path.Base(dir)
	if dir == "" {
		name = filepath.Base(lib.Root)
	}
	id, err := lib.St.CreateAlbum(name)
	if err != nil {
		return 0, err
	}
	return id, lib.St.AddToAlbum(id, ids)
}
