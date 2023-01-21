package main

import (
	"fmt"
	"github.com/andygrunwald/vdf"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"golang.org/x/sys/windows/registry"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func openSteamRegistry() (registry.Key, error) {
	var access uint32 = registry.QUERY_VALUE
	regKey, errRegKey := registry.OpenKey(registry.CURRENT_USER, "Software\\\\Valve\\\\Steam\\\\ActiveProcess", access)
	if errRegKey != nil {
		return regKey, errors.Wrap(errRegKey, "failed to get steam install path")
	}
	return regKey, nil
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
	return steamid.SID32ToSID64(steamid.SID32(activeUser)), nil
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
func getLaunchArgs(rconPass string, rconPort int) ([]string, error) {
	currentArgs, errUserArgs := getUserLaunchArgs()
	if errUserArgs != nil {
		return nil, errors.Wrap(errUserArgs, "Failed to get existing launch options")
	}
	newArgs := []string{
		"dumber",
		"-game", "tf",
		"-steam",
		"-secure",
		"-usercon",
		"+developer", "1", "+alias", "developer",
		"+contimes", "0", "+alias", "contimes",
		"+ip", "0.0.0.0", "+alias", "ip",
		"+sv_rcon_whitelist_address", "127.0.0.1", "+alias", "sv_rcon_whitelist_address",
		"+sv_quota_stringcmdspersecond", "1000000", "+alias", "sv_quota_stringcmdspersecond",
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

func launchTF2() {
	log.Println("Launching tf2...")
	hl2, errHl2 := getHL2Path()
	if errHl2 != nil {
		log.Println(errHl2)
		return
	}
	args, errArgs := getLaunchArgs(RconPass, 20000)
	if errArgs != nil {
		log.Println(errArgs)
		return
	}
	log.Printf("Calling: %s %s\n", hl2, strings.Join(args, " "))

	var procAttr os.ProcAttr
	procAttr.Files = []*os.File{os.Stdin,
		os.Stdout, os.Stderr}
	_, err := os.StartProcess(hl2, append([]string{hl2}, args...), &procAttr)
	if err == nil {
		log.Println(err)
	}
}
