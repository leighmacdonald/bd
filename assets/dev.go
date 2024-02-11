//go:build !release

package assets

import (
	"net/http"
	"path/filepath"

	"errors"
	"github.com/gin-gonic/gin"
)

var ErrStaticPath = errors.New("failed to setup static paths")

func StaticRoutes(engine *gin.Engine, testing bool) error {
	absStaticPath, errStaticPath := filepath.Abs("./assets/dist")
	if errStaticPath != nil {
		return errors.Join(errStaticPath, ErrStaticPath)
	}

	engine.StaticFS("/dist", http.Dir(absStaticPath))

	if !testing {
		engine.LoadHTMLFiles(filepath.Join(absStaticPath, "index.html"))
	}

	return nil
}
