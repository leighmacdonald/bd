//go:build !release

package frontend

import (
	"log/slog"
	"net/http"
	"path"
	"path/filepath"
)

func AddRoutes(mux *http.ServeMux, root string) error {
	if root == "" {
		root = "frontend/dist"
	}

	absRoot, _ := filepath.Abs(root)

	mux.HandleFunc("GET /", serveTrustedRoot(absRoot))

	return nil
}

// serveTrustedRoot acts as a secure local file server. While most users will only serve the app over localhost or LAN at most,
// This provides an extra security precaution to prevent path traversals if for some reason it was exposed to the internet by
// ensuring files are only served from the provided root folder.
func serveTrustedRoot(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filePath := "." + r.URL.Path
		if filePath == "./" {
			filePath = path.Join(root, "index.html")
		} else {
			filePath = path.Join(root, filePath)
		}

		filePath = path.Clean(filePath)

		if !inRoot(filePath, root) {
			http.Error(w, "Invalid path", http.StatusBadRequest)

			slog.Error("User request file outside of trusted root", slog.String("path", filePath))

			return
		}

		http.ServeFile(w, r, filePath)
	}
}

func inRoot(path string, root string) bool {
	for path != root {
		path = filepath.Dir(path)
		if path == root {
			return true
		}
	}

	return false
}
