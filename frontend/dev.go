//go:build !release

package frontend

import (
	"log/slog"
	"net/http"
)

func AddRoutes(_ *http.ServeMux, _ string) error {
	slog.Info("Use vite server for dev `make serve-ts`")

	return nil
}
