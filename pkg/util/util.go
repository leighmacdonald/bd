package util

import (
	"io"
	"log"
	"os"
)

func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

func LogClose(closer io.Closer) {
	if errClose := closer.Close(); errClose != nil {
		log.Printf("Error trying to close: %v\n", errClose)
	}
}

func IgnoreClose(closer io.Closer) {
	_ = closer.Close()
}
