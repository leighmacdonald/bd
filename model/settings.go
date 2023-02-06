package model

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/kirsle/configdir"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const configRoot = "bd"
const defaultConfigFileName = "bd.yaml"

var (
	errDuplicateList = errors.New("duplicate list")
)

type ListType string

const (
	ListTypeBD              ListType = "bd"
	ListTypeTF2BDPlayerList ListType = "tf2bd_playerlist"
	ListTypeTF2BDRules      ListType = "tf2bd_rules"
	ListTypeUnknown         ListType = "unknown"
)

type ListConfig struct {
	ListType ListType `yaml:"type"`
	Enabled  bool     `yaml:"enabled"`
	URL      string   `yaml:"url"`
}

type Settings struct {
	*sync.RWMutex `yaml:"-"`
	// Path to config used when reading settings
	configPath string `yaml:"-"`
	// Path to directory with steam.dll (C:\Program Files (x86)\Steam)
	SteamRoot string `yaml:"steam_root"`
	// Path to tf2 mod (C:\Program Files (x86)\Steam\steamapps\common\Team Fortress 2\tf)
	TF2Root              string             `yaml:"tf2_root"`
	ApiKey               string             `yaml:"api_key"`
	DisconnectedTimeout  string             `yaml:"disconnected_timeout"`
	KickerEnabled        bool               `yaml:"kicker_enabled"`
	ChatWarningsEnabled  bool               `yaml:"chat_warnings_enabled"`
	PartyWarningsEnabled bool               `yaml:"party_warnings_enabled"`
	Lists                []ListConfig       `yaml:"lists"`
	SteamId              string             `yaml:"steam_id"`
	RconMode             rconMode           `yaml:"rcon_mode"`
	Rcon                 rconConfigProvider `yaml:"-"`
}

func (s *Settings) GetSteamId() steamid.SID64 {
	v, err := steamid.StringToSID64(s.SteamId)
	if err != nil {
		log.Printf("Failed to parse stored steam id: %v\n", err)
		return 0
	}
	return v
}

func (s *Settings) AddList(config ListConfig) error {
	s.Lock()
	defer s.Unlock()
	for _, known := range s.Lists {
		if config.ListType == known.ListType &&
			strings.EqualFold(config.URL, known.URL) {
			return errDuplicateList
		}
	}
	s.Lists = append(s.Lists, config)
	return nil
}

func NewSettings() Settings {
	settings := Settings{
		RWMutex:              &sync.RWMutex{},
		configPath:           "",
		SteamRoot:            platform.DefaultSteamRoot,
		TF2Root:              platform.DefaultTF2Root,
		ApiKey:               "",
		DisconnectedTimeout:  "60s",
		KickerEnabled:        false,
		ChatWarningsEnabled:  false,
		PartyWarningsEnabled: true,
		Lists:                []ListConfig{},
		SteamId:              "",
		RconMode:             rconModeRandom,
		Rcon:                 newRconConfig(true),
	}
	return settings
}

func (s *Settings) ReadDefault() error {
	configPath := configdir.LocalConfig(configRoot)
	if err := configdir.MakePath(configPath); err != nil {
		return err
	}
	return s.ReadFilePath(filepath.Join(configPath, defaultConfigFileName))
}

func (s *Settings) ConfigRoot() string {
	configPath := configdir.LocalConfig(configRoot)
	if err := configdir.MakePath(configPath); err != nil {
		return ""
	}
	return configPath
}

func (s *Settings) DBPath() string {
	return filepath.Join(s.ConfigRoot(), "bd.sqlite")
}

func (s *Settings) ReadFilePath(filePath string) error {
	if !golib.Exists(filePath) {
		// Use defaults
		s.configPath = filePath
		return errors.New("config path does not exist")
	}
	settingsFile, errOpen := os.Open(filePath)
	if errOpen != nil {
		return errOpen
	}
	defer func() {
		if errClose := settingsFile.Close(); errClose != nil {
			log.Printf("Failed to close Settings file: %v\n", errClose)
		}
	}()
	if errRead := s.Read(settingsFile); errRead != nil {
		return errRead
	}
	s.configPath = filePath
	return nil
}

func (s *Settings) Read(inputFile io.Reader) error {
	s.Lock()
	defer s.Unlock()
	return yaml.NewDecoder(inputFile).Decode(&s)
}

func (s *Settings) Save() error {
	return s.WriteFilePath(s.configPath)
}

func (s *Settings) WriteFilePath(filePath string) error {
	settingsFile, errOpen := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0755)
	if errOpen != nil {
		return errors.Wrapf(errOpen, "Failed to open Settings file for writing")
	}
	defer func() {
		if errClose := settingsFile.Close(); errClose != nil {
			log.Printf("Failed to close Settings file: %v\n", errClose)
		}
	}()
	return s.Write(settingsFile)
}

func (s *Settings) Write(outputFile io.Writer) error {
	s.RLock()
	defer s.RUnlock()
	return yaml.NewEncoder(outputFile).Encode(s)
}

type rconMode string

const (
	rconModeStatic rconMode = "static"
	rconModeRandom rconMode = "random"
)

const (
	rconDefaultHost     = "0.0.0.0"
	rconDefaultPort     = 21212
	rconDefaultPassword = "pazer_sux_lol"
)

type rconConfig struct {
	address  string
	password string
	port     uint16
}

func (cfg rconConfig) String() string {
	return fmt.Sprintf("%s:%d", cfg.address, cfg.port)
}

func (cfg rconConfig) Host() string {
	return cfg.address
}

func (cfg rconConfig) Port() uint16 {
	return cfg.port
}
func (cfg rconConfig) Password() string {
	return cfg.password
}

func randPort() uint16 {
	const defaultPort = 21212
	var b [8]byte
	if _, errRead := rand.Read(b[:]); errRead != nil {
		log.Printf("Failed to generate port number, using default %d: %v\n", defaultPort, errRead)
		return defaultPort
	}
	return uint16(binary.LittleEndian.Uint64(b[:]))
}

type rconConfigProvider interface {
	String() string
	Host() string
	Port() uint16
	Password() string
}

func newRconConfig(static bool) rconConfigProvider {
	if static {
		return rconConfig{
			address:  rconDefaultHost,
			port:     rconDefaultPort,
			password: rconDefaultPassword,
		}
	}
	return rconConfig{
		address:  rconDefaultHost,
		port:     randPort(),
		password: golib.RandomString(10),
	}
}
