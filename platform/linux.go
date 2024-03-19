//go:build !windows

package platform

import (
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"path"

	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/go-ps"
	"github.com/pkg/browser"
)

type LinuxPlatform struct {
	defaultSteamRoot      string
	defaultTF2Root        string
	binaryName            string
	tf2RootValidationFile string
}

func New() LinuxPlatform {
	// We cant really auto-detect this stuff in the same manner as on windows with the registry
	// so linux users may need to configure this manually if .
	knownInstallLocations := []string{
		"~/.local/share/Steam", // Standard location
		"~/.steam/steam/Steam",
	}
	var steamRoot string
	for _, location := range knownInstallLocations {
		expanded, errExpand := homedir.Expand(location)
		if errExpand != nil {
			slog.Warn("Cannot expand home dir", slog.String("error", errExpand.Error()))
			continue
		}

		if Exists(expanded) {
			steamRoot = expanded
			break
		}
	}

	tf2Root := path.Join(steamRoot, "steamapps/common/Team Fortress 2/tf")

	return LinuxPlatform{
		defaultSteamRoot:      steamRoot,
		defaultTF2Root:        tf2Root,
		binaryName:            "hl2_linux",
		tf2RootValidationFile: "bin/client.so",
	}
}

func (l LinuxPlatform) DefaultSteamRoot() string {
	return l.defaultSteamRoot
}

func (l LinuxPlatform) DefaultTF2Root() string {
	return l.defaultTF2Root
}

// LaunchTF2 calls the steam binary directly
// This expects the steam  binary to be in %PATH%
func (l LinuxPlatform) LaunchTF2(_ string, args ...string) error {
	if len(args) != 2 {
		panic("invalid arg count")
	}
	err := exec.Command( //nolint:gosec
		"steam", "-applaunch", "440", "-usercon", "+ip", "0.0.0.0", "-g15",
		"+net_start", "-rpt", "+con_timestamp", "1", "+hostport", args[1], "+rcon_password", args[0],
	).Run()
	if err != nil {
		return errors.Join(err, ErrLaunchBinary)
	}

	return nil
}

func (l LinuxPlatform) OpenFolder(dir string) error {
	if errRun := exec.Command("xdg-open", dir).Start(); errRun != nil {
		return errors.Join(errRun, ErrStartProcess)
	}

	return nil
}

func (l LinuxPlatform) IsGameRunning() (bool, error) {
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

func (l LinuxPlatform) Icon() []byte {
	return readIcon(IconOther)
}

func (l LinuxPlatform) OpenURL(url string) error {
	if errOpen := browser.OpenURL(url); errOpen != nil {
		return errors.Join(errOpen, fmt.Errorf("%w: %s", ErrOpenURL, url))
	}

	return nil
}
