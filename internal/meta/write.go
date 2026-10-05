package meta

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

var ErrCannotEmbed = errors.New("the date cannot be stored inside this file type")

const exifStampLayout = "2006:01:02 15:04:05"

func CanEmbedDate(ext string) bool {
	switch ext {
	case "jpg", "jpeg", "png":
		return true
	}
	return false
}

func WriteDate(path, ext string, t time.Time) error {
	stamp := t.Format(exifStampLayout)
	var set func(data []byte, stamp string) ([]byte, error)
	switch ext {
	case "jpg", "jpeg":
		set = setJPEGDate
	case "png":
		set = setPNGDate
	default:
		return writeDateWithExiftool(path, stamp)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	updated, err := set(data, stamp)
	if err != nil {
		return err
	}
	return replaceFile(path, updated)
}

func replaceFile(path string, data []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".yearfold-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), info.Mode().Perm()); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func writeDateWithExiftool(path, stamp string) error {
	exiftool, err := exec.LookPath("exiftool")
	if err != nil {
		return ErrCannotEmbed
	}
	out, err := exec.Command(exiftool, "-overwrite_original",
		"-DateTimeOriginal="+stamp, "-CreateDate="+stamp, path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("exiftool: %s", out)
	}
	return nil
}
