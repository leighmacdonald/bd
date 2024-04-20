package platform

import (
	"errors"
	"os"
)

var (
	ErrLaunchBinary        = errors.New("failed to launch binary")
	ErrLaunchWait          = errors.New("process returned error during wait")
	ErrStartProcess        = errors.New("failed to start process")
	ErrReadProcess         = errors.New("failed to read process state")
	ErrOpenURL             = errors.New("failed to open URL")
	ErrInstallPath         = errors.New("failed to locate steam install path")
	ErrRootPath            = errors.New("failed to get steam root")
	ErrSteamLibraryFolders = errors.New("failed to find libraryfolders.vdf")
	ErrVDFOpen             = errors.New("failed to open vdf")
	ErrVDFParse            = errors.New("failed to parse vdf")
	ErrVDFKey              = errors.New("failed to get child key")
	ErrVDFValue            = errors.New("invalid vdf value")
	ErrGameInstallPath     = errors.New("game install path could not be found")
)

// Platform is used to implement operating system specific functionality across linux and windows.
type Platform interface {
	DefaultSteamRoot() string
	DefaultTF2Root() string
	LaunchTF2(steamRoot string, args ...string) error
	OpenFolder(dir string) error
	IsGameRunning() (bool, error)
	Icon() []byte
	OpenURL(url string) error
}

func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}
