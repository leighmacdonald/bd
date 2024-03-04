package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/andygrunwald/vdf"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

func getLocalConfigPath(steamRoot string, steamID steamid.SID64) (string, error) {
	if !steamID.Valid() { //nolint:nestif
		userDataRoot := path.Join(steamRoot, "userdata")
		// Attempt to use the id found in the userdata if only one exists
		entries, err := os.ReadDir(userDataRoot)
		if err != nil {
			return "", errors.Join(err, errSteamUserData)
		}

		dirCount := 0

		// List the userdata folder to find potential steamid. If there is more than one steamid looking
		// directory, then error out as we don't know which is the correct.
		for _, entry := range entries {
			if entry.IsDir() {
				sidInt, errParse := strconv.ParseInt(entry.Name(), 10, 32)
				if errParse != nil {
					continue
				}

				maybeSteamID := steamid.SID32ToSID64(steamid.SID32(sidInt))
				if maybeSteamID.Valid() {
					steamID = maybeSteamID
				}

				dirCount++
				if dirCount == 2 {
					return "", errors.Join(err, errSteamUserDataGuess)
				}
			}
		}
	}

	configPath := path.Join(steamRoot, "userdata", fmt.Sprintf("%d", steamID.SID32()), "config", "localconfig.vdf")
	if !platform.Exists(configPath) {
		return "", errPathNotExist
	}

	return configPath, nil
}

func getUserLaunchArgs(steamRoot string, steamID steamid.SID64) ([]string, error) {
	localConfigPath, errConfigPath := getLocalConfigPath(steamRoot, steamID)
	if errConfigPath != nil {
		return nil, errors.Join(errConfigPath, errSteamLocalConfig)
	}

	openVDF, errOpen := os.Open(localConfigPath)
	if errOpen != nil {
		return nil, errors.Join(errOpen, platform.ErrVDFOpen)
	}

	newParser := vdf.NewParser(openVDF)

	result, errParse := newParser.Parse()
	if errParse != nil {
		return nil, errors.Join(errOpen, platform.ErrVDFParse)
	}

	var (
		castOk     bool
		found      bool
		launchOpts []string
		pathKeys   = []string{"UserLocalConfigStore", "Software", "Valve", "sTeam", "apps", "440"}
	)

	for index, key := range pathKeys {
		// Find a matching existing key using case-insensitive match since casing can vary
		csKey := key

		for k := range result {
			if strings.EqualFold(k, key) {
				csKey = k

				break
			}
		}

		result, castOk = result[csKey].(map[string]any)
		if !castOk {
			return nil, errors.Join(errOpen, fmt.Errorf("%w: %s", platform.ErrVDFKey, key))
		}

		if index == len(pathKeys)-1 {
			launchStr, launchStrOk := result["LaunchOptions"].(string)
			if !launchStrOk {
				return nil, fmt.Errorf("%w: %s", platform.ErrVDFValue, "LaunchOptions")
			}

			launchOpts = strings.Split(launchStr, " ")
			found = true
		}
	}

	if !found {
		return nil, errGetLaunchOptions
	}

	return launchOpts, nil
}

func getLaunchArgs(rconPass string, rconPort uint16, steamRoot string, steamID steamid.SID64, udpEnabled bool, udpAddr string) ([]string, error) {
	userArgs, errUserArgs := getUserLaunchArgs(steamRoot, steamID)
	if errUserArgs != nil {
		return nil, errors.Join(errUserArgs, errSteamLaunchArgs)
	}

	bdArgs := []string{
		"-game", "tf",
		// "-noreactlogin", // needed for vac to load as of late 2022?
		"-steam",
		"-secure",
		"-usercon",
		"+ip", "0.0.0.0",
		"+sv_rcon_whitelist_address", "127.0.0.1",
		"+sv_quota_stringcmdspersecond", "1000000",
		"+rcon_password", rconPass,
		"+hostport", fmt.Sprintf("%d", rconPort),
		"+net_start",
		"+con_timestamp", "1",
		"-rpt", // Same as having -condebug, -conclearlog, and -console enabled
		"-g15",
	}

	if udpEnabled {
		bdArgs = append(bdArgs, "+logaddress_add", udpAddr)
	}

	var full []string //nolint:prealloc

	for _, arg := range append(bdArgs, userArgs...) {
		arg = strings.Trim(arg, " ")
		if !strings.HasSuffix(arg, "-") || strings.HasPrefix(arg, "+") {
			full = append(full, arg)

			continue
		}

		alreadyKnown := false

		for _, known := range full {
			if known == arg {
				// duplicate arg
				alreadyKnown = true

				break
			}
		}

		if alreadyKnown {
			continue
		}

		full = append(full, arg)
	}

	return full, nil
}
