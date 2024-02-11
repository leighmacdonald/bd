package main

import (
	"context"
	"log/slog"

	"fyne.io/systray"
)

type Systray struct {
	icon     []byte
	onOpen   func()
	onLaunch func()
	quit     *systray.MenuItem
}

func NewSystray(icon []byte, onOpen func(), onLaunch func()) *Systray {
	tray := &Systray{
		icon:     icon,
		onOpen:   onOpen,
		onLaunch: onLaunch,
	}

	return tray
}

func (s *Systray) OnReady(cancel context.CancelFunc) func() {
	return func() {
		systray.SetIcon(s.icon)
		systray.SetTitle("BD")
		systray.SetTooltip("Bot Detector")

		go func() {
			openWeb := systray.AddMenuItem("Open BD", "Open BD in your browser")
			openWeb.SetIcon(s.icon)
			openWeb.Enable()

			launchGame := systray.AddMenuItem("Launch TF2", "Launch Team Fortress 2")
			launchGame.SetIcon(s.icon)
			launchGame.Enable()

			systray.AddSeparator()

			s.quit = systray.AddMenuItem("Quit", "Quit the application")
			s.quit.Enable()

			for {
				select {
				case <-launchGame.ClickedCh:
					slog.Debug("launchGame clicked")
					go s.onLaunch()
				case <-openWeb.ClickedCh:
					slog.Debug("openWeb Clicked")
					s.onOpen()
				case <-s.quit.ClickedCh:
					slog.Debug("User Quit")
					cancel()
				}
			}
		}()
	}
}
