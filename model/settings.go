package model

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/kirsle/configdir"
	"github.com/leighmacdonald/bd/pkg/rules"
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
	errDuplicateList  = errors.New("duplicate list")
	errConfigNotFound = errors.New("config path does not exist")
)

type ListType string

const (
	//ListTypeBD              ListType = "bd"
	ListTypeTF2BDPlayerList ListType = "tf2bd_playerlist"
	ListTypeTF2BDRules      ListType = "tf2bd_rules"
	//ListTypeUnknown         ListType = "unknown"
)

type ListConfig struct {
	ListType ListType `yaml:"type"`
	Name     string   `yaml:"name"`
	Enabled  bool     `yaml:"enabled"`
	URL      string   `yaml:"url"`
}

// TODO add to steamid pkg
type SteamIdFormat string

const (
	Steam64 SteamIdFormat = "steam64"
	Steam3  SteamIdFormat = "steam3"
	Steam32 SteamIdFormat = "steam32"
	Steam   SteamIdFormat = "steam"
)

type LinkConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Name     string        `yaml:"name"`
	URL      string        `yaml:"url"`
	IdFormat SteamIdFormat `yaml:"id_format"`
}

type Settings struct {
	*sync.RWMutex `yaml:"-"`
	// Path to config used when reading settings
	configPath string `yaml:"-"`
	SteamID    string `yaml:"steam_id"`
	// Path to directory with steam.dll (C:\Program Files (x86)\Steam)
	// eg: -> ~/.local/share/Steam/userdata/123456789/config/localconfig.vdf
	SteamDir string `yaml:"steam_dir"`
	// Path to tf2 mod (C:\Program Files (x86)\Steam\steamapps\common\Team Fortress 2\tf)
	TF2Dir                 string               `yaml:"tf2_dir"`
	ApiKey                 string               `yaml:"api_key"`
	DisconnectedTimeout    string               `yaml:"disconnected_timeout"`
	DiscordPresenceEnabled bool                 `yaml:"discord_presence_enabled"`
	KickerEnabled          bool                 `yaml:"kicker_enabled"`
	ChatWarningsEnabled    bool                 `yaml:"chat_warnings_enabled"`
	PartyWarningsEnabled   bool                 `yaml:"party_warnings_enabled"`
	Lists                  ListConfigCollection `yaml:"lists"`
	Links                  []LinkConfig         `yaml:"links"`
	RconStatic             bool                 `yaml:"rcon_static"`
	Rcon                   RCONConfigProvider   `yaml:"-"`
}

type ListConfigCollection []*ListConfig

func (list ListConfigCollection) AsAny() []any {
	bl := make([]any, len(list))
	for i, r := range list {
		bl[i] = r
	}
	return bl
}

func (s *Settings) GetSteamId() steamid.SID64 {
	value, err := steamid.StringToSID64(s.SteamID)
	if err != nil {
		log.Printf("Failed to parse stored steam id: %v\n", err)
		return 0
	}
	return value
}

func (s *Settings) AddList(config *ListConfig) error {
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

func (s *Settings) GetLinks() []LinkConfig {
	s.RLock()
	defer s.RUnlock()
	return s.Links
}

func NewSettings() (*Settings, error) {
	settings := Settings{
		RWMutex:                &sync.RWMutex{},
		configPath:             "",
		SteamDir:               platform.DefaultSteamRoot,
		TF2Dir:                 platform.DefaultTF2Root,
		ApiKey:                 "",
		DisconnectedTimeout:    "60s",
		DiscordPresenceEnabled: true,
		KickerEnabled:          false,
		ChatWarningsEnabled:    false,
		PartyWarningsEnabled:   true,
		Lists: []*ListConfig{
			{
				ListType: "tf2bd_playerlist",
				Enabled:  false,
				URL:      "https://uncletopia.com/export/bans/tf2bd",
			},
			{
				ListType: "tf2bd_playerlist",
				Enabled:  true,
				URL:      "https://trusted.roto.lol/v1/steamids",
			},
			{
				ListType: "tf2bd_playerlist",
				Enabled:  true,
				URL:      "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/staging/cfg/playerlist.official.json",
			},
			{
				ListType: "tf2bd_rules",
				Enabled:  true,
				URL:      "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/staging/cfg/rules.official.json",
			},
		},
		Links: []LinkConfig{
			{
				Enabled:  true,
				Name:     "RGL",
				URL:      "https://rgl.gg/Public/PlayerProfile.aspx?p=%d",
				IdFormat: "steam64",
			},
			{
				Enabled:  true,
				Name:     "Steam",
				URL:      "https://steamcommunity.com/profiles/%d",
				IdFormat: "steam64",
			},
			{
				Enabled:  true,
				Name:     "OzFortress",
				URL:      "https://ozfortress.com/users/steam_id/%d",
				IdFormat: "steam64",
			},
			{
				Enabled:  true,
				Name:     "ESEA",
				URL:      "https://play.esea.net/index.php?s=search&query=%s",
				IdFormat: "steam3",
			},
			{
				Enabled:  true,
				Name:     "UGC",
				URL:      "https://www.ugcleague.com/players_page.cfm?player_id=%d",
				IdFormat: "steam64",
			},
			{
				Enabled:  true,
				Name:     "ETF2L",
				URL:      "https://etf2l.org/search/%d/",
				IdFormat: "steam64",
			},
			{
				Enabled:  true,
				Name:     "trends.tf",
				URL:      "https://trends.tf/player/%d/",
				IdFormat: "steam64",
			},
			{
				Enabled:  true,
				Name:     "demos.tf",
				URL:      "https://demos.tf/profiles/%d",
				IdFormat: "steam64",
			},
			{
				Enabled:  true,
				Name:     "logs.tf",
				URL:      "https://logs.tf/profile/%d",
				IdFormat: "steam64",
			},
		},
		SteamID:    "",
		RconStatic: false,
		Rcon:       NewRconConfig(false),
	}
	if !golib.Exists(settings.ListRoot()) {
		if err := os.MkdirAll(settings.ListRoot(), 0755); err != nil {
			return nil, errors.Wrap(err, "Failed to initialize settings directory")
		}
	}
	return &settings, nil
}

func (s *Settings) ReadDefaultOrCreate() error {
	configPath := configdir.LocalConfig(configRoot)
	if err := configdir.MakePath(configPath); err != nil {
		return err
	}
	errRead := s.ReadFilePath(filepath.Join(configPath, defaultConfigFileName))
	if errRead != nil && errors.Is(errRead, errConfigNotFound) {
		log.Printf("Creating new config file with defaults")
		return s.Save()
	}
	s.Rcon = NewRconConfig(s.RconStatic)
	return errRead
}

func (s *Settings) ListRoot() string {
	return filepath.Join(s.ConfigRoot(), "lists")
}

func (s *Settings) ConfigRoot() string {
	configPath := configdir.LocalConfig(configRoot)
	if err := configdir.MakePath(configPath); err != nil {
		return ""
	}
	return configPath
}

func (s *Settings) DBPath() string {
	return filepath.Join(s.ConfigRoot(), "bd.sqlite?cache=shared")
}

func (s *Settings) LocalPlayerListPath() string {
	return filepath.Join(s.ListRoot(), fmt.Sprintf("playerlist.%s.json", rules.LocalRuleName))
}

func (s *Settings) LocalRulesListPath() string {
	return filepath.Join(s.ListRoot(), fmt.Sprintf("rules.%s.json", rules.LocalRuleName))
}

func (s *Settings) ReadFilePath(filePath string) error {
	if !golib.Exists(filePath) {
		// Use defaults
		s.configPath = filePath
		return errConfigNotFound
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

const (
	rconDefaultHost     = "0.0.0.0"
	rconDefaultPort     = 21212
	rconDefaultPassword = "pazer_sux_lol"
)

type RCONConfig struct {
	address  string
	password string
	port     uint16
}

func (cfg RCONConfig) String() string {
	return fmt.Sprintf("%s:%d", cfg.address, cfg.port)
}

func (cfg RCONConfig) Host() string {
	return cfg.address
}

func (cfg RCONConfig) Port() uint16 {
	return cfg.port
}

func (cfg RCONConfig) Password() string {
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

type RCONConfigProvider interface {
	String() string
	Host() string
	Port() uint16
	Password() string
}

func NewRconConfig(static bool) RCONConfigProvider {
	if static {
		return RCONConfig{
			address:  rconDefaultHost,
			port:     rconDefaultPort,
			password: rconDefaultPassword,
		}
	}
	return RCONConfig{
		address:  rconDefaultHost,
		port:     randPort(),
		password: golib.RandomString(10),
	}
}
