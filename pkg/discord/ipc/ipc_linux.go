//go:build !windows
// +build !windows

package ipc

import (
	"net"
	"time"

	"github.com/pkg/errors"
)

// OpenSocket opens the discord-ipc-0 unix socket.
func (ipc *DiscordIPC) OpenSocket() error {
	sock, errDial := net.DialTimeout("unix", GetIpcPath()+"/discord-ipc-0", time.Second*2)
	if errDial != nil {
		return errors.Wrap(errDial, "Failed to connect to discord socket")
	}

	ipc.socket = sock

	return nil
}
