// Package legacy takes over data written when the app was called ImageManager.
package legacy

import (
	"errors"
	"os"
)

const (
	LibraryDir = ".imagemanager"
	ConfigDir  = "ImageManager"
)

func AdoptDir(oldPath, newPath string) error {
	if _, err := os.Stat(newPath); !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Stat(oldPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return os.Rename(oldPath, newPath)
}
