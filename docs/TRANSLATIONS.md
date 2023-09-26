# Editing Translations

## New Language

1. Make an empty translation file. e.g. for french: `internal/tr/translate.fr.yaml`
2. Generate translation file: `goi18n merge .\internal\tr\active.en.yaml .\internal\tr\translate.fr.yaml`
3. Edit `.\internal\tr\translate.fr.yaml` with translations
4. Rename `.\internal\tr\translate.fr.yaml` to `.\internal\tr\active.fr.yaml`
5. Merge changes: `make tr_merge`

## Updated Messages

1. `make tr_extract`
2. `make tr_gen_translate`
3. Edit updated `interlal/tr/translate.*.yaml` files
4. `make tr_merge`

See [go-i18n](https://github.com/nicksnyder/go-i18n) for more detailed instructions.
