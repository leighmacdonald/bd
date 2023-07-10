package detector

import (
	"fyne.io/systray"
	"go.uber.org/zap"
)

type Systray struct {
	icon []byte
	log  *zap.Logger
}

func NewSystray(logger *zap.Logger, icon []byte) *Systray {
	tray := &Systray{
		icon: icon,
		log:  logger.Named("systray"),
	}

	return tray
}

func (s *Systray) OnReady() {
	systray.SetIcon(s.icon)
	systray.SetTitle("BD")
	systray.SetTooltip("Bot Detector")

	go func() {
		launch := systray.AddMenuItem("Open BD", "Open BD in your browser")
		launch.SetIcon(s.icon)
		launch.Enable()

		quit := systray.AddMenuItem("Quit", "Quit the application")
		quit.Enable()

		for {
			select {
			case <-launch.ClickedCh:
				s.log.Info("launch Clicked")
			case <-quit.ClickedCh:
				s.log.Debug("User Quit")

				systray.Quit()
			}
		}
	}()
}
