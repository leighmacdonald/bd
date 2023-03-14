package detector

import (
	"context"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/pkg/errors"
)

func updatePlayerState(ctx context.Context, address string, password string) (string, error) {
	localCtx, cancel := context.WithTimeout(ctx, model.DurationRCONRequestTimeout)
	defer cancel()
	conn, errConn := rcon.Dial(localCtx, address, password, model.DurationRCONRequestTimeout)
	if errConn != nil {
		return "", errors.Wrap(errConn, "Failed to connect to client")
	}
	defer util.IgnoreClose(conn)
	// Sent to client, response via log output
	_, errStatus := conn.Exec("status")
	if errStatus != nil {
		return "", errors.Wrap(errStatus, "Failed to get status results")

	}
	// Sent to client, response via direct rcon response
	lobbyStatus, errDebug := conn.Exec("tf_lobby_debug")
	if errDebug != nil {
		return "", errors.Wrap(errDebug, "Failed to get debug results")
	}
	return lobbyStatus, nil
}
