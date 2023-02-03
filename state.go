package main

import (
	"context"
	"github.com/leighmacdonald/rcon/rcon"
	"log"
	"time"
)

func updatePlayerState(ctx context.Context, address string, password string) {
	conn, errConn := rcon.Dial(ctx, address, password, time.Second*5)
	if errConn != nil {
		log.Printf("Failed to connect to client: %v\n", errConn)
		return
	}
	defer func() {
		if errClose := conn.Close(); errClose != nil {
			log.Printf("Failed to Close rcon connection: %v\n", errClose)
		}
	}()
	_, errStatus := conn.Exec("status")
	if errStatus != nil {
		log.Printf("Failed to get status results: %v\n", errStatus)
		return
	}
}
