package platform

import "os"

type Platform interface {
	DefaultSteamRoot() string
	DefaultTF2Root() string
	LaunchTF2(_ string, args []string) error
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
