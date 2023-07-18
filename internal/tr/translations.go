package tr

import (
	"embed"

	"github.com/jeandeaual/go-locale"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
)

//go:embed *.yaml
var localeFS embed.FS

type Translator struct {
	bundle *i18n.Bundle
	*i18n.Localizer
}

func NewTranslator() (*Translator, error) {
	const defaultLocale = "en-GB"

	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

	for _, langFile := range []string{"active.en.yaml", "active.ru.yaml"} {
		_, errLoad := bundle.LoadMessageFileFS(localeFS, langFile)
		if errLoad != nil {
			return nil, errors.Wrap(errLoad, "Failed to load message bundle")
		}
	}

	userLocales, err := locale.GetLocales()
	if err != nil {
		userLocales = append(userLocales, defaultLocale)
	}

	validLanguages := make([]string, len(userLocales))

	for index, userLocale := range userLocales {
		langTag, langTagErr := language.Parse(userLocale)
		if langTagErr != nil {
			// Fallback to our default
			if langTag, langTagErr = language.Parse(defaultLocale); langTagErr != nil {
				return nil, errors.Wrapf(langTagErr, "Failed to parse language tag: %s", userLocale)
			}
		}

		validLanguages[index] = langTag.String()
	}

	return &Translator{
		bundle:    bundle,
		Localizer: i18n.NewLocalizer(bundle, validLanguages...),
	}, nil
}
