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
	LabelLaunch           Key = "label_launch"
	LabelSettings         Key = "label_settings"
	LabelListConfig       Key = "label_list_config"
	LabelChatLog          Key = "label_chat_log"
	LabelConfigFolder     Key = "label_config_folder"
	LabelHelp             Key = "label_help"
	LabelQuit             Key = "label_quit"
	LabelAbout            Key = "label_about"
	LabelHostname         Key = "label_hostname"
	LabelMap              Key = "label_map"
	LabelClose            Key = "label_close"
	LabelApply            Key = "label_apply"
	LabelAutoScroll       Key = "label_auto_scroll"
	LabelBottom           Key = "label_bottom"
	LabelClear            Key = "label_clear"
	LabelMarkAs           Key = "label_mark_as"
	LabelSortBy           Key = "label_sort_by"
	LabelAttributeName    Key = "label_attribute_name"
	LabelMessageCount     Key = "label_message_count"
	MenuVoteCheating      Key = "menu_vote_cheating"
	MenuVoteIdle          Key = "menu_vote_idle"
	MenuVoteScamming      Key = "menu_vote_scamming"
	MenuVoteOther         Key = "menu_vote_other"
	MenuCallVote          Key = "menu_call_vote"
	MenuMarkAs            Key = "menu_mark_as"
	MenuOpenExternal      Key = "menu_open_external"
	MenuCopySteamId       Key = "menu_copy_steamid"
	MenuChatHistory       Key = "menu_chat_history"
	MenuNameHistory       Key = "menu_name_history"
	WindowNameHistory     Key = "window_name_history"
	WindowChatHistoryUser Key = "window_chat_history_user"
	WindowChatHistoryGame Key = "window_chat_history_game"
	WindowMarkCustom      Key = "window_mark_custom"
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
