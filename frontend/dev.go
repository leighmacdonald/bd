//go:build !release

package frontend

import (
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
)

func AddRoutes(mux *http.ServeMux) (http.HandlerFunc, error) {
	absStaticPath, errPathInvalid := filepath.Abs("./frontend/dist")
	if errPathInvalid != nil {
		return nil, errors.Join(errPathInvalid, ErrStaticPath)
	}

	indexTmpl := template.Must(template.New("index.html").
		Delims("{{", "}}").
		ParseFiles(filepath.Join(absStaticPath, "index.html")))

	mux.Handle("/dist", http.StripPrefix("/dist", http.FileServer(http.Dir(absStaticPath))))

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			return
		}

		if err := indexTmpl.Execute(w, jsConfig{SiteName: "bd"}); err != nil {
			slog.Error("Failed to exec template", slog.String("error", err.Error()))
			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	}, nil
}
