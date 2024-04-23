// When not running in release mode, its expected that you are using vite's built in http serve functionality so that
// things like hot-reloading work properly.
//go:build !release

package main

import (
	"log/slog"
	"net/http"
)

func AddRoutes(_ *http.ServeMux, _ string) error {
	slog.Info("Use vite server for dev `make serve-ts`")

	return nil
}
