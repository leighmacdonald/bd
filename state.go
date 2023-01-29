package main

import (
	"context"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v2/extra"
	"log"
	"time"
)

func updatePlayerState(ctx context.Context, address string, password string, state *model.ServerState) {
	conn, errConn := rcon.Dial(ctx, address, password, time.Second*5)
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
		state.Players[lobbyPlayer.SteamId] = &lobbyPlayer
	}
	state.Server = status.ServerName
	state.CurrentMap = status.Map
	for _, statusPlayer := range status.Players {
		state.Players[statusPlayer.SID].Name = statusPlayer.Name
		state.Players[statusPlayer.SID].ConnectedTime = statusPlayer.ConnectedTime
		state.Players[statusPlayer.SID].UserId = statusPlayer.UserID
	}
}
