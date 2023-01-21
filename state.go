package main

import (
	"context"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"log"
	"time"
)

type serverState struct {
	server     string
	currentMap string
	players    map[steamid.SID64]*player
}

func updatePlayerState(ctx context.Context, state *serverState) {
	conn, errConn := rcon.Dial(ctx, "127.0.0.1:20000", RconPass, time.Second*5)
	if errConn != nil {
		log.Printf("Failed to connect to client: %v\n", errConn)
		return
	}
	defer func() {
		if errClose := conn.Close(); errClose != nil {
			log.Printf("Failed to close rcon connection: %v\n", errClose)
		}
	}()
	statusText, errStatus := conn.Exec("status")
	if errStatus != nil {
		log.Printf("Failed to get status results: %v\n", errStatus)
		return
	}
	lobbyText, errLobby := conn.Exec("tf_lobby_debug")
	if errLobby != nil {
		log.Printf("Failed to get tf_lobby_debug results: %v\n", errLobby)
		return
	}
	lobby := parseLobbyPlayers(lobbyText)
	status, errStatusParse := extra.ParseStatus(statusText, false)
	if errStatusParse != nil {
		log.Printf("Failed to parse status results: %v\n", errStatusParse)
		return
	}
	for _, lobbyPlayer := range lobby {
		state.players[lobbyPlayer.steamId] = &lobbyPlayer
	}
	state.server = status.ServerName
	state.currentMap = status.Map
	for _, statusPlayer := range status.Players {
		state.players[statusPlayer.SID].name = statusPlayer.Name
		state.players[statusPlayer.SID].connectedTime = statusPlayer.ConnectedTime
		state.players[statusPlayer.SID].userId = statusPlayer.UserID
	}
}
