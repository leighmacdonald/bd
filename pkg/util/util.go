package util

import (
	"crypto/rand"
	"io"
	"math/big"
	"os"

	"golang.org/x/exp/slog"
)

func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

func LogClose(logger *slog.Logger, closer io.Closer) {
	if errClose := closer.Close(); errClose != nil {
		logger.Error("Error trying to close", "err", errClose)
	}
}

func IgnoreClose(closer io.Closer) {
	_ = closer.Close()
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[RandInt(len(letters))]
	}
	return string(b)
}

func RandInt(i int) int {
	value, errInt := rand.Int(rand.Reader, big.NewInt(int64(i)))
	if errInt != nil {
		panic(errInt)
	}
	return int(value.Int64())
}
