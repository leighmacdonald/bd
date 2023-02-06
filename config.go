package main

import (
	"fmt"
	"github.com/andygrunwald/vdf"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

func getLocalConfigPath(steamRoot string, steamId steamid.SID64) (string, error) {
	fp := path.Join(steamRoot, "userdata", fmt.Sprintf("%d", steamid.SID64ToSID32(steamId)), "config", "localconfig.vdf")
	if !exists(fp) {
		return "", errors.New("Path does not exist")
	}
	return fp, nil
}

func (bd *BD) launchGameAndWait() {
	log.Println("Launching tf2...")
	hl2Path := filepath.Join(filepath.Dir(bd.settings.TF2Root), platform.BinaryName)
	args, errArgs := getLaunchArgs(
		bd.settings.Rcon.Password(),
		bd.settings.Rcon.Port(),
		bd.settings.SteamRoot,
		bd.settings.GetSteamId())
	if errArgs != nil {
		log.Println(errArgs)
		return
	}
	var procAttr os.ProcAttr
	procAttr.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}
	process, errStart := os.StartProcess(hl2Path, append([]string{hl2Path}, args...), &procAttr)
	if errStart != nil {
		log.Printf("Failed to launch TF2: %v", errStart)
		return
	}
	bd.gameProcess = process
	state, errWait := process.Wait()
	if errWait != nil {
		log.Printf("Error waiting for game process: %v\n", errWait)
	} else {
		log.Printf("Game exited: %s\n", state.String())
	}
	bd.gameProcess = nil
}

func getUserLaunchArgs(steamRoot string, steamId steamid.SID64) ([]string, error) {
	localConfigPath, errConfigPath := getLocalConfigPath(steamRoot, steamId)
	if errConfigPath != nil {
		return nil, errors.Wrap(errConfigPath, "Failed to locate localconfig.vdf")
	}
	openVDF, errOpen := os.Open(localConfigPath)
	if errOpen != nil {
		return nil, errors.Wrap(errOpen, "failed to open vdf")
	}

	parser := vdf.NewParser(openVDF)
	result, errParse := parser.Parse()
	if errParse != nil {
		return nil, errors.Wrap(errOpen, "failed to parse vdf")
	}
	var (
		ok         bool
		launchOpts []string
		pathKeys   = []string{"UserLocalConfigStore", "Software", "Valve", "Steam", "apps", "440"}
	)

	for i, key := range pathKeys {
		result, ok = result[key].(map[string]any)
		if !ok {
			return nil, errors.Wrapf(errOpen, "failed to find child key %s", key)
		}
		if i == len(pathKeys)-1 {
			launchOpts = strings.Split(result["LaunchOptions"].(string), "-")
		}
	}
	var normOpts []string
	for _, opt := range launchOpts {
		if opt == "" {
			continue
		}
		normOpts = append(normOpts, fmt.Sprintf("-%s", opt))
	}
	return normOpts, nil
}

func getLaunchArgs(rconPass string, rconPort uint16, steamRoot string, steamId steamid.SID64) ([]string, error) {
	currentArgs, errUserArgs := getUserLaunchArgs(steamRoot, steamId)
	if errUserArgs != nil {
		return nil, errors.Wrap(errUserArgs, "Failed to get existing launch options")
	}
	newArgs := []string{
		"xx",
		"-game", "tf",
		"-steam",
		"-secure",
		"-usercon",
		"+developer", "1", "+alias", "developer",
		"+contimes", "0", "+alias", "contimes",
		"+ip", "0.0.0.0", "+alias", "ip",
		"+sv_rcon_whitelist_address", "127.0.0.1",
		// "+alias", "sv_rcon_whitelist_address",
		// "+sv_quota_stringcmdspersecond", "1000000", "+alias", "sv_quota_stringcmdspersecond",
		"+rcon_password", rconPass, "+alias", "rcon_password",
		"+hostport", fmt.Sprintf("%d", rconPort), "+alias", "hostport",
		"+alias", "cl_reload_localization_files",
		"+net_start",
		"+con_timestamp", "1", "+alias", "con_timestamp",
		"-condebug",
		"-conclearlog",
	}
	var out []string
	for _, arg := range append(currentArgs, newArgs...) {
		out = append(out, strings.Trim(arg, " "))
	}
	return out, nil
}
