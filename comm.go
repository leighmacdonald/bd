package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"log"
	"regexp"
	"strings"
)

var (
	rx *regexp.Regexp
)

const (
	rconDefaultHost     = "0.0.0.0"
	rconDefaultPort     = 21212
	rconDefaultPassword = "pazer_sux_lol"
)

type rconConfig struct {
	address  string
	password string
	port     uint16
}

func (cfg rconConfig) String() string {
	return fmt.Sprintf("%s:%d", cfg.address, cfg.port)
}

func (cfg rconConfig) Host() string {
	return cfg.address
}

func (cfg rconConfig) Port() uint16 {
	return cfg.port
}
func (cfg rconConfig) Password() string {
	return cfg.password
}

func randPort() uint16 {
	const defaultPort = 21212
	var b [8]byte
	if _, errRead := rand.Read(b[:]); errRead != nil {
		log.Printf("Failed to generate port number, using default %d: %v\n", defaultPort, errRead)
		return defaultPort
	}
	return uint16(binary.LittleEndian.Uint64(b[:]))
}

type rconConfigProvider interface {
	String() string
	Host() string
	Port() uint16
	Password() string
}

func newRconConfig(static bool) rconConfigProvider {
	if static {
		return rconConfig{
			address:  rconDefaultHost,
			port:     rconDefaultPort,
			password: rconDefaultPassword,
		}
	}
	return rconConfig{
		address:  rconDefaultHost,
		port:     randPort(),
		password: golib.RandomString(10),
	}
}

type rconConnection interface {
	Exec(command string) (string, error)
}

func parseLobbyPlayers(body string) []model.PlayerState {
	var players []model.PlayerState
	for _, line := range strings.Split(body, "\n") {
		match := rx.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		ps := model.PlayerState{
			Name:        "",
			SteamId:     steamid.SID3ToSID64(steamid.SID3(match[3])),
			ConnectedAt: 0,
		}
		if match[4] == "TF_GC_TEAM_INVADERS" {
			ps.Team = model.Blu
		} else {
			ps.Team = model.Red
		}
		players = append(players, ps)
	}
	return players
}

func init() {
	rx = regexp.MustCompile(`^\s+(Pending|Member)\[(\d+)]\s+(\S+)\s+team\s=\s(TF_GC_TEAM_INVADERS|TF_GC_TEAM_DEFENDERS).+?$`)
}
