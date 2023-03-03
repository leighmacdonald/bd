package translations

import (
	"embed"
	"github.com/jeandeaual/go-locale"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
	"log"
)

//go:embed *.yaml
var localeFS embed.FS

var (
	bundle    *i18n.Bundle
	localizer *i18n.Localizer
)

type Key string

const (
	CheckboxRconStatic               Key = "checkbox_rcon_static"
	LabelLaunch                      Key = "label_launch"
	LabelSettings                    Key = "label_settings"
	LabelListConfig                  Key = "label_list_config"
	LabelChatLog                     Key = "label_chat_log"
	LabelConfigFolder                Key = "label_config_folder"
	LabelHelp                        Key = "label_help"
	LabelQuit                        Key = "label_quit"
	LabelAbout                       Key = "label_about"
	LabelHostname                    Key = "label_hostname"
	LabelMap                         Key = "label_map"
	LabelClose                       Key = "label_close"
	LabelApply                       Key = "label_apply"
	LabelAutoScroll                  Key = "label_auto_scroll"
	LabelBottom                      Key = "label_bottom"
	LabelClear                       Key = "label_clear"
	LabelMarkAs                      Key = "label_mark_as"
	LabelSortBy                      Key = "label_sort_by"
	LabelDelete                      Key = "label_delete"
	LabelEdit                        Key = "label_edit"
	LabelEnabled                     Key = "label_enabled"
	LabelAttributeName               Key = "label_attribute_name"
	LabelMessageCount                Key = "label_message_count"
	LabelAboutBuiltBy                Key = "label_about_built_by"
	LabelAboutBuildDate              Key = "label_about_build_date"
	LabelAboutVersion                Key = "label_about_version"
	LabelAboutCommit                 Key = "label_about_commit"
	LabelName                        Key = "label_name"
	LabelURL                         Key = "label_url"
	LabelAdd                         Key = "label_add"
	LabelSelect                      Key = "label_select"
	LabelSettingsVoteKicker          Key = "label_settings_vote_kicker"
	LabelSettingsVoteKickerHint      Key = "label_settings_vote_kicker_hint"
	LabelSettingsKickableTags        Key = "label_settings_kickable_tags"
	LabelSettingsKickableTagsHint    Key = "label_settings_kickable_tags_hint"
	LabelSettingsChatWarnings        Key = "label_settings_chat_warnings"
	LabelSettingsChatWarningsHint    Key = "label_settings_chat_warnings_hint"
	LabelSettingsPartyWarnings       Key = "label_settings_party_warnings"
	LabelSettingsPartyWarningsHint   Key = "label_settings_party_warnings_hint"
	LabelSettingsDiscordPresence     Key = "label_settings_discord_presence"
	LabelAutoCloseOnGameExit         Key = "label_auto_close_on_game_exit"
	LabelAutoCloseOnGameExitHint     Key = "label_auto_close_on_game_exit_hint"
	LabelAutoLaunchGame              Key = "label_auto_launch_game"
	LabelAutoLaunchGameHint          Key = "label_auto_launch_game_hint"
	LabelSettingsDiscordPresenceHint Key = "label_settings_discord_presence_hint"
	LabelSettingsSteamApiKey         Key = "label_settings_steam_api_key"
	LabelSettingsSteamApiKeyHint     Key = "label_settings_steam_api_key_hint"
	LabelSettingsSteamId             Key = "label_settings_steam_id"
	LabelSettingsSteamIdHint         Key = "label_settings_steam_id_hint"
	LabelSettingsSteamRoot           Key = "label_settings_steam_root"
	LabelSettingsSteamRootHint       Key = "label_settings_steam_root_hint"
	LabelSettingsTF2Root             Key = "label_settings_tf2_root"
	LabelSettingsTF2RootHint         Key = "label_settings_tf2_root_hint"
	LabelSettingsRCONMode            Key = "label_settings_rcon_mode"
	LabelSettingsRCONModeHint        Key = "label_settings_rcon_mode_hint"
	LabelConfirmDeleteList           Key = "label_confirm_delete_list"
	TitleDeleteConfirm               Key = "title_delete_confirm"
	TitleImportUrl                   Key = "title_import_url"
	TitleListConfig                  Key = "title_list_config"
	MenuVoteCheating                 Key = "menu_vote_cheating"
	MenuVoteIdle                     Key = "menu_vote_idle"
	MenuVoteScamming                 Key = "menu_vote_scamming"
	MenuVoteOther                    Key = "menu_vote_other"
	MenuCallVote                     Key = "menu_call_vote"
	MenuMarkAs                       Key = "menu_mark_as"
	MenuOpenExternal                 Key = "menu_open_external"
	MenuCopySteamId                  Key = "menu_copy_steamid"
	MenuChatHistory                  Key = "menu_chat_history"
	MenuWhitelist                    Key = "menu_whitelist"
	MenuNameHistory                  Key = "menu_name_history"
	WindowNameHistory                Key = "window_name_history"
	WindowChatHistoryUser            Key = "window_chat_history_user"
	WindowChatHistoryGame            Key = "window_chat_history_game"
	TitleSettings                    Key = "title_settings"
	WindowMarkCustom                 Key = "window_mark_custom"
	ErrorNameEmpty                   Key = "error_name_empty"
	ErrorInvalidURL                  Key = "error_invalid_url"
	ErrorAttributeEmpty              Key = "error_attribute_empty"
	ErrorAttributeDuplicate          Key = "error_attribute_duplicate"
	ErrorNoNamesFound                Key = "error_no_names_found"
	ErrorSteamIdMisconfigured        Key = "error_steam_id_misconfigured"
	ErrorInvalidApiKey               Key = "error_invalid_api_key"
	ErrorInvalidSteamId              Key = "error_invalid_steam_id"
	ErrorInvalidPath                 Key = "error_invalid_path"
	ErrorInvalidSteamRoot            Key = "error_invalid_steam_root"
	ErrorInvalidSteamDirUserData     Key = "error_invalid_steam_root_user_data"
	ErrorValidateAPICall             Key = "error_validate_api_call"
)

func Tr(message *i18n.Message, count int, tmplData map[string]interface{}) string {
	translation := localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: message,
		TemplateData:   tmplData,
		PluralCount:    count,
	})
	return translation
}

func One(key Key) string {
	return Tr(&i18n.Message{ID: string(key)}, 1, nil)
}

func init() {
	bundle = i18n.NewBundle(language.Russian)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	for _, langFile := range []string{"active.en.yaml", "ru.yaml"} {
		_, errLoad := bundle.LoadMessageFileFS(localeFS, langFile)
		if errLoad != nil {
			log.Fatalf(errLoad.Error())
		}
	}
	userLocales, err := locale.GetLocales()
	if err != nil {
		log.Printf("Failed to load user locale: %v\n", err)
		userLocales = append(userLocales, "en-GB")
	}
	var validLanguages []string
	for _, ul := range userLocales {
		langTag, langTagErr := language.Parse(ul)
		if langTagErr != nil {
			log.Printf("Failed to parse language tag: %s %v\n", ul, langTagErr)
			continue
		}
		validLanguages = append(validLanguages, langTag.String())
	}

	localizer = i18n.NewLocalizer(bundle, validLanguages...)
}
