package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"time"

	"github.com/kirsle/configdir"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/bd/store"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	errDuplicateList  = errors.New("duplicate list")
	errConfigNotFound = errors.New("config path does not exist")
)

type ListType int

const (
	ListTypeTF2BDPlayerList ListType = 1
	ListTypeTF2BDRules      ListType = 2
)

type ListConfig struct {
	ListType ListType `yaml:"list_type" json:"list_type"`
	Name     string   `yaml:"name" json:"name"`
	Enabled  bool     `yaml:"enabled" json:"enabled"`
	URL      string   `yaml:"url" json:"url"`
}

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

type configManager struct {
	// Path to config used when reading userSettings
	configRoot string
	platform   platform.Platform
	queries    *store.Queries
}

func newSettingsManager(configRoot string, queries *store.Queries, plat platform.Platform) configManager {
	return configManager{platform: plat, configRoot: configRoot, queries: queries}
}

func (sm configManager) DBPath() string {
	return filepath.Join(sm.configRoot, "bd.sqlite?cache=shared")
}

var (
	reDuration         = regexp.MustCompile(`^(\d+)([smhdwMy])$`)
	errInvalidDuration = errors.New("invalid duration")
	errDecodeDuration  = errors.New("invalid duration, cannot decode")
	errQueryConfig     = errors.New("failed to query config")
	errQueryLists      = errors.New("failed to query lists")
)

func (sm configManager) settings(ctx context.Context) (userSettings, error) {
	config, errConfig := sm.queries.Config(ctx)
	if errConfig != nil {
		return userSettings{}, errors.Join(errConfig, errQueryConfig)
	}

	settings := userSettings{
		Config:           config,
		Rcon:             newRconConfig(config.RconStatic),
		SteamID:          steamid.New(config.SteamID),
		configRoot:       sm.configRoot,
		defaultSteamRoot: sm.platform.DefaultSteamRoot(),
	}

	return settings, nil
}

func (sm configManager) lists(ctx context.Context) ([]store.ListsRow, error) {
	lists, err := sm.queries.Lists(ctx)
	if err != nil {
		return nil, errors.Join(err, errQueryLists)
	}

	return lists, nil
}

var errInsertList = errors.New("failed to insert list")

func (sm configManager) AddList(ctx context.Context, list store.List) (store.List, error) {
	now := time.Now()

	newList, errList := sm.queries.ListsInsert(ctx, store.ListsInsertParams{
		ListType:  list.ListType,
		Url:       list.Url,
		Enabled:   list.Enabled,
		UpdatedOn: now,
		CreatedOn: now,
	})
	if errList != nil {
		return store.List{}, errors.Join(errList, errInsertList)
	}

	return newList, nil
}

const apiKeyLen = 32

func validateSettings(settings userSettings) error {
	var err error
	if settings.BdApiEnabled {
		if settings.BdApiAddress == "" {
			err = errors.Join(errSettingsBDAPIAddr)
		} else {
			_, errParse := url.Parse(settings.BdApiAddress)
			if errParse != nil {
				err = errors.Join(errParse, errSettingAddress)
			}
		}
	} else {
		if settings.ApiKey == "" {
			err = errors.Join(errSettingsAPIKeyMissing)
		} else if len(settings.ApiKey) != apiKeyLen {
			err = errors.Join(errSettingAPIKeyInvalid)
		}
	}

	return err
}

var errConfigSave = errors.New("failed to save config")

func (sm configManager) save(ctx context.Context, settings userSettings) error {
	if errValidate := validateSettings(settings); errValidate != nil {
		return errValidate
	}

	if err := sm.queries.ConfigUpdate(ctx, store.ConfigUpdateParams{
		SteamID:                 settings.SteamID.Int64(),
		SteamDir:                settings.SteamDir,
		Tf2Dir:                  settings.Tf2Dir,
		AutoLaunchGame:          settings.AutoLaunchGame,
		AutoCloseOnGameExit:     settings.AutoCloseOnGameExit,
		BdApiEnabled:            settings.BdApiEnabled,
		BdApiAddress:            settings.BdApiAddress,
		ApiKey:                  settings.ApiKey,
		SystrayEnabled:          settings.SystrayEnabled,
		VoiceBansEnabled:        settings.VoiceBansEnabled,
		RconStatic:              settings.RconStatic,
		HttpEnabled:             settings.HttpEnabled,
		HttpListenAddr:          settings.HttpListenAddr,
		PlayerExpiredTimeout:    settings.PlayerExpiredTimeout,
		PlayerDisconnectTimeout: settings.DisconnectedTimeout,
		RunMode:                 settings.RunMode,
		LogLevel:                settings.LogLevel,
		RconAddress:             settings.RconAddress,
		RconPort:                settings.RconPort,
		RconPassword:            settings.RconPassword,
	}); err != nil {
		return errors.Join(err, errConfigSave)
	}

	return nil
}

type userSettings struct {
	store.Config
	configRoot       string
	defaultSteamRoot string
	Rcon             RCONConfig      `mapstructure:"rcon" json:"rcon"`
	SteamID          steamid.SteamID `json:"steam_id"`
	Lists            []store.List    `json:"lists"`
	Links            []store.Link    `json:"links"`
}

func (settings userSettings) LocalPlayerListPath() string {
	return filepath.Join(settings.configRoot, fmt.Sprintf("playerlist.%s.json", rules.LocalRuleName))
}

func (settings userSettings) LocalRulesListPath() string {
	return filepath.Join(settings.configRoot, fmt.Sprintf("rules.%s.json", rules.LocalRuleName))
}

func (settings userSettings) LogFilePath() string {
	return filepath.Join(configdir.LocalConfig(settings.configRoot), "bd.log")
}

func (settings userSettings) AppURL() string {
	return fmt.Sprintf("http://%s/", settings.HttpListenAddr)
}

func (settings userSettings) locateSteamDir() string {
	if settings.SteamDir != "" {
		return settings.SteamDir
	}

	return settings.defaultSteamRoot
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
