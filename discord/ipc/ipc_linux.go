//go:build !windows
// +build !windows

package ipc

import (
	"errors"
	"net"
	"time"
)

// OpenSocket opens the discord-ipc-0 unix socket.
func (ipc *DiscordIPC) OpenSocket() error {
	sock, errDial := net.DialTimeout("unix", GetIpcPath()+"/discord-ipc-0", time.Second*2)
	if errDial != nil {
		return errors.Join(errDial, ErrConnIPC)
	}

	ipc.socket = sock

	return nil
}
