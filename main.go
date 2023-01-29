package main

import (
	"context"
	_ "github.com/leighmacdonald/bd/translations"
	"github.com/leighmacdonald/bd/ui"
)

func main() {
	ctx := context.Background()
	rc := newRconConfig(true)
	gui := ui.New(ctx)
	botDetector := New(ctx, rc)
	botDetector.AttachGui(gui)
	go botDetector.start()
	gui.Start()
}
