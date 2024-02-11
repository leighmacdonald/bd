package main

import (
	"crypto/rand"
	"encoding/binary"
	errjoin "errors"
	"fmt"
	"github.com/leighmacdonald/bd/rules"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/kirsle/configdir"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	configRoot            = "bd"
	defaultConfigFileName = "bd.yaml"
)

var (
	errDuplicateList  = errors.New("duplicate list")
	errConfigNotFound = errors.New("config path does not exist")
)

type ListType string

const (
	ListTypeTF2BDPlayerList ListType = "tf2bd_playerlist"
	ListTypeTF2BDRules      ListType = "tf2bd_rules"
)

type ListConfig struct {
	ListType ListType `yaml:"list_type" json:"list_type"`
	Name     string   `yaml:"name" json:"name"`
	Enabled  bool     `yaml:"enabled" json:"enabled"`
	URL      string   `yaml:"url" json:"url"`
}

// SteamIDFormat TODO add to steamid pkg.
type SteamIDFormat string

//goland:noinspection ALL
const (
	Steam64 SteamIDFormat = "steam64"
	Steam3  SteamIDFormat = "steam3"
	Steam32 SteamIDFormat = "steam32"
	Steam   SteamIDFormat = "steam"
)

type LinkConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Name     string `yaml:"name" json:"name"`
	URL      string `yaml:"url" json:"url"`
	IDFormat string `yaml:"id_format" json:"id_format"`
	Deleted  bool   `yaml:"-" json:"deleted"`
}

type LinkConfigCollection []*LinkConfig

func (list LinkConfigCollection) AsAny() []any {
	bl := make([]any, len(list))
	for i, r := range list {
		bl[i] = r
	}

	return bl
}

type ListConfigCollection []*ListConfig

func (list ListConfigCollection) AsAny() []any {
	bl := make([]any, len(list))
	for i, r := range list {
		bl[i] = r
	}

	return bl
}

type RunModes string

const (
	ModeRelease = "release"
	ModeTest    = "test"
	ModeDebug   = "debug"
)

type UserSettings struct {
	// Path to config used when reading UserSettings
	ConfigPath string        `yaml:"-" json:"config_path"`
	SteamID    steamid.SID64 `yaml:"steam_id" json:"steam_id"`
	// Path to directory with steam.dll (C:\Program Files (x86)\Steam)
	// eg: -> ~/.local/share/Steam/userdata/123456789/config/localconfig.vdf
	SteamDir string `yaml:"steam_dir" json:"steam_dir"`
	// Path to tf2 mod (C:\Program Files (x86)\Steam\steamapps\common\Team Fortress 2\tf)
	TF2Dir                  string               `yaml:"tf2_dir" json:"tf2_dir"`
	AutoLaunchGame          bool                 `yaml:"auto_launch_game" json:"auto_launch_game"`
	AutoCloseOnGameExit     bool                 `yaml:"auto_close_on_game_exit" json:"auto_close_on_game_exit"`
	BdAPIEnabled            bool                 `yaml:"bd_api_enabled" json:"bd_api_enabled"`
	BdAPIAddress            string               `yaml:"bd_api_address" json:"bd_api_address"`
	APIKey                  string               `yaml:"api_key" json:"api_key"`
	DisconnectedTimeout     string               `yaml:"disconnected_timeout" json:"disconnected_timeout"`
	DiscordPresenceEnabled  bool                 `yaml:"discord_presence_enabled" json:"discord_presence_enabled"`
	KickerEnabled           bool                 `yaml:"kicker_enabled" json:"kicker_enabled"`
	ChatWarningsEnabled     bool                 `yaml:"chat_warnings_enabled" json:"chat_warnings_enabled"`
	PartyWarningsEnabled    bool                 `yaml:"party_warnings_enabled" json:"party_warnings_enabled"`
	KickTags                []string             `yaml:"kick_tags" json:"kick_tags"`
	VoiceBansEnabled        bool                 `yaml:"voice_bans_enabled" json:"voice_bans_enabled"`
	DebugLogEnabled         bool                 `yaml:"debug_log_enabled" json:"debug_log_enabled"`
	Lists                   ListConfigCollection `yaml:"lists" json:"lists"`
	Links                   []*LinkConfig        `yaml:"links" json:"links"`
	RCONStatic              bool                 `yaml:"rcon_static" json:"rcon_static"`
	HTTPEnabled             bool                 `yaml:"http_enabled" json:"http_enabled"`
	HTTPListenAddr          string               `yaml:"http_listen_addr" json:"http_listen_addr"`
	PlayerExpiredTimeout    int                  `yaml:"player_expired_timeout" json:"player_expired_timeout"`
	PlayerDisconnectTimeout int                  `yaml:"player_disconnect_timeout" json:"player_disconnect_timeout"`
	RunMode                 RunModes             `yaml:"run_mode" json:"run_mode"`
	LogLevel                string               `yaml:"log_level" json:"log_level"`
	Rcon                    RCONConfig           `yaml:"rcon" json:"rcon"`
}

func NewSettings() (UserSettings, error) {
	plat := platform.New()
	newSettings := UserSettings{
		ConfigPath:              ".",
		SteamID:                 "",
		SteamDir:                plat.DefaultSteamRoot(),
		TF2Dir:                  plat.DefaultTF2Root(),
		AutoLaunchGame:          false,
		AutoCloseOnGameExit:     false,
		APIKey:                  "",
		BdAPIEnabled:            true,
		BdAPIAddress:            "",
		DisconnectedTimeout:     "60s",
		DiscordPresenceEnabled:  true,
		KickerEnabled:           false,
		ChatWarningsEnabled:     false,
		PartyWarningsEnabled:    true,
		KickTags:                []string{"cheater", "bot", "trigger_name", "trigger_msg"},
		VoiceBansEnabled:        false,
		DebugLogEnabled:         false,
		RunMode:                 ModeRelease,
		LogLevel:                "info",
		PlayerExpiredTimeout:    6,
		PlayerDisconnectTimeout: 20,
		Lists: []*ListConfig{
			{
				Name:     "Uncletopia",
				ListType: "tf2bd_playerlist",
				Enabled:  false,
				URL:      "https://uncletopia.com/export/bans/tf2bd",
			},
			{
				Name:     "@trusted",
				ListType: "tf2bd_playerlist",
				Enabled:  true,
				URL:      "https://trusted.roto.lol/v1/steamids",
			},
			{
				Name:     "TF2BD Players",
				ListType: "tf2bd_playerlist",
				Enabled:  true,
				URL:      "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/staging/cfg/playerlist.official.json",
			},
			{
				Name:     "TF2BD Rules",
				ListType: "tf2bd_rules",
				Enabled:  true,
				URL:      "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/staging/cfg/rules.official.json",
			},
		},
		Links: []*LinkConfig{
			{
				Enabled:  true,
				Name:     "RGL",
				URL:      "https://rgl.gg/Public/PlayerProfile.aspx?p=%d",
				IDFormat: "steam64",
			},
			{
				Enabled:  true,
				Name:     "Steam",
				URL:      "https://steamcommunity.com/profiles/%d",
				IDFormat: "steam64",
			},
			{
				Enabled:  true,
				Name:     "OzFortress",
				URL:      "https://ozfortress.com/users/steam_id/%d",
				IDFormat: "steam64",
			},
			{
				Enabled:  true,
				Name:     "ESEA",
				URL:      "https://play.esea.net/index.php?s=search&query=%s",
				IDFormat: "steam3",
			},
			{
				Enabled:  true,
				Name:     "UGC",
				URL:      "https://www.ugcleague.com/players_page.cfm?player_id=%d",
				IDFormat: "steam64",
			},
			{
				Enabled:  true,
				Name:     "ETF2L",
				URL:      "https://etf2l.org/search/%d/",
				IDFormat: "steam64",
			},
			{
				Enabled:  true,
				Name:     "trends.tf",
				URL:      "https://trends.tf/player/%d/",
				IDFormat: "steam64",
			},
			{
				Enabled:  true,
				Name:     "demos.tf",
				URL:      "https://demos.tf/profiles/%d",
				IDFormat: "steam64",
			},
			{
				Enabled:  true,
				Name:     "logs.tf",
				URL:      "https://logs.tf/profile/%d",
				IDFormat: "steam64",
			},
		},
		RCONStatic:     false,
		HTTPEnabled:    true,
		HTTPListenAddr: "localhost:8900",
		Rcon:           NewRconConfig(false),
	}

	if !platform.Exists(newSettings.ListRoot()) {
		if err := os.MkdirAll(newSettings.ListRoot(), 0o755); err != nil {
			return newSettings, errors.Wrap(err, "Failed to initialize UserSettings directory")
		}
	}

	return newSettings, nil
}

func (s *UserSettings) LogFilePath() string {
	return filepath.Join(configdir.LocalConfig(configRoot), "bd.log")
}

func (s *UserSettings) AddList(config *ListConfig) error {
	for _, known := range s.Lists {
		if config.ListType == known.ListType &&
			strings.EqualFold(config.URL, known.URL) {
			return errDuplicateList
		}
	}

	s.Lists = append(s.Lists, config)

	return nil
}

func (s *UserSettings) ReadDefaultOrCreate() error {
	configPath := configdir.LocalConfig(configRoot)
	if err := configdir.MakePath(configPath); err != nil {
		return errors.Wrap(err, "Failed to make config dir")
	}

	errRead := s.ReadFilePath(filepath.Join(configPath, defaultConfigFileName))
	if errRead != nil && errors.Is(errRead, errConfigNotFound) {
		return s.Save()
	}

	s.reload()

	return errRead
}

func (s *UserSettings) Validate() error {
	const apiKeyLen = 32

	var err error
	if s.SteamID != "" && !s.SteamID.Valid() {
		err = errjoin.Join(err, steamid.ErrInvalidSID)
	}

	if s.BdAPIEnabled {
		if s.BdAPIAddress == "" {
			err = errjoin.Join(errors.New("BD-API Address cannot be empty"))
		} else {
			_, errParse := url.Parse(s.BdAPIAddress)
			if errParse != nil {
				err = errjoin.Join(errors.New("Invalid address, cannot parse"))
			}
		}
	} else {
		if s.APIKey == "" {
			err = errjoin.Join(errors.New("Must set steam api key when not using bdapi"))
		} else if len(s.APIKey) != apiKeyLen {
			err = errjoin.Join(errors.New("Invalid Steam API Key"))
		}
	}

	return err
}

func (s *UserSettings) ListRoot() string {
	return filepath.Join(s.ConfigRoot(), "lists")
}

func (s *UserSettings) ConfigRoot() string {
	configPath := configdir.LocalConfig(configRoot)
	if err := configdir.MakePath(configPath); err != nil {
		return ""
	}

	return configPath
}

func (s *UserSettings) DBPath() string {
	return filepath.Join(s.ConfigRoot(), "bd.sqlite?cache=shared")
}

func (s *UserSettings) LocalPlayerListPath() string {
	return filepath.Join(s.ListRoot(), fmt.Sprintf("playerlist.%s.json", rules.LocalRuleName))
}

func (s *UserSettings) LocalRulesListPath() string {
	return filepath.Join(s.ListRoot(), fmt.Sprintf("rules.%s.json", rules.LocalRuleName))
}

func (s *UserSettings) ReadFilePath(filePath string) error {
	if !platform.Exists(filePath) {
		// Use defaults
		s.ConfigPath = filePath

		return errConfigNotFound
	}

	settingsFile, errOpen := os.Open(filePath)
	if errOpen != nil {
		return errors.Wrap(errOpen, "Failed to open settings file")
	}

	defer IgnoreClose(settingsFile)

	if errRead := s.Read(settingsFile); errRead != nil {
		return errRead
	}

	s.ConfigPath = filePath

	return nil
}

func (s *UserSettings) Read(inputFile io.Reader) error {
	if errDecode := yaml.NewDecoder(inputFile).Decode(&s); errDecode != nil {
		return errors.Wrap(errDecode, "Failed to decode settings")
	}

	s.reload()

	return nil
}

func (s *UserSettings) Save() error {
	return s.WriteFilePath(s.ConfigPath)
}

func (s *UserSettings) reload() {
	newCfg := NewRconConfig(s.RCONStatic)

	s.Rcon = newCfg
}

func (s *UserSettings) WriteFilePath(filePath string) error {
	settingsFile, errOpen := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o755)
	if errOpen != nil {
		return errors.Wrapf(errOpen, "Failed to open UserSettings file for writing")
	}

	defer IgnoreClose(settingsFile)

	return s.Write(settingsFile)
}

func (s *UserSettings) Write(outputFile io.Writer) error {
	if errEncode := yaml.NewEncoder(outputFile).Encode(s); errEncode != nil {
		return errors.Wrap(errEncode, "Failed to encode settings")
	}

	return nil
}

const (
	rconDefaultListenAddr = "127.0.0.1"
	rconDefaultPort       = 21212
	rconDefaultPassword   = "pazer_sux_lol" //nolint:gosec
)

type RCONConfig struct {
	Address  string `json:"address" yaml:"address"`
	Password string `json:"password" yaml:"password"`
	Port     uint16 `json:"port" yaml:"port"`
}

func (cfg RCONConfig) String() string {
	return fmt.Sprintf("%s:%d", cfg.Address, cfg.Port)
}

func randPort() uint16 {
	const defaultPort = 21212

	var b [8]byte
	if _, errRead := rand.Read(b[:]); errRead != nil {
		return defaultPort
	}

	return uint16(binary.LittleEndian.Uint64(b[:]))
}

func NewRconConfig(static bool) RCONConfig {
	if static {
		return RCONConfig{
			Address:  rconDefaultListenAddr,
			Port:     rconDefaultPort,
			Password: rconDefaultPassword,
		}
	}

	return RCONConfig{
		Address:  rconDefaultListenAddr,
		Port:     randPort(),
		Password: RandomString(10),
	}
}
