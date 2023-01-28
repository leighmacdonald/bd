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

func Tr(message *i18n.Message, count int, tmplData map[string]interface{}) string {
	translation := localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: message,
		TemplateData:   tmplData,
		PluralCount:    count,
	})
	return translation
}

func init() {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	for _, langFile := range []string{"active.en.yaml"} {
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
