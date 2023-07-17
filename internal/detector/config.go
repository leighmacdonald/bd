package detector

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/andygrunwald/vdf"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

func getLocalConfigPath(steamRoot string, steamID steamid.SID64) (string, error) {
	if !steamID.Valid() { //nolint:nestif
		userDataRoot := path.Join(steamRoot, "userdata")
		// Attempt to use the id found in the userdata if only one exists
		entries, err := os.ReadDir(userDataRoot)
		if err != nil {
			return "", errors.Wrap(err, "Failed to read userdata root")
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
					return "", errors.Wrap(err, "Failed to guess userdata root, too many choices")
				}
			}
		}
	}

	configPath := path.Join(steamRoot, "userdata", fmt.Sprintf("%d", steamID.SID32()), "config", "localconfig.vdf")
	if !util.Exists(configPath) {
		return "", errors.New("Path does not exist")
	}

	return configPath, nil
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

	newParser := vdf.NewParser(openVDF)

	result, errParse := newParser.Parse()
	if errParse != nil {
		return nil, errors.Wrap(errOpen, "failed to parse vdf")
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
			return nil, errors.Wrapf(errOpen, "failed to find child key %s", key)
		}

		if index == len(pathKeys)-1 {
			launchStr, launchStrOk := result["LaunchOptions"].(string)
			if !launchStrOk {
				return nil, errors.New("Failed to cast LaunchOptions")
			}

			launchOpts = strings.Split(launchStr, " ")
			found = true
		}
	}

	if !found {
		return nil, errors.New("Failed to read LaunchOptions key")
	}

	return launchOpts, nil
}

func getLaunchArgs(rconPass string, rconPort uint16, steamRoot string, steamID steamid.SID64) ([]string, error) {
	userArgs, errUserArgs := getUserLaunchArgs(steamRoot, steamID)
	if errUserArgs != nil {
		return nil, errors.Wrap(errUserArgs, "Failed to get existing launch options")
	}

	bdArgs := []string{
		"-game", "tf",
		"-noreactlogin", // needed for vac to load as of late 2022?
		"-steam",
		"-secure",
		"-usercon",
		"+ip", "0.0.0.0", "+alias", "ip",
		"+sv_rcon_whitelist_address", "127.0.0.1",
		"+sv_quota_stringcmdspersecond", "1000000",
		"+rcon_password", rconPass, "+alias", "rcon_password",
		"+hostport", fmt.Sprintf("%d", rconPort), "+alias", "hostport",
		"+net_start",
		"+con_timestamp", "1", "+alias", "con_timestamp",
		"-condebug",
		"-conclearlog",
		"-g15",
		"xx",
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
