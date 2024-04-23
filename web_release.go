// When the binary is compiled in release mode, assets are embedded and served from the binary directly. This expects
// a ./dist directory containing the frontend assets. This can be created by running the `make frontend`. This is
// handled automatically if building with goreleaser via `make snapshot`.
//go:build release

package main

import (
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
)

//go:embed dist
var embedFS embed.FS

var ErrEmbedFS = errors.New("failed to load embed.fs path")

func AddRoutes(mux *http.ServeMux, _ string) error {
	subFs, errSubFS := fs.Sub(embedFS, "dist")
	if errSubFS != nil {
		return errors.Join(errSubFS, ErrEmbedFS)
	}

	mux.Handle("GET /assets/", http.FileServer(http.FS(subFs)))
	mux.HandleFunc("GET /", func(writer http.ResponseWriter, _ *http.Request) {
		index, errIndex := embedFS.ReadFile("dist/index.html")
		if errIndex != nil {
			slog.Error("failed to open index.html", slog.String("error", errIndex.Error()))

			http.Error(writer, "", http.StatusInternalServerError)

			return
		}

		_, err := writer.Write(index)
		if err != nil {
			slog.Error("Failed to write index response", slog.String("error", errIndex.Error()))
		}
	})

	return nil
}
