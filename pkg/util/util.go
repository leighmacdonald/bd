package util

import (
	"go.uber.org/zap"
	"io"
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
