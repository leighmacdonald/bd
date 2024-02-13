//go:build release

package frontend

import (
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
)

//go:embed dist/*
var embedFS embed.FS

var ErrEmbedFS = errors.New("failed to load embed.fs path")

func AddRoutes(mux *http.ServeMux, _ string) error {
	subFs, errSubFS := fs.Sub(embedFS, "dist")
	if errSubFS != nil {
		return errors.Join(errSubFS, ErrEmbedFS)
	}

	mux.Handle("GET /", http.StripPrefix("/dist", http.FileServer(http.FS(subFs))))
	mux.HandleFunc("GET /index.html", func(writer http.ResponseWriter, _ *http.Request) {
		index, errIndex := embedFS.ReadFile("index.html")
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
