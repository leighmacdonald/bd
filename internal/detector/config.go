package detector

import (
	"fmt"
	"github.com/andygrunwald/vdf"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"os"
	"path"
	"strings"
)

func getLocalConfigPath(steamRoot string, steamID steamid.SID64) (string, error) {
	fp := path.Join(steamRoot, "userdata", fmt.Sprintf("%d", steamid.SID64ToSID32(steamID)), "config", "localconfig.vdf")
	if !util.Exists(fp) {
		return "", errors.New("Path does not exist")
	}
	return fp, nil
}

func getUserLaunchArgs(logger *zap.Logger, steamRoot string, steamID steamid.SID64) ([]string, error) {
	localConfigPath, errConfigPath := getLocalConfigPath(steamRoot, steamID)
	if errConfigPath != nil {
		return nil, errors.Wrap(errConfigPath, "Failed to locate localconfig.vdf")
	}
	logger.Info("Reading userdata", zap.String("path", localConfigPath))
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
		found      bool
		launchOpts []string
		pathKeys   = []string{"UserLocalConfigStore", "Software", "Valve", "sTeam", "apps", "440"}
	)
	for i, key := range pathKeys {
		// Find a matching existing key using case-insensitive match since casing can vary
		csKey := key
		for k := range result {
			if strings.EqualFold(k, key) {
				csKey = k
				break
			}
		}
		result, ok = result[csKey].(map[string]any)
		if !ok {
			return nil, errors.Wrapf(errOpen, "failed to find child key %s", key)
		}

		if i == len(pathKeys)-1 {
			logger.Info("Raw args via userdata", zap.String("args", result["LaunchOptions"].(string)))
			launchOpts = strings.Split(result["LaunchOptions"].(string), " ")
			found = true
		}
	}
	if !found {
		return nil, errors.New("Failed to read LaunchOptions key")
	}
	return launchOpts, nil
}

func getLaunchArgs(logger *zap.Logger, rconPass string, rconPort uint16, steamRoot string, steamID steamid.SID64) ([]string, error) {
	userArgs, errUserArgs := getUserLaunchArgs(logger, steamRoot, steamID)
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
	}

	var full []string
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
