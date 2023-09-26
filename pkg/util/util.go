package util

import (
	"crypto/rand"
	"io"
	"math/big"
	"os"

	"go.uber.org/zap"
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

func RandomString(n int) string {
	var (
		letters  = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
		outBytes = make([]rune, n)
	)

	for i := range outBytes {
		outBytes[i] = letters[RandInt(len(letters))]
	}

	return string(outBytes)
}

func RandInt(i int) int {
	value, errInt := rand.Int(rand.Reader, big.NewInt(int64(i)))
	if errInt != nil {
		panic(errInt)
	}

	return int(value.Int64())
}
