//go:build !windows

package platform

import (
	"github.com/leighmacdonald/bd"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

var cachedSteamId steamid.SID64

// getSteamId will scan the user data directory and try to find a directory with a localconfig.vdf
// This has the potential to fail on linux since it will return the first match and not necessarily the user
// actively running steam.
// TODO Find reliable method for linux/(mac?)
func getSteamId() (steamid.SID64, error) {
	if cachedSteamId.Int64() != 0 {
		return cachedSteamId, nil
	}
	sr, errSr := getSteamRoot()
	if errSr != nil {
		return 0, errSr
	}
	dirEntries, errReadDir := os.ReadDir(filepath.Join(sr, "userdata"))
	if errReadDir != nil {
		return 0, errReadDir
	}
	for _, dirPath := range dirEntries {
		if !dirPath.IsDir() {
			continue
		}
		fp := path.Join(sr, "userdata", dirPath.Name(), "config", "localconfig.vdf")
		if !main.exists(fp) {
			continue
		}
		foundId, foundIdParse := strconv.ParseUint(dirPath.Name(), 10, 32)
		if foundIdParse != nil {
			continue
		}
		sid := steamid.SID32ToSID64(steamid.SID32(foundId))
		if !sid.Valid() {
			continue
		}
		cachedSteamId = sid
		return sid, nil
	}

	return 0, errors.New("Could not find eligible steamid")
}

// getSteamRoot on linux currently makes assumptions about the installation path since there is no registry to reference
func getSteamRoot() (string, error) {
	sp, errSp := homedir.Expand("~/.steam/steam")
	if errSp != nil {
		return "", errors.Wrap(errSp, "Failed to get user home steam dir")
	}
	if !main.exists(sp) {
		return "", errors.Errorf("User home steam dir does not exist: %s", sp)
	}
	return sp, nil
}

func getHL2Path() (string, error) {
	tf2Dir, errTF2Dir := main.getTF2Folder()
	if errTF2Dir != nil {
		return "", errTF2Dir
	}
	hl2Path := filepath.Join(tf2Dir, "..", "hl2.sh")
	if !main.exists(hl2Path) {
		return "", errors.New("Failed to find hl2")
	}
	return hl2Path, nil
}

// On linux args may overflow the allowed length. This will often be 512chars as it's based on the stack size
func launchTF2(rconPass string, rconPort uint16) {
	log.Println("Launching tf2...")
	hl2, errHl2 := getHL2Path()
	if errHl2 != nil {
		log.Println(errHl2)
		return
	}
	args, errArgs := main.getLaunchArgs(rconPass, rconPort)
	if errArgs != nil {
		log.Println(errArgs)
		return
	}

	binary := "/usr/bin/bash"
	args = append([]string{hl2}, args...)
	log.Printf("Calling: %s %s\n", binary, strings.Join(args, " "))

	var procAttr os.ProcAttr
	procAttr.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}
	procAttr.Env = "TODO add steam ld paths"
	procAttr.Dir = filepath.Dir(hl2)
	_, errStart := os.StartProcess(binary, append([]string{binary, hl2}, args...), &procAttr)
	if errStart != nil {
		log.Printf("Failed to launch TF2: %v", errStart)
	}
}
