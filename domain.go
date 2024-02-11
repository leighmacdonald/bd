package main

import (
	"errors"
	"time"
)

var (
	errFetchSummaries         = errors.New("failed to fetch summaries")
	errFetchBans              = errors.New("failed to fetch bans")
	errAPIKey                 = errors.New("failed to set steam api key")
	errCreateRequest          = errors.New("failed to create request")
	errPerformRequest         = errors.New("failed to perform request")
	errDecodeResponse         = errors.New("failed to decode response")
	errReadResponse           = errors.New("failed to read response")
	errCacheSetup             = errors.New("failed to setup cache dirs")
	errCreateCacheDir         = errors.New("failed to make output cache dir")
	errOpenCacheFile          = errors.New("failed to open output cache file")
	errWriteCacheFile         = errors.New("failed to write output file")
	errReadCacheFile          = errors.New("failed to read cache file content")
	errSteamUserData          = errors.New("failed to read steam userdata root folder")
	errSteamUserDataGuess     = errors.New("could not determine userdata folder")
	errSteamLocalConfig       = errors.New("failed to locate localconfig.vdf")
	errSteamLaunchArgs        = errors.New("failed to get existing launch options")
	errLogTailCreate          = errors.New("could not create tail reader")
	errDuration               = errors.New("failed to parse connected duration")
	errDataSourceAPI          = errors.New("failed to load api data source")
	errDataSourceLocal        = errors.New("failed to load local data source")
	errVoiceBanOpen           = errors.New("failed to open voice_ban.dt")
	errVoiceBanWrite          = errors.New("failed to write voice_ban.dt")
	errMark                   = errors.New("failed to mark player")
	errPlayerListOpen         = errors.New("failed to open player list")
	errPathNotExist           = errors.New("path does not exist")
	errCreateMessage          = errors.New("failed to create user message")
	errSaveMessage            = errors.New("failed to save user message")
	errGetNames               = errors.New("failed to load name history")
	errSaveNames              = errors.New("failed to save name history")
	errGetPlayer              = errors.New("failed to load player record")
	errSavePlayer             = errors.New("failed to save player to database")
	errRCONConnect            = errors.New("failed to connect to game client RCON")
	errRCONStatus             = errors.New("failed to get status result")
	errRCONG15                = errors.New("failed to get g15 result")
	errRCONExec               = errors.New("failed to exec rcon command")
	errRCONRead               = errors.New("failed to read rcon response")
	errG15Parse               = errors.New("failed to parse g15 result")
	errInvalidChatType        = errors.New("invalid chat destination type")
	errInvalidReadyState      = errors.New("invalid ready state")
	errNotMarked              = errors.New("mark does not exist")
	errGameStopped            = errors.New("game is not running")
	errDiscordActivity        = errors.New("failed to set discord activity")
	errOpenDatabase           = errors.New("failed to open database")
	errCloseDatabase          = errors.New("failed to cleanly close database")
	errCloseWeb               = errors.New("failed to cleanly close web service")
	errParseTimestamp         = errors.New("failed to parse timestamp")
	errReaderG15              = errors.New("failed to read from g15 reader")
	errStoreIOFSOpen          = errors.New("failed to create migration iofs")
	errStoreIOFSClose         = errors.New("failed to close migration iofs")
	errStoreDriver            = errors.New("failed to create db driver")
	errCreateMigration        = errors.New("failed to create migrator")
	errPerformMigration       = errors.New("failed to migrate database")
	errStorePragma            = errors.New("failed to enable pragma")
	errGenerateQuery          = errors.New("failed to generate query")
	errScanResult             = errors.New("failed to scan results")
	errStoreExec              = errors.New("failed to exec query")
	errRows                   = errors.New("rows had error value")
	errInvalidSid             = errors.New("invalid steamid")
	errEmptyValue             = errors.New("value cannot be empty")
	errFetchPlayerList        = errors.New("failed to fetch player list")
	errSettingDirectoryCreate = errors.New("failed to initialize UserSettings directory")
	errSettingAddress         = errors.New("invalid address, cannot parse")
	errSettingsAPIKeyMissing  = errors.New("must set steam api key when not using bdapi")
	errSettingAPIKeyInvalid   = errors.New("invalid Steam API Key")
	errSettingsOpen           = errors.New("failed to open settings file")
	errSettingsDecode         = errors.New("failed to decode settings")
	errSettingsOpenOutput     = errors.New("failed to open UserSettings file for writing")
	errSettingsEncode         = errors.New("failed to encode settings")
	errVoiceBanVersion        = errors.New("invalid version")
	errVoiceBanReadVersion    = errors.New("failed to read binary version")
	errVoiceBanWriteVersion   = errors.New("failed to write binary version")
	errVoiceBanWriteSteamID   = errors.New("failed to write binary steamid data")
	errHTTPListen             = errors.New("HTTP server returned error")
	errHTTPRoutes             = errors.New("failed to setup static routes")
	errHTTPShutdown           = errors.New("failed to shutdown http service")
	errTempDir                = errors.New("failed to create temp dir")
	errTestApp                = errors.New("failed to create test app")
)

const (
	DurationStatusUpdateTimer = time.Second * 2

	DurationCheckTimer           = time.Second * 3
	DurationUpdateTimer          = time.Second * 1
	DurationAnnounceMatchTimeout = time.Minute * 5
	DurationCacheTimeout         = time.Hour * 12
	DurationWebRequestTimeout    = time.Second * 5
	DurationRCONRequestTimeout   = time.Second * 2
	DurationProcessTimeout       = time.Second * 3
)

type EventType int

const (
	EvtKill EventType = iota
	EvtMsg
	EvtConnect
	EvtDisconnect
	EvtStatusID
	EvtHostname
	EvtMap
	EvtTags
	EvtAddress
	EvtLobby
)

type KickReason string

const (
	KickReasonIdle     KickReason = "idle"
	KickReasonScamming KickReason = "scamming"
	KickReasonCheating KickReason = "cheating"
	KickReasonOther    KickReason = "other"
)

type ChatDest string

const (
	ChatDestAll   ChatDest = "all"
	ChatDestTeam  ChatDest = "team"
	ChatDestParty ChatDest = "party"
)

type Version struct {
	Version string
	Commit  string
	Date    string
	BuiltBy string
}
