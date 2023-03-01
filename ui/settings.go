package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/bd/translations"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"log"
	"path/filepath"
	"strings"
)

type boundSettings struct {
	binding.Struct
}

func (s *boundSettings) getBoundStringDefault(key string, def string) binding.String {
	value, apiKeyErr := s.GetValue(key)
	if apiKeyErr != nil {
		value = def
	}
	v := value.(string)
	return binding.BindString(&v)
}

func (s *boundSettings) getBoundBoolDefault(key string, def bool) binding.Bool {
	value, apiKeyErr := s.GetValue(key)
	if apiKeyErr != nil {
		value = def
	}
	v := value.(bool)
	return binding.BindBool(&v)
}

func newSettingsDialog(parent fyne.Window, boundSettings boundSettings, settings *model.Settings) dialog.Dialog {
	const testSteamId = 76561197961279983

	var createSelectorRow = func(label string, icon fyne.Resource, entry *widget.Entry, defaultPath string) *container.Split {
		fileInputContainer := container.NewHSplit(widget.NewButtonWithIcon("Edit", icon, func() {
			d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
				if err != nil || uri == nil {
					return
				}
				entry.SetText(uri.Path())
			}, parent)
			d.Show()
		}), entry)
		fileInputContainer.SetOffset(0.0)
		return fileInputContainer
	}

	apiKey := boundSettings.getBoundStringDefault("ApiKey", "")
	apiKeyOriginal, _ := apiKey.Get()
	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.Bind(apiKey)
	apiKeyEntry.Validator = func(newApiKey string) error {
		if len(newApiKey) > 0 && len(newApiKey) != 32 {
			return errors.New(translations.One(translations.ErrorInvalidApiKey))
		}
		// Wait until all validation is complete to keep the setting
		defer func() {
			_ = steamweb.SetKey(apiKeyOriginal)
		}()
		if newApiKey == "" {
			return nil
		}
		if errSetKey := steamweb.SetKey(newApiKey); errSetKey != nil {
			return errSetKey
		}
		res, errRes := steamweb.PlayerSummaries(steamid.Collection{testSteamId})
		if errRes != nil {
			log.Printf("Failed to fetch player summary for validation: %v", errRes)
			return errors.New(translations.One(translations.ErrorValidateAPICall))
		}
		if len(res) != 1 {
			log.Printf("Received incorrect number of steam api validation call\n")
			return errors.New(translations.One(translations.ErrorValidateAPICall))
		}
		return nil
	}

	steamId := boundSettings.getBoundStringDefault("SteamID", "")
	steamIdEntry := widget.NewEntry()
	steamIdEntry.Bind(steamId)
	steamIdEntry.Validator = validateSteamId

	tf2Dir := boundSettings.getBoundStringDefault("TF2Dir", platform.DefaultTF2Root)
	tf2RootEntry := widget.NewEntryWithData(tf2Dir)
	tf2RootEntry.Validator = validateSteamRoot

	steamDir := boundSettings.getBoundStringDefault("SteamDir", platform.DefaultSteamRoot)
	steamDirEntry := widget.NewEntryWithData(steamDir)
	steamDirEntry.Validator = func(newRoot string) error {
		if len(newRoot) > 0 {
			if !golib.Exists(newRoot) {
				return errors.New(translations.One(translations.ErrorInvalidPath))
			}
			userDataDir := filepath.Join(newRoot, "userdata")
			if !golib.Exists(userDataDir) {
				return errors.New(translations.One(translations.ErrorInvalidSteamDirUserData))
			}
			if tf2RootEntry.Text == "" {
				dp := filepath.Join(newRoot, "steamapps/common/Team Fortress 2/tf")
				if errValid := validateSteamRoot(dp); errValid == nil && golib.Exists(dp) {
					tf2RootEntry.SetText(dp)
				}
			}
		}
		return nil
	}

	kickerEnabled := boundSettings.getBoundBoolDefault("KickerEnabled", true)
	kickerEnabledEntry := widget.NewCheckWithData("", kickerEnabled)

	chatWarningsEnabled := boundSettings.getBoundBoolDefault("ChatWarningsEnabled", false)
	chatWarningsEnabledEntry := widget.NewCheckWithData("", chatWarningsEnabled)

	partyWarningsEnabled := boundSettings.getBoundBoolDefault("PartyWarningsEnabled", true)
	partyWarningsEnabledEntry := widget.NewCheckWithData("", partyWarningsEnabled)

	discordPresenceEnabled := boundSettings.getBoundBoolDefault("DiscordPresenceEnabled", false)
	discordPresenceEnabledEntry := widget.NewCheckWithData("", discordPresenceEnabled)

	rconModeStatic := boundSettings.getBoundBoolDefault("RconStatic", false)
	rconModeStaticEntry := widget.NewCheckWithData(translations.One(translations.CheckboxRconStatic), rconModeStatic)

	staticConfig := model.NewRconConfig(true)
	boundTags := binding.NewString()
	if errSet := boundTags.Set(strings.Join(settings.KickTags, ",")); errSet != nil {
		log.Printf("Failed to set tags: %v\n", errSet)
	}

	tagsEntry := widget.NewEntryWithData(boundTags)
	tagsEntry.Validator = validateTags

	settingsForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: translations.One(translations.LabelSettingsVoteKicker), Widget: kickerEnabledEntry,
				HintText: translations.One(translations.LabelSettingsVoteKickerHint)},
			{Text: translations.One(translations.LabelSettingsKickableTags), Widget: tagsEntry,
				HintText: translations.One(translations.LabelSettingsKickableTagsHint)},
			{Text: translations.One(translations.LabelSettingsChatWarnings), Widget: chatWarningsEnabledEntry,
				HintText: translations.One(translations.LabelSettingsChatWarningsHint)},
			{Text: translations.One(translations.LabelSettingsPartyWarnings), Widget: partyWarningsEnabledEntry,
				HintText: translations.One(translations.LabelSettingsPartyWarningsHint)},
			{Text: translations.One(translations.LabelSettingsDiscordPresence), Widget: discordPresenceEnabledEntry,
				HintText: translations.One(translations.LabelSettingsDiscordPresenceHint)},
			{Text: translations.One(translations.LabelSettingsSteamApiKey), Widget: apiKeyEntry,
				HintText: translations.One(translations.LabelSettingsSteamApiKeyHint)},
			{Text: translations.One(translations.LabelSettingsSteamId), Widget: steamIdEntry,
				HintText: translations.One(translations.LabelSettingsSteamIdHint)},
			{Text: translations.One(translations.LabelSettingsSteamRoot),
				Widget:   createSelectorRow(translations.One(translations.LabelSelect), theme.FileTextIcon(), steamDirEntry, ""),
				HintText: translations.One(translations.LabelSettingsSteamRootHint)},
			{Text: translations.One(translations.LabelSettingsTF2Root),
				Widget:   createSelectorRow(translations.One(translations.LabelSelect), theme.FileTextIcon(), tf2RootEntry, ""),
				HintText: translations.One(translations.LabelSettingsTF2RootHint)},
			{Text: translations.One(translations.LabelSettingsRCONMode), Widget: rconModeStaticEntry,
				HintText: translations.Tr(&i18n.Message{ID: string(translations.LabelSettingsRCONModeHint)},
					1, map[string]interface{}{"Port": staticConfig.Port(), "Password": staticConfig.Password()}),
			},
		},
	}

	settingsWindow := dialog.NewCustom(
		translations.One(translations.TitleSettings),
		translations.One(translations.LabelClose),
		container.NewVScroll(settingsForm),
		parent,
	)

	settingsForm.OnSubmit = func() {
		settings.Lock()
		// Update it to our preferred format
		if steamIdEntry.Text != "" {
			newSid, errSid := steamid.StringToSID64(steamIdEntry.Text)
			if errSid != nil {
				// Should never happen? was validated previously.
				log.Panicf("Steamid state invalid?: %v\n", errSid)
			}
			settings.SteamID = newSid.String()
			steamIdEntry.SetText(newSid.String())
		}
		var newTags []string
		for _, t := range strings.Split(tagsEntry.Text, ",") {
			if t == "" {
				continue
			}
			newTags = append(newTags, strings.Trim(t, " "))
		}
		settings.KickTags = newTags
		settings.ApiKey = apiKeyEntry.Text
		settings.SteamDir = steamDirEntry.Text
		settings.TF2Dir = tf2RootEntry.Text
		settings.KickerEnabled = kickerEnabledEntry.Checked
		settings.ChatWarningsEnabled = chatWarningsEnabledEntry.Checked
		settings.PartyWarningsEnabled = partyWarningsEnabledEntry.Checked
		settings.RconStatic = rconModeStaticEntry.Checked
		settings.Unlock()
		if apiKeyOriginal != apiKeyEntry.Text {
			if errSetKey := steamweb.SetKey(apiKeyEntry.Text); errSetKey != nil {
				log.Printf("Failed to set new steam key: %v\n", errSetKey)
			}
		}
		if errSave := settings.Save(); errSave != nil {
			log.Printf("Failed to save settings: %v\n", errSave)
		}
		settingsWindow.Hide()
	}
	settingsForm.Refresh()
	settingsWindow.Resize(fyne.NewSize(750, 800))
	return settingsWindow
}
