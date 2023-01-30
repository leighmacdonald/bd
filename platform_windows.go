//go:build windows

package main

import (
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"golang.org/x/sys/windows/registry"
	"path/filepath"
)

func openSteamRegistry() (registry.Key, error) {
	var access uint32 = registry.QUERY_VALUE
	regKey, errRegKey := registry.OpenKey(registry.CURRENT_USER, "Software\\\\Valve\\\\Steam\\\\ActiveProcess", access)
	if errRegKey != nil {
		return regKey, errors.Wrap(errRegKey, "failed to get steam install path")
	}
	return regKey, nil
}

func getSteamRoot() (string, error) {
	regKey, errRegKey := openSteamRegistry()
	if errRegKey != nil {
		return "", errRegKey
	}
	installPath, _, err := regKey.GetStringValue("SteamClientDll")
	if err != nil {
		return "", errors.Wrap(err, "Failed to read SteamClientDll value")
	}
	return filepath.Dir(installPath), nil
}

func getSteamId() (steamid.SID64, error) {
	regKey, errRegKey := openSteamRegistry()
	if errRegKey != nil {
		return 0, errRegKey
	}
	activeUser, _, err := regKey.GetIntegerValue("ActiveUser")
	if err != nil {
		return 0, errors.Wrap(err, "Failed to read ActiveUser value")
	}
	foundId := steamid.SID32ToSID64(steamid.SID32(activeUser))
	if !foundId.Valid() {
		return 0, errors.New("Invalid registry steam id")
	}
	return foundId, nil
}

func getHL2Path() (string, error) {
	tf2Dir, errTF2Dir := getTF2Folder()
	if errTF2Dir != nil {
		return "", errTF2Dir
	}
	hl2Path := filepath.Join(tf2Dir, "..", "hl2.exe")
	if !exists(hl2Path) {
		return "", errors.New("Failed to find hl2.exe")
	}
	return hl2Path, nil
}
func launchTF2(rconPass string, rconPort uint16) {
	log.Println("Launching tf2...")
	hl2, errHl2 := getHL2Path()
	if errHl2 != nil {
		log.Println(errHl2)
		return
	}
	args, errArgs := getLaunchArgs(rconPass, rconPort)
	if errArgs != nil {
		log.Println(errArgs)
		return
	}
	var procAttr os.ProcAttr
	procAttr.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}
	_, errStart := os.StartProcess(hl2, append([]string{hl2}, args...), &procAttr)
	if errStart != nil {
		log.Printf("Failed to launch TF2: %v", errStart)
	}
}
