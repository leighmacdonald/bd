//go:build !windows

package main

import (
	"github.com/pkg/errors"
)

func getSteamRoot() (string, error) {
	return "", errors.New("Not implemented")
}
