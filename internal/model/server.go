package model

import (
	"net"
	"time"
)

type Server struct {
	ServerName string
	Addr       net.IP
	Port       uint16
	CurrentMap string
	Tags       []string
	LastUpdate time.Time
}
