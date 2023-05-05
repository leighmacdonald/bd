package model

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/kirsle/configdir"
	"github.com/leighmacdonald/bd/internal/platform"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io"
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
	Enabled  bool   `yaml:"enabled"`
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	IdFormat string `yaml:"id_format"`
	Deleted  bool   `yaml:"-"`
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

type Settings struct {
	*sync.RWMutex `yaml:"-"`
	// Path to config used when reading Settings
	configPath string `yaml:"-"`
	steamID    string `yaml:"steam_id"`
	// Path to directory with steam.dll (C:\Program Files (x86)\Steam)
	// eg: -> ~/.local/share/Steam/userdata/123456789/config/localconfig.vdf
	steamDir string `yaml:"steam_dir"`
	// Path to tf2 mod (C:\Program Files (x86)\Steam\steamapps\common\Team Fortress 2\tf)
	tf2Dir                 string               `yaml:"tf2_dir"`
	autoLaunchGame         bool                 `yaml:"auto_launch_game_auto"`
	autoCloseOnGameExit    bool                 `yaml:"auto_close_on_game_exit"`
	apiKey                 string               `yaml:"api_key"`
	disconnectedTimeout    string               `yaml:"disconnected_timeout"`
	discordPresenceEnabled bool                 `yaml:"discord_presence_enabled"`
	kickerEnabled          bool                 `yaml:"kicker_enabled"`
	chatWarningsEnabled    bool                 `yaml:"chat_warnings_enabled"`
	partyWarningsEnabled   bool                 `yaml:"party_warnings_enabled"`
	kickTags               []string             `yaml:"kick_tags"`
	voiceBansEnabled       bool                 `yaml:"voice_bans_enabled"`
	debugLogEnabled        bool                 `yaml:"debug_log_enabled"`
	lists                  ListConfigCollection `yaml:"lists"`
	links                  []*LinkConfig        `yaml:"links"`
	rconStatic             bool                 `yaml:"rcon_static"`
	guiEnabled             bool                 `yaml:"gui_enabled"`
	httpEnabled            bool                 `yaml:"http_enabled"`
	httpListenAddr         string               `yaml:"http_listen_addr"`
	rcon                   RCONConfigProvider   `yaml:"-"`
}

func (s *Settings) GetVoiceBansEnabled() bool {
	s.RLock()
	defer s.RUnlock()
	return s.voiceBansEnabled
}

func (s *Settings) SetVoiceBansEnabled(enabled bool) {
	s.Lock()
	defer s.Unlock()
	s.voiceBansEnabled = enabled
}

func (s *Settings) GetDebugLogEnabled() bool {
	s.RLock()
	defer s.RUnlock()
	return s.debugLogEnabled
}

func (s *Settings) SetDebugLogEnabled(enabled bool) {
	s.Lock()
	defer s.Unlock()
	s.debugLogEnabled = enabled
}

func (s *Settings) GetHttpEnabled() bool {
	s.RLock()
	defer s.RUnlock()
	return s.httpEnabled
}

func (s *Settings) SetHttpEnabled(enabled bool) {
	s.Lock()
	defer s.Unlock()
	s.httpEnabled = enabled
}

func (s *Settings) GetGuiEnabled() bool {
	s.RLock()
	defer s.RUnlock()
	return s.guiEnabled
}

func (s *Settings) SetGuiEnabled(enabled bool) {
	s.Lock()
	defer s.Unlock()
	s.guiEnabled = enabled
}

func (s *Settings) GetRcon() RCONConfigProvider {
	s.RLock()
	defer s.RUnlock()
	return s.rcon
}

func (s *Settings) GetRCONStatic() bool {
	s.RLock()
	defer s.RUnlock()
	return s.rconStatic
}

func (s *Settings) GetKickerEnabled() bool {
	s.RLock()
	defer s.RUnlock()
	return s.kickerEnabled
}

func (s *Settings) GetAutoCloseOnGameExit() bool {
	s.RLock()
	defer s.RUnlock()
	return s.autoCloseOnGameExit
}

func (s *Settings) SetSteamID(steamID string) {
	s.Lock()
	defer s.Unlock()
	s.steamID = steamID
}
func (s *Settings) SetAutoCloseOnGameExit(autoClose bool) {
	s.Lock()
	defer s.Unlock()
	s.autoCloseOnGameExit = autoClose
}

func (s *Settings) SetAutoLaunchGame(autoLaunch bool) {
	s.Lock()
	defer s.Unlock()
	s.autoLaunchGame = autoLaunch
}

func (s *Settings) SetRconStatic(static bool) {
	s.Lock()
	defer s.Unlock()
	s.rconStatic = static
}

func (s *Settings) SetChatWarningsEnabled(enabled bool) {
	s.Lock()
	defer s.Unlock()
	s.chatWarningsEnabled = enabled
}

func (s *Settings) SetPartyWarningsEnabled(enabled bool) {
	s.Lock()
	defer s.Unlock()
	s.partyWarningsEnabled = enabled
}

func (s *Settings) SetKickerEnabled(enabled bool) {
	s.Lock()
	defer s.Unlock()
	s.kickerEnabled = enabled
}

func (s *Settings) SetTF2Dir(dir string) {
	s.Lock()
	defer s.Unlock()
	s.tf2Dir = dir
}

func (s *Settings) SetSteamDir(dir string) {
	s.Lock()
	defer s.Unlock()
	s.steamID = dir
}

func (s *Settings) SetKickTags(tags []string) {
	s.Lock()
	defer s.Unlock()
	s.kickTags = tags
}

func (s *Settings) SetAPIKey(key string) {
	s.Lock()
	defer s.Unlock()
	s.apiKey = key
}

func (s *Settings) SetLists(lists ListConfigCollection) {
	s.Lock()
	defer s.Unlock()
	s.lists = lists
}

func (s *Settings) SetLinks(links []*LinkConfig) {
	s.Lock()
	defer s.Unlock()
	s.links = links
}

func (s *Settings) GetAutoLaunchGame() bool {
	s.RLock()
	defer s.RUnlock()
	return s.autoLaunchGame
}

func (s *Settings) SetDiscordPresenceEnabled(enabled bool) {
	s.Lock()
	defer s.Unlock()
	s.discordPresenceEnabled = enabled
}

func (s *Settings) GetDiscordPresenceEnabled() bool {
	s.RLock()
	defer s.RUnlock()
	return s.discordPresenceEnabled
}

func (s *Settings) GetPartyWarningsEnabled() bool {
	s.RLock()
	defer s.RUnlock()
	return s.partyWarningsEnabled
}

func (s *Settings) GetChatWarningsEnabled() bool {
	s.RLock()
	defer s.RUnlock()
	return s.chatWarningsEnabled
}

func (s *Settings) GetAPIKey() string {
	s.RLock()
	defer s.RUnlock()
	return s.apiKey
}
func (s *Settings) GetConfigPath() string {
	s.RLock()
	defer s.RUnlock()
	return s.configPath
}
func (s *Settings) GetTF2Dir() string {
	s.RLock()
	defer s.RUnlock()
	return s.tf2Dir
}

func (s *Settings) GetSteamDir() string {
	s.RLock()
	defer s.RUnlock()
	return s.steamDir
}

func (s *Settings) GetLists() ListConfigCollection {
	s.RLock()
	defer s.RUnlock()
	return s.lists
}

func (s *Settings) GetKickTags() []string {
	s.RLock()
	defer s.RUnlock()
	return s.kickTags
}

func (s *Settings) GetSteamId() steamid.SID64 {
	value, err := steamid.StringToSID64(s.steamID)
	if err != nil {
		return 0
	}
	return value
}

func (s *Settings) AddList(config *ListConfig) error {
	s.Lock()
	defer s.Unlock()
	for _, known := range s.lists {
		if config.ListType == known.ListType &&
			strings.EqualFold(config.URL, known.URL) {
			return errDuplicateList
		}
	}
	s.lists = append(s.lists, config)
	return nil
}

func (s *Settings) GetLinks() LinkConfigCollection {
	s.RLock()
	defer s.RUnlock()
	return s.links
}

func NewSettings() (*Settings, error) {
	settings := Settings{
		RWMutex:                &sync.RWMutex{},
		configPath:             "",
		steamID:                "",
		steamDir:               platform.DefaultSteamRoot,
		tf2Dir:                 platform.DefaultTF2Root,
		autoLaunchGame:         false,
		autoCloseOnGameExit:    false,
		apiKey:                 "",
		disconnectedTimeout:    "60s",
		discordPresenceEnabled: true,
		kickerEnabled:          false,
		chatWarningsEnabled:    false,
		partyWarningsEnabled:   true,
		kickTags:               []string{"cheater", "bot", "trigger_name", "trigger_msg"},
		voiceBansEnabled:       false,
		guiEnabled:             true,
		debugLogEnabled:        false,
		lists: []*ListConfig{
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
		links: []*LinkConfig{
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
		rconStatic:     false,
		httpEnabled:    true,
		httpListenAddr: "localhost:8900",
		rcon:           NewRconConfig(false),
	}
	if !util.Exists(settings.ListRoot()) {
		if err := os.MkdirAll(settings.ListRoot(), 0755); err != nil {
			return nil, errors.Wrap(err, "Failed to initialize Settings directory")
		}
	}
	return &settings, nil
}

func (s *Settings) LogFilePath() string {
	return filepath.Join(configdir.LocalConfig(configRoot), "bd.log")
}

func (s *Settings) ReadDefaultOrCreate() error {
	configPath := configdir.LocalConfig(configRoot)
	if err := configdir.MakePath(configPath); err != nil {
		return err
	}
	errRead := s.ReadFilePath(filepath.Join(configPath, defaultConfigFileName))
	if errRead != nil && errors.Is(errRead, errConfigNotFound) {
		return s.Save()
	}
	s.reload()
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
	if !util.Exists(filePath) {
		// Use defaults
		s.configPath = filePath
		return errConfigNotFound
	}
	settingsFile, errOpen := os.Open(filePath)
	if errOpen != nil {
		return errOpen
	}
	defer util.IgnoreClose(settingsFile)
	if errRead := s.Read(settingsFile); errRead != nil {
		return errRead
	}
	s.configPath = filePath
	return nil
}

func (s *Settings) Read(inputFile io.Reader) error {
	s.Lock()
	if errDecode := yaml.NewDecoder(inputFile).Decode(&s); errDecode != nil {
		s.Unlock()
		return errDecode
	}
	s.Unlock()
	s.reload()
	return nil
}

func (s *Settings) Save() error {
	if errWrite := s.WriteFilePath(s.GetConfigPath()); errWrite != nil {
		return errWrite
	}
	s.reload()
	return nil
}

func (s *Settings) reload() {
	newCfg := NewRconConfig(s.GetRCONStatic())
	s.Lock()
	s.rcon = newCfg
	s.Unlock()
}

func (s *Settings) WriteFilePath(filePath string) error {
	settingsFile, errOpen := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if errOpen != nil {
		return errors.Wrapf(errOpen, "Failed to open Settings file for writing")
	}
	defer util.IgnoreClose(settingsFile)
	return s.Write(settingsFile)
}

func (s *Settings) Write(outputFile io.Writer) error {
	s.RLock()
	defer s.RUnlock()
	return yaml.NewEncoder(outputFile).Encode(s)
}

const (
	rconDefaultListenAddr = "127.0.0.1"
	rconDefaultPort       = 21212
	rconDefaultPassword   = "pazer_sux_lol"
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

func NewRconConfig(static bool) RCONConfigProvider {
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
		password: golib.RandomString(10),
	}
}
