//go:build !windows

package platform

import (
	"os/exec"

	"github.com/leighmacdonald/bd/internal/asset"
	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/go-ps"
	"github.com/pkg/browser"
	"github.com/pkg/errors"
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
	steamRoot, _ := homedir.Expand("~/.local/share/steam/Steam")
	tf2Root, _ := homedir.Expand("~/.local/share/steam/Steam/steamapps/common/Team Fortress 2/tf")

	return LinuxPlatform{
		defaultSteamRoot:      steamRoot,
		defaultTF2Root:        tf2Root,
		binaryName:            "hl2",
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
func (l LinuxPlatform) LaunchTF2(_ string, args []string) error {
	fa := []string{"-applaunch", "440"}
	fa = append(fa, args...)
	cmd := exec.Command("steam", fa...)

	if errLaunch := cmd.Run(); errLaunch != nil {
		return errors.Wrap(errLaunch, "Could not launch binary")
	}

	return nil
}

func (l LinuxPlatform) OpenFolder(dir string) error {
	if errRun := exec.Command("xdg-open", dir).Start(); errRun != nil {
		return errors.Wrap(errRun, "Failed to start process")
	}

	return nil
}

func (l LinuxPlatform) IsGameRunning() (bool, error) {
	processes, errPs := ps.Processes()
	if errPs != nil {
		return false, errors.Wrap(errPs, "Failed to read processes")
	}

	for _, process := range processes {
		if process.Executable() == l.binaryName {
			return true, nil
		}
	}

	return false, nil
}

func (l LinuxPlatform) Icon() []byte {
	return asset.Read(asset.IconOther)
}

func (l LinuxPlatform) OpenURL(url string) error {
	if errOpen := browser.OpenURL(url); errOpen != nil {
		return errors.Wrap(errOpen, "Failed to open url")
	}

	return nil
}
