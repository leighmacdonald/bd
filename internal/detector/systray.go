package detector

import (
	"fyne.io/systray"
)

type Systray struct {
	launch *systray.MenuItem
	quit   *systray.MenuItem
	icon   []byte
}

func NewSystray(icon []byte) *Systray {
	return &Systray{
		launch: systray.AddMenuItem("Open BD", "Open BD in your browser"),
		quit:   systray.AddMenuItem("Quit", "Quit the application"),
		icon:   icon,
	}
}

func (s *Systray) onReady() {
	systray.SetIcon(s.icon)
	systray.SetTitle("BD")
	systray.SetTooltip("Bot Detector")
}

func (s *Systray) onExit() {
}

func (s *Systray) start() {
	systray.RunWithExternalLoop(s.onReady, s.onExit)
}
