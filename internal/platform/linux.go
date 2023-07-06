//go:build !windows

package platform

import (
	"os/exec"

	"github.com/leighmacdonald/bd/internal/asset"
	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/go-ps"
	"github.com/pkg/errors"
)

var (
	DefaultSteamRoot      = "~/.local/share/steam/Steam"                                     //nolint:gochecknoglobals
	DefaultTF2Root        = "~/.local/share/steam/Steam/steamapps/common/Team Fortress 2/tf" //nolint:gochecknoglobals
	BinaryName            = "hl2"                                                            //nolint:gochecknoglobals
	TF2RootValidationFile = "bin/client.so"                                                  //nolint:gochecknoglobals
)

// LaunchTF2 calls the steam binary directly
// On linux args may overflow the allowed length. This will often be 512chars as it's based on the stack size.
func LaunchTF2(_ string, args []string) error {
	fa := []string{"-applaunch", "440"}
	fa = append(fa, args...)
	cmd := exec.Command("steam", fa...)

	if errLaunch := cmd.Run(); errLaunch != nil {
		return errors.Wrap(errLaunch, "Could not launch binary")
	}

	return nil
}

func OpenFolder(dir string) error {
	if errRun := exec.Command("xdg-open", dir).Start(); errRun != nil {
		return errors.Wrap(errRun, "Failed to start process")
	}

	return nil
}

func IsGameRunning() (bool, error) {
	processes, errPs := ps.Processes()
	if errPs != nil {
		return false, errors.Wrap(errPs, "Failed to read processes")
	}

	for _, process := range processes {
		if process.Executable() == BinaryName {
			return true, nil
		}
	}

	return false, nil
}

func Icon() []byte {
	return asset.Read(asset.IconOther)
}

func init() {
	// We cant really auto-detect this stuff in the same manner line on windows with the registry
	// so linux users may need to configure this manually.
	steamRoot, errSR := homedir.Expand(DefaultSteamRoot)
	if errSR == nil {
		DefaultSteamRoot = steamRoot
	}

	tf2Root, errTF2Root := homedir.Expand(DefaultTF2Root)
	if errTF2Root == nil {
		DefaultTF2Root = tf2Root
	}
}
