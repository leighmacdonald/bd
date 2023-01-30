package main

import (
	"fmt"
	"github.com/andygrunwald/vdf"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
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

func getLocalConfigPath() (string, error) {
	steamDir, errSteamDir := getSteamRoot()
	if errSteamDir != nil {
		return "", errors.Wrap(errSteamDir, "Could not locate steam root")
	}
	steamId, errSteamId := getSteamId()
	if errSteamId != nil {
		return "", errors.Wrap(errSteamDir, "Could not locate active steam user")
	}
	fp := path.Join(steamDir, "userdata", fmt.Sprintf("%d", steamid.SID64ToSID32(steamId)), "config", "localconfig.vdf")
	if !exists(fp) {
		return "", errors.New("Path does not exist")
	}
	return fp, nil
}

func getUserLaunchArgs() ([]string, error) {
	localConfigPath, errConfigPath := getLocalConfigPath()
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

func getTF2Folder() (string, error) {
	steamRoot, errRoot := getSteamRoot()
	if errRoot != nil {
		return "", errRoot
	}
	libPath := filepath.Join(steamRoot, "steamapps", "libraryfolders.vdf")
	if !exists(libPath) {
		return "", errors.New("Could not find libraryfolders.vdf")
	}
	openVDF, errOpen := os.Open(libPath)
	if errOpen != nil {
		return "", errors.Wrap(errOpen, "failed to open vdf")
	}
	parser := vdf.NewParser(openVDF)
	result, errParse := parser.Parse()
	if errParse != nil {
		return "", errors.Wrap(errOpen, "failed to parse vdf")
	}
	for _, library := range result["libraryfolders"].(map[string]any) {
		currentLibraryPath := library.(map[string]any)["path"]
		apps := library.(map[string]any)["apps"]
		for key := range apps.(map[string]any) {
			if key == "440" {
				return filepath.Join(currentLibraryPath.(string), "steamapps", "common", "Team Fortress 2", "tf"), nil
			}
		}
	}

	return "", errors.New("TF2 install path could not be found")
}

func getLaunchArgs(rconPass string, rconPort uint16) ([]string, error) {
	currentArgs, errUserArgs := getUserLaunchArgs()
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
