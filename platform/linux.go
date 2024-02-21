//go:build !windows

package platform

import (
	"errors"
	"fmt"
	"github.com/leighmacdonald/bd/frontend"
	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/go-ps"
	"github.com/pkg/browser"
	"log/slog"
	"os/exec"
	"path"
	"strings"
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
	var knownInstallLocations = []string{
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
// On linux args may overflow the allowed length. This will often be 512chars as it's based on the stack size.
func (l LinuxPlatform) LaunchTF2(steamRoot string, args []string) error {
	cmdArgs := fmt.Sprintf("steam://rungameid/440/%s/", strings.Join(args, "/"))
	slog.Debug("launching game", slog.String("args", cmdArgs))
	cmd := exec.Command("xdg-open", cmdArgs)
	//steamBin, errBin := exec.LookPath("steam")
	//if errBin != nil {
	//	return errBin
	//}

	//steamBin := path.Join(steamRoot, "steamapps/common/Team Fortress 2/hl2.sh")

	//fa := []string{steamBin, "-applaunch", "440"}
	//fa := []string{steamBin}
	//fa = append(fa, args...)
	//slog.Debug(fmt.Sprintf("calling %s %s", steamBin, strings.Join(fa, " ")))
	//cmd := exec.Command("/bin/bash", fa...)

	if errLaunch := cmd.Run(); errLaunch != nil {
		return errors.Join(errLaunch, ErrLaunchBinary)
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
	return frontend.Read(frontend.IconOther)
}

func (l LinuxPlatform) OpenURL(url string) error {
	if errOpen := browser.OpenURL(url); errOpen != nil {
		return errors.Join(errOpen, fmt.Errorf("%w: %s", ErrOpenURL, url))
	}

	return nil
}
