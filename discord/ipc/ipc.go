package ipc

import (
	"bytes"
	"encoding/binary"
	"net"
	"os"

	"errors"
)

var (
	ErrReadIPC      = errors.New("failed to read from discord ipc socket")
	ErrWriteOp      = errors.New("failed to write opcode")
	ErrWritePayload = errors.New("failed to write payload")
	ErrSendBuffer   = errors.New("failed to send payload buffer")
	ErrConnIPC      = errors.New("failed to connect to discord ipc socket/pipe")
)

type DiscordIPC struct {
	socket net.Conn
}

func New() *DiscordIPC {
	return &DiscordIPC{}
}

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

func (ipc *DiscordIPC) Close() error {
	if ipc.socket != nil {
		if errClose := ipc.Close(); errClose != nil {
			return errClose
		}

		ipc.socket = nil
	}

	return nil
}

// Read the socket response.
func (ipc *DiscordIPC) Read() (string, error) {
	buf := make([]byte, 512)
	payloadLen, errRead := ipc.socket.Read(buf)

	if errRead != nil {
		return "", errors.Join(errRead, ErrReadIPC)
	}

	buffer := new(bytes.Buffer)
	for i := 8; i < payloadLen; i++ {
		buffer.WriteByte(buf[i])
	}

	return buffer.String(), nil
}

// Send opcode and payload to the unix socket.
func (ipc *DiscordIPC) Send(opcode int, payload string) (string, error) {
	buf := new(bytes.Buffer)

	if errOpCode := binary.Write(buf, binary.LittleEndian, int32(opcode)); errOpCode != nil {
		return "", errors.Join(errOpCode, ErrWriteOp)
	}

	if errPayload := binary.Write(buf, binary.LittleEndian, int32(len(payload))); errPayload != nil {
		return "", errors.Join(errPayload, ErrWritePayload)
	}

	buf.Write([]byte(payload))

	_, errWrite := ipc.socket.Write(buf.Bytes())
	if errWrite != nil {
		return "", errors.Join(errWrite, ErrSendBuffer)
	}

	return ipc.Read()
}
