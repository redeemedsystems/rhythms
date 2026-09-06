package server

import (
	"io/fs"
	"net/http"
)

// serveEmbedded serves one file straight out of the embedded web FS with an
// explicit content type, used for files (manifest.json, sw.js) that need to
// be reachable at the site root rather than under /static/.
func serveEmbedded(fsys fs.FS, path, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(data)
	}
}
