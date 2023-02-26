package main

import (
	"context"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/pkg/errors"
	"log"
	"time"
)

func updatePlayerState(ctx context.Context, address string, password string) (string, error) {
	conn, errConn := rcon.Dial(ctx, address, password, time.Second*5)
	if errConn != nil {
		return "", errors.Wrap(errConn, "Failed to connect to client")
	}
	defer func() {
		if errClose := conn.Close(); errClose != nil {
			log.Printf("Failed to Close rcon connection: %v\n", errClose)
		}
	}()
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
