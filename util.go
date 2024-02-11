package main

import (
	"crypto/rand"
	"io"
	"log/slog"
	"math/big"
)

func LogClose(closer io.Closer) {
	if errClose := closer.Close(); errClose != nil {
		slog.Error("Error trying to close", errAttr(errClose))
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
