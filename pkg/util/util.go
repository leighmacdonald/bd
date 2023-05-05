package util

import (
	"go.uber.org/zap"
	"io"
	"math/rand"
	"os"
)

func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

func LogClose(logger *zap.Logger, closer io.Closer) {
	if errClose := closer.Close(); errClose != nil {
		logger.Error("Error trying to close", zap.Error(errClose))
	}
}

func IgnoreClose(closer io.Closer) {
	_ = closer.Close()
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
