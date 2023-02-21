package main

import (
	"fmt"
	"github.com/andygrunwald/vdf"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"os"
	"path"
	"strings"
)

func exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

func getLocalConfigPath(steamRoot string, steamID steamid.SID64) (string, error) {
	fp := path.Join(steamRoot, "userdata", fmt.Sprintf("%d", steamid.SID64ToSID32(steamID)), "config", "localconfig.vdf")
	if !exists(fp) {
		return "", errors.New("Path does not exist")
	}
	return fp, nil
}

func getUserLaunchArgs(steamRoot string, steamID steamid.SID64) ([]string, error) {
	localConfigPath, errConfigPath := getLocalConfigPath(steamRoot, steamID)
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

func getLaunchArgs(rconPass string, rconPort uint16, steamRoot string, steamID steamid.SID64) ([]string, error) {
	currentArgs, errUserArgs := getUserLaunchArgs(steamRoot, steamID)
	if errUserArgs != nil {
		return nil, errors.Wrap(errUserArgs, "Failed to get existing launch options")
	}
	newArgs := []string{
		"-game", "tf",
		"-noreactlogin", // needed for vac to load as of late 2022?
		"-steam",
		"-secure",
		"-usercon",
		"+developer", "1", "+alias", "developer",
		"+ip", "0.0.0.0", "+alias", "ip",
		"+sv_rcon_whitelist_address", "127.0.0.1",
		"+sv_quota_stringcmdspersecond", "1000000",
		"+rcon_password", rconPass, "+alias", "rcon_password",
		"+hostport", fmt.Sprintf("%d", rconPort), "+alias", "hostport",
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
