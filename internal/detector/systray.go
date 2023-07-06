package detector

import (
	"fyne.io/systray"
	"github.com/leighmacdonald/bd/internal/platform"
)

type Systray struct {
	launch *systray.MenuItem
	quit   *systray.MenuItem
}

func NewSystray() *Systray {
	return &Systray{
		launch: systray.AddMenuItem("Open BD", "Open BD in your browser"),
		quit:   systray.AddMenuItem("Quit", "Quit the application"),
	}
}

func (s *Systray) onReady() {
	systray.SetIcon(platform.Icon())
	systray.SetTitle("BD")
	systray.SetTooltip("Bot Detector")

}

func (s *Systray) onExit() {

}

func (s *Systray) start() {
	systray.RunWithExternalLoop(s.onReady, s.onExit)
}
