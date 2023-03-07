package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	clone "github.com/huandu/go-clone/generic"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/internal/translations"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"log"
	"path/filepath"
	"strings"
)

func newSettingsDialog(parent fyne.Window, origSettings *model.Settings) dialog.Dialog {
	const testSteamId = 76561197961279983

	settings := clone.Clone[*model.Settings](origSettings)

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
	apiKeyOriginal := settings.APIKey
	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.Bind(binding.BindString(&settings.APIKey))
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

	steamIdEntry := widget.NewEntry()
	steamIdEntry.Bind(binding.BindString(&settings.SteamID))
	steamIdEntry.Validator = validateSteamId

	tf2RootEntry := widget.NewEntryWithData(binding.BindString(&settings.TF2Dir))
	tf2RootEntry.Validator = validateSteamRoot

	steamDirEntry := widget.NewEntryWithData(binding.BindString(&settings.SteamDir))
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
	autoCloseOnGameExitEntry := widget.NewCheckWithData("", binding.BindBool(&settings.AutoCloseOnGameExit))
	autoLaunchGameEntry := widget.NewCheckWithData("", binding.BindBool(&settings.AutoLaunchGame))
	kickerEnabledEntry := widget.NewCheckWithData("", binding.BindBool(&settings.KickerEnabled))
	chatWarningsEnabledEntry := widget.NewCheckWithData("", binding.BindBool(&settings.ChatWarningsEnabled))
	partyWarningsEnabledEntry := widget.NewCheckWithData("", binding.BindBool(&settings.PartyWarningsEnabled))
	discordPresenceEnabledEntry := widget.NewCheckWithData("", binding.BindBool(&settings.DiscordPresenceEnabled))
	rconModeStaticEntry := widget.NewCheckWithData(translations.One(translations.CheckboxRconStatic), binding.BindBool(&settings.RCONStatic))
	staticConfig := model.NewRconConfig(true)
	boundTags := binding.NewString()
	if errSet := boundTags.Set(strings.Join(settings.GetKickTags(), ",")); errSet != nil {
		log.Printf("Failed to set tags: %v\n", errSet)
	}
	tagsEntry := widget.NewEntryWithData(boundTags)
	tagsEntry.Validator = validateTags
	linksDialog := newLinksDialog(parent, settings)
	linksButton := widget.NewButtonWithIcon("Edit Links", theme.SettingsIcon(), func() {
		linksDialog.Show()
	})
	linksButton.Alignment = widget.ButtonAlignLeading
	linksButton.Refresh()

	listsDialog := newRuleListConfigDialog(parent, settings)
	listsButton := widget.NewButtonWithIcon("Edit Lists", theme.SettingsIcon(), func() {
		listsDialog.Show()
	})
	listsButton.Alignment = widget.ButtonAlignLeading
	listsButton.Refresh()

	settingsForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Lists & Rules", Widget: listsButton, HintText: "Configure your 3rd party player and rule lists"},
			{Text: "External Links", Widget: linksButton, HintText: "Customize external links menu"},
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
			{Text: translations.One(translations.LabelAutoLaunchGame), Widget: autoLaunchGameEntry,
				HintText: translations.One(translations.LabelAutoLaunchGameHint)},
			{Text: translations.One(translations.LabelAutoCloseOnGameExit), Widget: autoCloseOnGameExitEntry,
				HintText: translations.One(translations.LabelAutoCloseOnGameExitHint)},
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
	onSave := func(status bool) {
		if !status {
			return
		}
		// Update it to our preferred format
		if steamIdEntry.Text != "" {
			newSid, errSid := steamid.StringToSID64(steamIdEntry.Text)
			if errSid != nil {
				// Should never happen? was validated previously.
				log.Panicf("Steamid state invalid?: %v\n", errSid)
			}
			origSettings.SetSteamID(newSid.String())
			steamIdEntry.SetText(newSid.String())
		}
		var newTags []string
		for _, t := range strings.Split(tagsEntry.Text, ",") {
			if t == "" {
				continue
			}
			newTags = append(newTags, strings.Trim(t, " "))
		}
		origSettings.SetKickTags(newTags)
		origSettings.SetAPIKey(apiKeyEntry.Text)
		origSettings.SetSteamDir(steamDirEntry.Text)
		origSettings.SetTF2Dir(tf2RootEntry.Text)
		origSettings.SetKickerEnabled(kickerEnabledEntry.Checked)
		origSettings.SetChatWarningsEnabled(chatWarningsEnabledEntry.Checked)
		origSettings.SetPartyWarningsEnabled(partyWarningsEnabledEntry.Checked)
		origSettings.SetRconStatic(rconModeStaticEntry.Checked)
		origSettings.SetAutoCloseOnGameExit(autoCloseOnGameExitEntry.Checked)
		origSettings.SetAutoLaunchGame(autoLaunchGameEntry.Checked)
		origSettings.SetLinks(settings.GetLinks())
		origSettings.SetLists(settings.GetLists())

		if apiKeyOriginal != apiKeyEntry.Text {
			if errSetKey := steamweb.SetKey(apiKeyEntry.Text); errSetKey != nil {
				log.Printf("Failed to set new steam key: %v\n", errSetKey)
			}
		}
		if errSave := origSettings.Save(); errSave != nil {
			log.Printf("Failed to save settings: %v\n", errSave)
		}
	}
	settingsWindow := dialog.NewCustomConfirm(
		translations.One(translations.TitleSettings),
		"save",
		translations.One(translations.LabelClose),
		container.NewVScroll(settingsForm),
		onSave,
		parent,
	)

	settingsForm.Refresh()
	settingsWindow.Resize(fyne.NewSize(800, 800))
	return settingsWindow
}
