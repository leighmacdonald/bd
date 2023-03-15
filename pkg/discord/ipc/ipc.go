package ipc

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"net"
	"os"
)

var socket net.Conn

func GetIpcPath() string {
	variableNames := []string{"XDG_RUNTIME_DIR", "TMPDIR", "TMP", "TEMP"}
	for _, name := range variableNames {
		path, exists := os.LookupEnv(name)
		if exists {
			return path
		}
	}

	return "/tmp"
}

func Close() error {
	if socket != nil {
		if errClose := socket.Close(); errClose != nil {
			return errClose
		}
		socket = nil
	}
	return nil
}

// Read the socket response
func Read() (string, error) {
	buf := make([]byte, 512)
	payloadLen, errRead := socket.Read(buf)
	if errRead != nil {
		return "", errRead
	}

	buffer := new(bytes.Buffer)
	for i := 8; i < payloadLen; i++ {
		buffer.WriteByte(buf[i])
	}

	return buffer.String(), nil
}

// Send opcode and payload to the unix socket
func Send(opcode int, payload string) (string, error) {
	buf := new(bytes.Buffer)

	if errOpCode := binary.Write(buf, binary.LittleEndian, int32(opcode)); errOpCode != nil {
		return "", errors.Wrap(errOpCode, "Failed to write opcode")
	}

	if errPayload := binary.Write(buf, binary.LittleEndian, int32(len(payload))); errPayload != nil {
		return "", errors.Wrap(errPayload, "Failed to write payload")
	}

	buf.Write([]byte(payload))
	_, err := socket.Write(buf.Bytes())
	if err != nil {
		return "", errors.Wrap(err, "Failed to send payload buffer")
	}

	return Read()
}
