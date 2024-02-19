package main

import (
	"context"
	"github.com/leighmacdonald/bd/addons"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/bd/rules"
	"log/slog"
	"sync/atomic"
)

type processState struct {
	gameProcessActive  atomic.Bool
	gameHasStartedOnce atomic.Bool
	rcon               rconConnection
	platform           platform.Platform
}

func newProcessState(platform platform.Platform, rcon rconConnection) *processState {
	isRunning, _ := platform.IsGameRunning()

	ps := &processState{
		gameProcessActive:  atomic.Bool{},
		gameHasStartedOnce: atomic.Bool{},
		platform:           platform,
		rcon:               rcon,
	}

	ps.gameProcessActive.Store(isRunning)
	ps.gameHasStartedOnce.Store(isRunning)

	return ps
}

// LaunchGameAndWait is the main entry point to launching the game. It will install the included addon, write the
// voice bans out if enabled and execute the platform specific launcher command, blocking until exit.
func (p *processState) LaunchGameAndWait(rules *rules.Engine, settings UserSettings) {
	defer func() {
		p.gameProcessActive.Store(false)
	}()

	if errInstall := addons.Install(settings.TF2Dir); errInstall != nil {
		slog.Error("Error trying to install addon", errAttr(errInstall))
	}

	// TODO Move outside of here
	// if settings.VoiceBansEnabled {
	//	if errVB := rules.ExportVoiceBans(settings.TF2Dir, settings.KickTags); errVB != nil {
	//		slog.Error("Failed to export voiceban list", errAttr(errVB))
	//	}
	// }

	args, errArgs := getLaunchArgs(
		settings.Rcon.Password,
		settings.Rcon.Port,
		settings.SteamDir,
		settings.SteamID)

	if errArgs != nil {
		slog.Error("Failed to get TF2 launch args", errAttr(errArgs))

		return
	}

	p.gameHasStartedOnce.Store(true)

	if errLaunch := p.platform.LaunchTF2(settings.TF2Dir, args); errLaunch != nil {
		slog.Error("Failed to launch game", errAttr(errLaunch))
	}
}

func (p *processState) Quit(ctx context.Context) error {
	if !p.gameProcessActive.Load() {
		return errNotMarked
	}

	_, err := p.rcon.exec(ctx, "quit", false)
	if err != nil {
		return err
	}

	return nil
}
