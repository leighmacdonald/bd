//go:build release

package frontend

import (
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
)

//go:embed dist/*
var embedFS embed.FS

var ErrEmbedFS = errors.New("failed to load embed.fs path")

func AddRoutes(mux *http.ServeMux) (http.HandlerFunc, error) {
	subFs, errSubFS := fs.Sub(embedFS, "dist")
	if errSubFS != nil {
		return nil, errors.Join(errSubFS, ErrEmbedFS)
	}

	indexTmpl := template.Must(template.New("index.html").
		Delims("{{", "}}").
		ParseFS(subFs, "index.html"))

	mux.Handle("GET /dist/", http.StripPrefix("/dist", http.FileServer(http.FS(subFs))))

	return func(w http.ResponseWriter, _ *http.Request) {
		if err := indexTmpl.Execute(w, jsConfig{SiteName: "bd"}); err != nil {
			slog.Error("Failed to exec template", slog.String("error", err.Error()))
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
	}, nil
}
