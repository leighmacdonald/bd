package main

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kirsle/configdir"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"gopkg.in/yaml.v3"
)

const (
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

type settingsManager struct {
	// Path to config used when reading userSettings
	configPath string
	configRoot string
	settingsMu sync.RWMutex
	settings   userSettings
	platform   platform.Platform
}

func newSettingsManager(plat platform.Platform) *settingsManager {
	sm := settingsManager{
		platform:   plat,
		configRoot: "bd",
	}

	return &sm
}

func (sm *settingsManager) setup() error {
	if !platform.Exists(sm.ListRoot()) {
		if err := os.MkdirAll(sm.ListRoot(), 0o755); err != nil {
			return errors.Join(err, errSettingDirectoryCreate)
		}
	}

	return nil
}

func (sm *settingsManager) ConfigRoot() string {
	configPath := configdir.LocalConfig(sm.configRoot)
	if err := configdir.MakePath(configPath); err != nil {
		return ""
	}

	return configPath
}

func (sm *settingsManager) ListRoot() string {
	return filepath.Join(sm.ConfigRoot(), "lists")
}

func (sm *settingsManager) DBPath() string {
	return filepath.Join(sm.ConfigRoot(), "bd.sqlite?cache=shared")
}

func (sm *settingsManager) LocalPlayerListPath() string {
	return filepath.Join(sm.ListRoot(), fmt.Sprintf("playerlist.%s.json", rules.LocalRuleName))
}

func (sm *settingsManager) LocalRulesListPath() string {
	return filepath.Join(sm.ListRoot(), fmt.Sprintf("rules.%s.json", rules.LocalRuleName))
}

func (sm *settingsManager) LogFilePath() string {
	return filepath.Join(configdir.LocalConfig(sm.configRoot), "bd.log")
}

func (sm *settingsManager) readDefaultOrCreate() (userSettings, error) {
	var settings userSettings
	configPath := configdir.LocalConfig(sm.configRoot)
	if err := configdir.MakePath(configPath); err != nil {
		return settings, errors.Join(err, errSettingDirectoryCreate)
	}

	settingsFilePath := filepath.Join(configPath, defaultConfigFileName)

	errRead := sm.readFilePath(settingsFilePath, &settings)
	if errRead != nil {
		if errors.Is(errRead, errConfigNotFound) {
			slog.Info("Creating default config")
			defaultSettings := newSettings(sm.platform)
			if errSave := sm.save(); errSave != nil {
				return settings, errSave
			}

			return defaultSettings, nil
		}

		return userSettings{}, errRead
	}

	// Make sure we have defaults defined if not configured
	if settings.SteamDir == "" {
		settings.SteamDir = sm.locateSteamDir()
	}

	if settings.TF2Dir == "" {
		settings.TF2Dir = sm.locateTF2Dir()
	}

	return settings, nil
}

func (sm *settingsManager) Settings() userSettings {
	sm.settingsMu.RLock()
	defer sm.settingsMu.RUnlock()

	return sm.settings
}

func (sm *settingsManager) validateAndLoad() error {
	settings, errReadSettings := sm.readDefaultOrCreate()
	if errReadSettings != nil {
		return errReadSettings
	}

	if errValidate := settings.Validate(); errValidate != nil {
		return errValidate
	}

	if settings.APIKey != "" {
		if errSetSteamKey := steamweb.SetKey(settings.APIKey); errSetSteamKey != nil {
			slog.Error("Failed to set steam api key", errAttr(errSetSteamKey))
		}
	}

	sm.settingsMu.Lock()
	sm.settings = settings
	sm.settingsMu.Unlock()

	sm.reload()

	return nil
}

func (sm *settingsManager) readFilePath(filePath string, settings *userSettings) error {
	if !platform.Exists(filePath) {
		return errConfigNotFound
	}

	settingsFile, errOpen := os.Open(filePath)
	if errOpen != nil {
		return errors.Join(errOpen, errSettingsOpen)
	}

	defer IgnoreClose(settingsFile)

	if errRead := sm.read(settingsFile, settings); errRead != nil {
		return errRead
	}

	return nil
}

func (sm *settingsManager) read(inputFile io.Reader, settings *userSettings) error {
	if errDecode := yaml.NewDecoder(inputFile).Decode(&settings); errDecode != nil {
		return errors.Join(errDecode, errSettingsDecode)
	}

	settings.Rcon = newRconConfig(settings.RCONStatic)

	return nil
}

func (sm *settingsManager) save() error {
	sm.settingsMu.RLock()

	if errValidate := sm.settings.Validate(); errValidate != nil {
		sm.settingsMu.RUnlock()
		return errValidate
	}

	sm.settingsMu.RUnlock()

	return sm.writeFilePath(sm.configPath)
}

func (sm *settingsManager) reload() {
	sm.settingsMu.Lock()
	defer sm.settingsMu.Unlock()

	sm.settings.Rcon = newRconConfig(sm.settings.RCONStatic)
}

func (sm *settingsManager) writeFilePath(filePath string) error {
	settingsFile, errOpen := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o755)
	if errOpen != nil {
		return errors.Join(errOpen, errSettingsOpenOutput)
	}

	defer IgnoreClose(settingsFile)

	return sm.write(settingsFile)
}

func (sm *settingsManager) write(outputFile io.Writer) error {
	sm.settingsMu.RLock()
	defer sm.settingsMu.RUnlock()

	if errEncode := yaml.NewEncoder(outputFile).Encode(sm.settings); errEncode != nil {
		return errors.Join(errEncode, errSettingsEncode)
	}

	return nil
}

func (sm *settingsManager) replace(newSettings userSettings) error {
	sm.settingsMu.Lock()
	sm.settings = newSettings
	sm.settingsMu.Unlock()

	return sm.writeFilePath(sm.configPath)
}

func (sm *settingsManager) locateSteamDir() string {
	sm.settingsMu.RLock()
	defer sm.settingsMu.RUnlock()

	if sm.settings.SteamDir != "" {
		return sm.settings.SteamDir
	}

	return sm.platform.DefaultSteamRoot()
}

func (sm *settingsManager) locateTF2Dir() string {
	sm.settingsMu.RLock()
	defer sm.settingsMu.RUnlock()

	if sm.settings.TF2Dir != "" {
		return sm.settings.TF2Dir
	}

	return sm.platform.DefaultTF2Root()
}

type userSettings struct {
	SteamID steamid.SID64 `yaml:"steam_id" json:"steam_id"`
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
	SystrayEnabled          bool                 `yaml:"systray_enabled" json:"systray_enabled"`
	Rcon                    RCONConfig           `yaml:"rcon" json:"rcon"`
}

func newSettings(plat platform.Platform) userSettings {
	settings := userSettings{
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
		SystrayEnabled:          true,
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
		Rcon:           newRconConfig(false),
	}

	return settings
}

func (s *userSettings) AppURL() string {
	return fmt.Sprintf("http://%s/", s.HTTPListenAddr)
}

func (s *userSettings) AddList(config *ListConfig) error {
	for _, known := range s.Lists {
		if config.ListType == known.ListType &&
			strings.EqualFold(config.URL, known.URL) {
			return errDuplicateList
		}
	}

	s.Lists = append(s.Lists, config)

	return nil
}

func (s *userSettings) Validate() error {
	const apiKeyLen = 32

	var err error
	if s.SteamID != "" && !s.SteamID.Valid() {
		err = errors.Join(err, steamid.ErrInvalidSID)
	}

	if s.BdAPIEnabled {
		if s.BdAPIAddress == "" {
			err = errors.Join(errSettingsBDAPIAddr)
		} else {
			_, errParse := url.Parse(s.BdAPIAddress)
			if errParse != nil {
				err = errors.Join(errParse, errSettingAddress)
			}
		}
	} else {
		if s.APIKey == "" {
			err = errors.Join(errSettingsAPIKeyMissing)
		} else if len(s.APIKey) != apiKeyLen {
			err = errors.Join(errSettingAPIKeyInvalid)
		}
	}

	return err
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

func newRconConfig(static bool) RCONConfig {
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
