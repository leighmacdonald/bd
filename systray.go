package main

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/systray"
	"github.com/leighmacdonald/bd/platform"
)

// appSystray provides a interface for the creation and control of a systray icon. The base functionality
// is provided by the fyne systray package. Some linux systems may not properly support how this is implemented
// due to the varying systray standards.
type appSystray struct {
	platform    platform.Platform
	settingsMgr configManager
	process     *processState
	quit        *systray.MenuItem
}

func newAppSystray(platform platform.Platform, settingsMgr configManager, process *processState) *appSystray {
	return &appSystray{
		platform:    platform,
		settingsMgr: settingsMgr,
		process:     process,
	}
}

func (s *appSystray) onOpen(ctx context.Context) {
	settings, errSettings := s.settingsMgr.settings(ctx)
	if errSettings != nil {
		slog.Error("Failed to read settings", errAttr(errSettings))
		return
	}

	if errOpen := s.platform.OpenURL(settings.AppURL()); errOpen != nil {
		slog.Error("Failed to open browser", errAttr(errOpen))
	}
}

func (s *appSystray) onLaunch(ctx context.Context) {
	settings, errSettings := s.settingsMgr.settings(ctx)
	if errSettings != nil {
		slog.Error("Failed to read settings", errAttr(errSettings))
		return
	}

	go s.process.launchGame(settings)
}

// OnReady is called by the systray package and handles user click events along with
// shutting down the systray subsystem when either the parent context is cancelled or the user
// clicks the quit button.
func (s *appSystray) OnReady(ctx context.Context) func() {
	return func() {
		settings, errSettings := s.settingsMgr.settings(ctx)
		if errSettings != nil {
			slog.Error("Failed to read settings", errAttr(errSettings))
			return
		}

		systray.SetIcon(s.platform.Icon())
		systray.SetTitle("BD")
		systray.SetTooltip(fmt.Sprintf("Bot Detector\n%s", settings.AppURL()))

		go func() {
			openWeb := systray.AddMenuItem("Open BD", "Open BD in your browser")
			openWeb.SetIcon(s.platform.Icon())
			openWeb.Enable()

			launchGame := systray.AddMenuItem("Launch TF2", "Launch Team Fortress 2")
			launchGame.SetIcon(s.platform.Icon())
			launchGame.Enable()

			systray.AddSeparator()

			s.quit = systray.AddMenuItem("Quit", "Quit the application")
			s.quit.Enable()

			for {
				select {
				case <-launchGame.ClickedCh:
					slog.Debug("launchGame clicked")

					go s.onLaunch(ctx)
				case <-openWeb.ClickedCh:
					slog.Debug("openWeb Clicked")
					s.onOpen(ctx)
				case <-s.quit.ClickedCh:
					slog.Debug("User Quit")
					systray.Quit()
					return

				case <-ctx.Done():
					systray.Quit()
					return
				}
			}
		}()
	}
}
