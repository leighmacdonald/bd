//go:build !windows

package main

import (
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
)

func getSteamId() (steamid.SID64, error) {
	return 0, errors.New("unimplemented")
}

func getSteamRoot() (string, error) {
	return "", errors.New("Not implemented")
}
