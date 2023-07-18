package detector

import (
	"context"

	"fyne.io/systray"
	"go.uber.org/zap"
)

type Systray struct {
	icon     []byte
	log      *zap.Logger
	onOpen   func()
	onLaunch func()
	quit     *systray.MenuItem
}

func NewSystray(logger *zap.Logger, icon []byte, onOpen func(), onLaunch func()) *Systray {
	tray := &Systray{
		icon:     icon,
		log:      logger.Named("systray"),
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
					s.log.Debug("launchGame clicked")
					go s.onLaunch()
				case <-openWeb.ClickedCh:
					s.log.Debug("openWeb Clicked")
					s.onOpen()
				case <-s.quit.ClickedCh:
					s.log.Debug("User Quit")
					cancel()
				}
			}
		}()
	}
}
