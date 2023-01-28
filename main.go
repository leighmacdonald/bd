package main

import (
	_ "github.com/leighmacdonald/bd/translations"
)

// Embedded via goreleaser
var (
	// Current Git tag or snapshot name
	version = "dev"
	// Current git commit SHA
	commit = "none"
	// Date in the RFC3339 format
	date = "unknown"
)

func main() {
	botDetector := New()
	botDetector.start()
}
