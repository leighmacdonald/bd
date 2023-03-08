package tr

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
	Localizer *i18n.Localizer
)

func init() {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	for _, langFile := range []string{"active.en.yaml", "active.ru.yaml"} {
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

	Localizer = i18n.NewLocalizer(bundle, validLanguages...)
}
