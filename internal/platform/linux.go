//go:build !windows

package platform

import (
	"github.com/mitchellh/go-homedir"
	"log"
	"os/exec"
	"strings"
)

var (
	DefaultSteamRoot      = "~/.local/share/steam/Steam"
	DefaultTF2Root        = "~/.local/share/steam/Steam/steamapps/common/Team Fortress 2/tf"
	TF2RootValidationFile = "bin/client.so"
)

// LaunchTF2 calls the steam binary directly
// On linux args may overflow the allowed length. This will often be 512chars as it's based on the stack size
func LaunchTF2(_ string, args []string) error {
	fa := []string{"-applaunch", "440"}
	fa = append(fa, args...)
	log.Printf("Launching game: steam %s", strings.Join(args, " "))
	cmd := exec.Command("steam", fa...)
	if errLaunch := cmd.Run(); errLaunch != nil {
		return errLaunch
	}
	return nil
}

func OpenFolder(dir string) {
	if errRun := exec.Command("xdg-open", dir).Start(); errRun != nil {
		log.Printf("Failed to start process: %v\n", errRun)
		return
	}
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
