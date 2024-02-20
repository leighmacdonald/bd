//go:build !release

package frontend

import (
	"log/slog"
	"net/http"
)

func AddRoutes(mux *http.ServeMux, root string) error {
	slog.Info("Use vite server for dev `make serve-ts`")

	return nil
}
