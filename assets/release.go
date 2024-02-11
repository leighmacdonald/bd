//go:build release

package assets

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	"errors"
	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var embedFS embed.FS

var ErrEmbedFS = errors.New("failed to load embed.fs path")

func StaticRoutes(engine *gin.Engine, _ bool) error {
	subFs, errSubFS := fs.Sub(embedFS, "dist")
	if errSubFS != nil {
		return errors.Join(errSubFS, ErrEmbedFS)
	}

	engine.SetHTMLTemplate(template.
		Must(template.New("").
			Delims("{{", "}}").
			Funcs(engine.FuncMap).
			ParseFS(subFs, "index.html")))
	engine.StaticFS("/dist", http.FS(subFs))

	return nil
}
