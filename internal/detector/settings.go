package detector

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kirsle/configdir"
	"github.com/leighmacdonald/bd/internal/platform"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/bd/pkg/util"
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
	ModeProd  = "prod"
	ModeTest  = "test"
	ModeDebug = "debug"
)

type UserSettings struct {
	*sync.RWMutex `yaml:"-"`
	// Path to config used when reading UserSettings
	configPath string `yaml:"-"`
	SteamID    string `yaml:"steam_id" json:"steam_id"`
	// Path to directory with steam.dll (C:\Program Files (x86)\Steam)
	// eg: -> ~/.local/share/Steam/userdata/123456789/config/localconfig.vdf
	SteamDir string `yaml:"steam_dir" json:"steam_dir"`
	// Path to tf2 mod (C:\Program Files (x86)\Steam\steamapps\common\Team Fortress 2\tf)
	TF2Dir                  string               `yaml:"tf2_dir" json:"tf2_dir"`
	AutoLaunchGame          bool                 `yaml:"auto_launch_game" json:"auto_launch_game"`
	AutoCloseOnGameExit     bool                 `yaml:"auto_close_on_game_exit" json:"auto_close_on_game_exit"`
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
	GUIEnabled              bool                 `yaml:"gui_enabled" json:"gui_enabled"`
	HTTPEnabled             bool                 `yaml:"http_enabled" json:"http_enabled"`
	HTTPListenAddr          string               `yaml:"http_listen_addr" json:"http_listen_addr"`
	PlayerExpiredTimeout    int                  `yaml:"player_expired_timeout" json:"player_expired_timeout"`
	PlayerDisconnectTimeout int                  `yaml:"player_disconnect_timeout" json:"player_disconnect_timeout"`
	RunMode                 RunModes             `yaml:"run_mode" json:"run_mode"`
	rcon                    RCONConfigProvider   `yaml:"-" `
}

func (s *UserSettings) GetVoiceBansEnabled() bool {
	s.RLock()
	defer s.RUnlock()

	return s.VoiceBansEnabled
}

func (s *UserSettings) SetVoiceBansEnabled(enabled bool) {
	s.Lock()
	defer s.Unlock()
	s.VoiceBansEnabled = enabled
}

func (s *UserSettings) GetDebugLogEnabled() bool {
	s.RLock()
	defer s.RUnlock()

	return s.DebugLogEnabled
}

func (s *UserSettings) SetDebugLogEnabled(enabled bool) {
	s.Lock()
	defer s.Unlock()
	s.DebugLogEnabled = enabled
}

func (s *UserSettings) GetHTTPEnabled() bool {
	s.RLock()
	defer s.RUnlock()

	return s.HTTPEnabled
}

func (s *UserSettings) SetHTTPEnabled(enabled bool) {
	s.Lock()
	defer s.Unlock()
	s.HTTPEnabled = enabled
}

func (s *UserSettings) GetGuiEnabled() bool {
	s.RLock()
	defer s.RUnlock()

	return s.GUIEnabled
}

func (s *UserSettings) SetGuiEnabled(enabled bool) {
	s.Lock()
	defer s.Unlock()

	s.GUIEnabled = enabled
}

func (s *UserSettings) GetRcon() RCONConfigProvider { // nolint:ireturn
	s.RLock()
	defer s.RUnlock()

	return s.rcon
}

func (s *UserSettings) GetRCONStatic() bool {
	s.RLock()
	defer s.RUnlock()

	return s.RCONStatic
}

func (s *UserSettings) GetKickerEnabled() bool {
	s.RLock()
	defer s.RUnlock()

	return s.KickerEnabled
}

func (s *UserSettings) GetAutoCloseOnGameExit() bool {
	s.RLock()
	defer s.RUnlock()

	return s.AutoCloseOnGameExit
}

func (s *UserSettings) SetSteamID(steamID string) {
	s.Lock()
	defer s.Unlock()

	s.SteamID = steamID
}

func (s *UserSettings) SetAutoCloseOnGameExit(autoClose bool) {
	s.Lock()
	defer s.Unlock()

	s.AutoCloseOnGameExit = autoClose
}

func (s *UserSettings) SetAutoLaunchGame(autoLaunch bool) {
	s.Lock()
	defer s.Unlock()

	s.AutoLaunchGame = autoLaunch
}

func (s *UserSettings) SetRconStatic(static bool) {
	s.Lock()
	defer s.Unlock()

	s.RCONStatic = static
}

func (s *UserSettings) SetChatWarningsEnabled(enabled bool) {
	s.Lock()
	defer s.Unlock()

	s.ChatWarningsEnabled = enabled
}

func (s *UserSettings) SetPartyWarningsEnabled(enabled bool) {
	s.Lock()
	defer s.Unlock()

	s.PartyWarningsEnabled = enabled
}

func (s *UserSettings) SetKickerEnabled(enabled bool) {
	s.Lock()
	defer s.Unlock()

	s.KickerEnabled = enabled
}

func (s *UserSettings) SetTF2Dir(dir string) {
	s.Lock()
	defer s.Unlock()

	s.TF2Dir = dir
}

func (s *UserSettings) SetSteamDir(dir string) {
	s.Lock()
	defer s.Unlock()

	s.SteamID = dir
}

func (s *UserSettings) SetKickTags(tags []string) {
	s.Lock()
	defer s.Unlock()

	s.KickTags = tags
}

func (s *UserSettings) SetAPIKey(key string) {
	s.Lock()
	defer s.Unlock()

	s.APIKey = key
}

func (s *UserSettings) SetLists(lists ListConfigCollection) {
	s.Lock()
	defer s.Unlock()

	s.Lists = lists
}

func (s *UserSettings) SetLinks(links []*LinkConfig) {
	s.Lock()
	defer s.Unlock()

	s.Links = links
}

func (s *UserSettings) GetAutoLaunchGame() bool {
	s.RLock()
	defer s.RUnlock()

	return s.AutoLaunchGame
}

func (s *UserSettings) SetDiscordPresenceEnabled(enabled bool) {
	s.Lock()
	defer s.Unlock()

	s.DiscordPresenceEnabled = enabled
}

func (s *UserSettings) GetDiscordPresenceEnabled() bool {
	s.RLock()
	defer s.RUnlock()

	return s.DiscordPresenceEnabled
}

func (s *UserSettings) GetPartyWarningsEnabled() bool {
	s.RLock()
	defer s.RUnlock()

	return s.PartyWarningsEnabled
}

func (s *UserSettings) GetChatWarningsEnabled() bool {
	s.RLock()
	defer s.RUnlock()

	return s.ChatWarningsEnabled
}

func (s *UserSettings) GetAPIKey() string {
	s.RLock()
	defer s.RUnlock()

	return s.APIKey
}

func (s *UserSettings) GetConfigPath() string {
	s.RLock()
	defer s.RUnlock()

	return s.configPath
}

func (s *UserSettings) GetTF2Dir() string {
	s.RLock()
	defer s.RUnlock()

	return s.TF2Dir
}

func (s *UserSettings) GetSteamDir() string {
	s.RLock()
	defer s.RUnlock()

	return s.SteamDir
}

func (s *UserSettings) GetLists() ListConfigCollection {
	s.RLock()
	defer s.RUnlock()

	return s.Lists
}

func (s *UserSettings) GetKickTags() []string {
	s.RLock()
	defer s.RUnlock()

	return s.KickTags
}

func (s *UserSettings) GetSteamID() steamid.SID64 {
	value, err := steamid.StringToSID64(s.SteamID)
	if err != nil {
		return ""
	}

	return value
}

func (s *UserSettings) AddList(config *ListConfig) error {
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

func (s *UserSettings) GetLinks() LinkConfigCollection {
	s.RLock()
	defer s.RUnlock()

	return s.Links
}

func NewSettings() (*UserSettings, error) {
	newSettings := UserSettings{
		RWMutex:                 &sync.RWMutex{},
		configPath:              ".",
		SteamID:                 "",
		SteamDir:                platform.DefaultSteamRoot,
		TF2Dir:                  platform.DefaultTF2Root,
		AutoLaunchGame:          false,
		AutoCloseOnGameExit:     false,
		APIKey:                  "",
		DisconnectedTimeout:     "60s",
		DiscordPresenceEnabled:  true,
		KickerEnabled:           false,
		ChatWarningsEnabled:     false,
		PartyWarningsEnabled:    true,
		KickTags:                []string{"cheater", "bot", "trigger_name", "trigger_msg"},
		VoiceBansEnabled:        false,
		GUIEnabled:              true,
		DebugLogEnabled:         false,
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
		rcon:           NewRconConfig(false),
	}

	if !util.Exists(newSettings.ListRoot()) {
		if err := os.MkdirAll(newSettings.ListRoot(), 0o755); err != nil {
			return nil, errors.Wrap(err, "Failed to initialize UserSettings directory")
		}
	}

	return &newSettings, nil
}

func (s *UserSettings) LogFilePath() string {
	return filepath.Join(configdir.LocalConfig(configRoot), "bd.log")
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

func (s *UserSettings) MustValidate() {
	if !(s.GetHTTPEnabled() || s.GetGuiEnabled()) {
		panic("Must enable at least one of the gui or http packages")
	}
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
	if !util.Exists(filePath) {
		// Use defaults
		s.configPath = filePath

		return errConfigNotFound
	}

	settingsFile, errOpen := os.Open(filePath)
	if errOpen != nil {
		return errors.Wrap(errOpen, "Failed to open settings file")
	}

	defer util.IgnoreClose(settingsFile)

	if errRead := s.Read(settingsFile); errRead != nil {
		return errRead
	}

	s.configPath = filePath

	return nil
}

func (s *UserSettings) Read(inputFile io.Reader) error {
	s.Lock()
	if errDecode := yaml.NewDecoder(inputFile).Decode(&s); errDecode != nil {
		s.Unlock()

		return errors.Wrap(errDecode, "Failed to decode settings")
	}

	s.Unlock()
	s.reload()

	return nil
}

func (s *UserSettings) Save() error {
	if errWrite := s.WriteFilePath(s.GetConfigPath()); errWrite != nil {
		return errWrite
	}

	s.reload()

	return nil
}

func (s *UserSettings) reload() {
	newCfg := NewRconConfig(s.GetRCONStatic())
	s.Lock()
	s.rcon = newCfg
	s.Unlock()
}

func (s *UserSettings) WriteFilePath(filePath string) error {
	settingsFile, errOpen := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o755)
	if errOpen != nil {
		return errors.Wrapf(errOpen, "Failed to open UserSettings file for writing")
	}

	defer util.IgnoreClose(settingsFile)

	return s.Write(settingsFile)
}

func (s *UserSettings) Write(outputFile io.Writer) error {
	s.RLock()
	defer s.RUnlock()

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

func NewRconConfig(static bool) RCONConfig {
	if static {
		return RCONConfig{
			address:  rconDefaultListenAddr,
			port:     rconDefaultPort,
			password: rconDefaultPassword,
		}
	}

	return RCONConfig{
		address:  rconDefaultListenAddr,
		port:     randPort(),
		password: util.RandomString(10),
	}
}
