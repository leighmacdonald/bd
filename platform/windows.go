//go:build windows

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/andygrunwald/vdf"
	"github.com/leighmacdonald/bd/assets"
	"github.com/mitchellh/go-ps"
	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"golang.org/x/sys/windows/registry"
)

type WindowsPlatform struct {
	defaultSteamRoot      string
	defaultTF2Root        string
	binaryName            string
	tf2RootValidationFile string
}

func New() WindowsPlatform {
	var (
		defaultSteamRoot = "C:/Program Files (x86)/Steam"
		defaultTF2Root   = "C:/Program Files (x86)/Steam/steamapps/common/Team Fortress 2/tf"
	)

	foundSteamRoot, errFoundSteamRoot := getSteamRoot()
	if errFoundSteamRoot == nil && Exists(foundSteamRoot) {
		defaultSteamRoot = foundSteamRoot

		tf2Dir, errTf2Dir := getTF2Folder()
		if errTf2Dir == nil {
			defaultTF2Root = tf2Dir
		}
	}

	return WindowsPlatform{
		defaultSteamRoot:      defaultSteamRoot,
		defaultTF2Root:        defaultTF2Root,
		binaryName:            "hl2.exe",
		tf2RootValidationFile: "bin/client.dll",
	}
}

func (l WindowsPlatform) DefaultSteamRoot() string {
	return l.defaultSteamRoot
}

func (l WindowsPlatform) DefaultTF2Root() string {
	return l.defaultTF2Root
}

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
	if !Exists(libPath) {
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

	libs, libsOk := result["libraryfolders"].(map[string]any)
	if !libsOk {
		return "", errors.New("Failed to cast libs")
	}

	for _, library := range libs {
		currentLibraryPath, currentLibraryPathOk := library.(map[string]any)["path"]
		if !currentLibraryPathOk {
			return "", errors.New("Failed to cast currentLibraryPath")
		}

		apps, appsOk := library.(map[string]any)["apps"]
		if !appsOk {
			return "", errors.New("Failed to cast apps")
		}

		sm, smOk := apps.(map[string]any)
		if !smOk {
			return "", errors.New("Failed to cast sm")
		}

		for key := range sm {
			if key == "440" {
				gameLibPath, gameLibPathOk := currentLibraryPath.(string)
				if !gameLibPathOk {
					return "", errors.New("Failed to cast libPath")
				}

				return filepath.Join(gameLibPath, "steamapps", "common", "Team Fortress 2", "tf"), nil
			}
		}
	}

	return "", errors.New("TF2 install path could not be found")
}

func (l WindowsPlatform) LaunchTF2(tf2Dir string, args []string) error {
	var (
		procAttr os.ProcAttr
		hl2Path  = filepath.Join(filepath.Dir(tf2Dir), l.binaryName)
	)

	procAttr.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}

	process, errStart := os.StartProcess(hl2Path, append([]string{hl2Path}, args...), &procAttr)
	if errStart != nil {
		return errors.Wrap(errStart, "failed to launch TF2")
	}

	_, errWait := process.Wait()
	if errWait != nil {
		return errors.Wrap(errWait, "error waiting for game process")
	}

	return nil
}

func (l WindowsPlatform) OpenFolder(dir string) error {
	if errRun := exec.Command("explorer", strings.ReplaceAll(dir, "/", "\\")).Start(); errRun != nil { //nolint:gosec
		return errors.Wrap(errRun, "failed to start process")
	}

	return nil
}

func (l WindowsPlatform) IsGameRunning() (bool, error) {
	processes, errPs := ps.Processes()
	if errPs != nil {
		return false, errors.Wrap(errPs, "Failed to get process list")
	}

	for _, process := range processes {
		if process.Executable() == l.binaryName {
			return true, nil
		}
	}

	return false, nil
}

func (l WindowsPlatform) Icon() []byte {
	return assets.Read(assets.IconWindows)
}

func (l WindowsPlatform) OpenURL(url string) error {
	if errOpen := browser.OpenURL(url); errOpen != nil {
		return errors.Wrap(errOpen, "Failed to open url")
	}

	return nil
}
