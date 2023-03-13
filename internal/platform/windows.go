//go:build windows

package platform

import (
	"github.com/andygrunwald/vdf"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/mitchellh/go-ps"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sys/windows/registry"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	DefaultSteamRoot      = "C:/Program Files (x86)/Steam"
	DefaultTF2Root        = "C:/Program Files (x86)/Steam/steamapps/common/Team Fortress 2/tf"
	BinaryName            = "hl2.exe"
	TF2RootValidationFile = "bin/client.dll"
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
	if !util.Exists(libPath) {
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

func LaunchTF2(logger *zap.Logger, tf2Dir string, args []string) error {
	hl2Path := filepath.Join(filepath.Dir(tf2Dir), BinaryName)
	var procAttr os.ProcAttr
	procAttr.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}
	logger.Info("Launching game", zap.Strings("args", args), zap.String("binary", hl2Path))
	process, errStart := os.StartProcess(hl2Path, append([]string{hl2Path}, args...), &procAttr)
	if errStart != nil {
		return errors.Wrap(errStart, "Failed to launch TF2\n")
	}
	_, errWait := process.Wait()
	if errWait != nil {
		logger.Error("Error waiting for game process", zap.Error(errWait))
	} else {
		logger.Info("Game exited")
	}
	return nil
}

func OpenFolder(dir string) error {
	if errRun := exec.Command("explorer", strings.ReplaceAll(dir, "/", "\\")).Start(); errRun != nil {
		return errors.Wrap(errRun, "Failed to start process")
	}
	return nil
}

func IsGameRunning() (bool, error) {
	processes, errPs := ps.Processes()
	if errPs != nil {
		return false, errPs
	}
	for _, process := range processes {
		if process.Executable() == BinaryName {
			return true, nil
		}
	}
	return false, nil
}

func init() {
	foundSteamRoot, errFoundSteamRoot := getSteamRoot()
	if errFoundSteamRoot == nil && util.Exists(foundSteamRoot) {
		DefaultSteamRoot = foundSteamRoot
		tf2Dir, errTf2Dir := getTF2Folder()
		if errTf2Dir == nil {
			DefaultTF2Root = tf2Dir
		}
	}
}
