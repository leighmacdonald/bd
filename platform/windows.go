//go:build windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/andygrunwald/vdf"
	"github.com/mitchellh/go-ps"
	"github.com/pkg/browser"
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
		binaryName:            "tf_win64.exe",
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
		return regKey, errors.Join(errRegKey, ErrInstallPath)
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
		return "", errors.Join(err, ErrRootPath)
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
		return "", ErrSteamLibraryFolders
	}

	openVDF, errOpen := os.Open(libPath)
	if errOpen != nil {
		return "", errors.Join(errOpen, ErrVDFOpen)
	}

	parser := vdf.NewParser(openVDF)

	result, errParse := parser.Parse()
	if errParse != nil {
		return "", errors.Join(errOpen, ErrVDFParse)
	}

	libs, libsOk := result["libraryfolders"].(map[string]any)
	if !libsOk {
		return "", fmt.Errorf("%w: %s", ErrVDFValue, "libraryfolders")
	}

	for _, library := range libs {
		currentLibraryPath, currentLibraryPathOk := library.(map[string]any)["path"]
		if !currentLibraryPathOk {
			return "", fmt.Errorf("%w: %s", ErrVDFValue, "currentLibraryPath")
		}

		apps, appsOk := library.(map[string]any)["apps"]
		if !appsOk {
			return "", fmt.Errorf("%w: %s", ErrVDFValue, "apps")
		}

		sm, smOk := apps.(map[string]any)
		if !smOk {
			return "", fmt.Errorf("%w: %s", ErrVDFValue, "sm")
		}

		for key := range sm {
			if key == "440" {
				gameLibPath, gameLibPathOk := currentLibraryPath.(string)
				if !gameLibPathOk {
					return "", fmt.Errorf("%w: %s", ErrVDFValue, "libPath")
				}

				return filepath.Join(gameLibPath, "steamapps", "common", "Team Fortress 2", "tf"), nil
			}
		}
	}

	return "", ErrGameInstallPath
}

func (l WindowsPlatform) LaunchTF2(tf2Dir string, args ...string) error {
	var (
		procAttr os.ProcAttr
		hl2Path  = filepath.Join(filepath.Dir(tf2Dir), l.binaryName)
	)

	procAttr.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}

	process, errStart := os.StartProcess(hl2Path, append([]string{hl2Path}, args...), &procAttr)
	if errStart != nil {
		return errors.Join(errStart, ErrLaunchBinary)
	}

	_, errWait := process.Wait()
	if errWait != nil {
		return errors.Join(errWait, ErrLaunchWait)
	}

	return nil
}

func (l WindowsPlatform) OpenFolder(dir string) error {
	if errRun := exec.Command("explorer", strings.ReplaceAll(dir, "/", "\\")).Start(); errRun != nil { //nolint:gosec
		return errors.Join(errRun, ErrStartProcess)
	}

	return nil
}

func (l WindowsPlatform) IsGameRunning() (bool, error) {
	processes, errPs := ps.Processes()
	if errPs != nil {
		return false, errors.Join(errPs, ErrReadProcess)
	}

	for _, process := range processes {
		if process.Executable() == l.binaryName {
			return true, nil
		}
	}

	return false, nil
}

func (l WindowsPlatform) Icon() []byte {
	return readIcon(IconWindows)
}

func (l WindowsPlatform) OpenURL(url string) error {
	if errOpen := browser.OpenURL(url); errOpen != nil {
		return errors.Join(errOpen, ErrStartProcess)
	}

	return nil
}
