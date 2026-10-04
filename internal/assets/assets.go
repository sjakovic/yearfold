// Package assets serves thumbnails and original files to the webview.
package assets

import (
	"image/jpeg"
	"net/http"
	"strconv"
	"strings"

	"github.com/sjakovic/yearfold/internal/library"
	"github.com/sjakovic/yearfold/internal/store"
	"github.com/sjakovic/yearfold/internal/thumbs"
)

// previewSize is the longest edge of formats converted for the browser.
const previewSize = 2560

// Handler serves /thumb/{id} and /file/{id} for the currently open library.
func Handler(current func() *library.Library) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		kind, rest, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
		id, err := strconv.ParseInt(rest, 10, 64)
		lib := current()
		if err != nil || lib == nil || (kind != "thumb" && kind != "file") {
			http.NotFound(w, r)
			return
		}
		f, err := lib.St.GetFile(id)
		if err != nil || f.Status == store.StatusMissing {
			http.NotFound(w, r)
			return
		}
		src := lib.Abs(f)
		heic := f.Ext == "heic" || f.Ext == "heif"
		// HEIC decoding already applies the stored rotation.
		orientation := f.Orientation
		if heic {
			orientation = 1
		}

		if kind == "thumb" {
			p, err := lib.Thumbs.Get(f.ID, src, f.Ext, f.Kind, orientation)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Cache-Control", "max-age=3600")
			http.ServeFile(w, r, p)
			return
		}

		// Webviews on Windows and Linux cannot show HEIC, so it is converted.
		if heic {
			img, err := thumbs.Decode(src, f.Ext)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "image/jpeg")
			jpeg.Encode(w, thumbs.Resize(img, previewSize), &jpeg.Options{Quality: 90})
			return
		}
		http.ServeFile(w, r, src)
	})
}
