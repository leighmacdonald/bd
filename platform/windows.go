//go:build windows

package platform

import (
	"github.com/andygrunwald/vdf"
	"github.com/leighmacdonald/golib"
	"github.com/pkg/errors"
	"golang.org/x/sys/windows/registry"
	"os"
	"path/filepath"
)

var (
	DefaultSteamRoot        = "C:/Program Files (x86)/Steam"
	DefaultTF2Root          = "C:/Program Files (x86)/Steam/steamapps/common/Team Fortress 2/tf"
	BinaryName              = "hl2.exe"
	SteamRootValidationFile = "Steam.dll"
	TF2RootValidationFile   = "bin/client.dll"
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

func getTF2Folder() (string, error) {
	steamRoot, errRoot := getSteamRoot()
	if errRoot != nil {
		return "", errRoot
	}
	libPath := filepath.Join(steamRoot, "steamapps", "libraryfolders.vdf")
	if !golib.Exists(libPath) {
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
	if !golib.Exists(hl2Path) {
		return "", errors.New("Failed to find hl2.exe")
	}
	return hl2Path, nil
}

func LaunchTF2(tf2Dir string, args []string) error {
	hl2Path := filepath.Join(filepath.Dir(tf2Dir), platform.BinaryName)
	process, errStart := os.StartProcess(hl2Path, append([]string{hl2Path}, args...), &procAttr)
	if errStart != nil {
		log.Printf("Failed to launch TF2: %v\n", errStart)
		return
	}
	state, errWait := process.Wait()
	if errWait != nil {
		log.Printf("Error waiting for game process: %v\n", errWait)
	} else {
		log.Printf("Game exited: %s\n", state.String())
	}
	return nil
}

func init() {
	foundSteamRoot, errFoundSteamRoot := getSteamRoot()
	if errFoundSteamRoot == nil && golib.Exists(foundSteamRoot) {
		DefaultSteamRoot = foundSteamRoot
	}
}
