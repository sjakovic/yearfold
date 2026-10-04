package library

import (
	"errors"
	"os"
	"path/filepath"
)

// The application was called ImageManager before version 0.4.0. Data written
// under the old names is renamed in place the first time it is needed.
const (
	legacyMetaDir   = ".imagemanager"
	legacyConfigDir = "ImageManager"
)

// adoptLegacyDir renames oldPath to newPath when only the old one exists.
func adoptLegacyDir(oldPath, newPath string) error {
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

// adoptLegacyLibrary takes over a library data folder created under the old name.
func adoptLegacyLibrary(root, dir string) error {
	return adoptLegacyDir(filepath.Join(root, legacyMetaDir), dir)
}
