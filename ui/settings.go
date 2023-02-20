package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"log"
	"path/filepath"
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

func (ui *Ui) newSettingsDialog(parent fyne.Window, onClose func()) dialog.Dialog {
	const testSteamId = 76561197961279983

	var createSelectorRow = func(label string, icon fyne.Resource, entry *widget.Entry, defaultPath string) *container.Split {
		fileInputContainer := container.NewHSplit(widget.NewButtonWithIcon("Edit", icon, func() {
			d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
				if err != nil || uri == nil {
					return
				}
				entry.SetText(uri.Path())
			}, ui.rootWindow)
			d.Show()
		}), entry)
		fileInputContainer.SetOffset(0.0)
		return fileInputContainer
	}

	apiKey := ui.settings.getBoundStringDefault("ApiKey", "")
	apiKeyOriginal, _ := apiKey.Get()
	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.Bind(apiKey)
	apiKeyEntry.Validator = func(s string) error {
		if len(apiKeyEntry.Text) > 0 && len(apiKeyEntry.Text) != 32 {
			return errors.New("Invalid api key")
		}
		// Wait until all validation is complete to keep the setting
		defer func() {
			_ = steamweb.SetKey(apiKeyOriginal)
		}()
		if apiKeyEntry.Text == "" {
			return nil
		}
		if errSetKey := steamweb.SetKey(apiKeyEntry.Text); errSetKey != nil {
			return errSetKey
		}
		res, errRes := steamweb.PlayerSummaries(steamid.Collection{testSteamId})
		if errRes != nil {
			log.Printf("Failed to fetch player summary for validation: %v", errRes)
			return errors.New("Could not validate api call")
		}
		if len(res) != 1 {
			return errors.New("Failed to fetch summary")
		}
		return nil
	}

	steamId := ui.settings.getBoundStringDefault("SteamId", "")
	steamIdEntry := widget.NewEntry()
	steamIdEntry.Bind(steamId)
	steamIdEntry.Validator = func(s string) error {
		if len(steamIdEntry.Text) > 0 {
			_, err := steamid.StringToSID64(steamIdEntry.Text)
			if err != nil {
				return errors.New("Invalid Steam ID")
			}
		}

		return nil
	}

	tf2Root := ui.settings.getBoundStringDefault("TF2Root", platform.DefaultTF2Root)
	tf2RootEntry := widget.NewEntryWithData(tf2Root)
	tf2RootEntry.Validator = func(s string) error {
		if len(tf2RootEntry.Text) > 0 {
			if !golib.Exists(tf2RootEntry.Text) {
				return errors.New("Path does not exist")
			}
			fp := filepath.Join(tf2RootEntry.Text, platform.TF2RootValidationFile)
			if !golib.Exists(fp) {
				return errors.Errorf("Could not find %s inside, invalid steam root", platform.TF2RootValidationFile)
			}
		}
		return nil
	}

	steamRoot := ui.settings.getBoundStringDefault("SteamRoot", platform.DefaultSteamRoot)
	steamRootEntry := widget.NewEntryWithData(steamRoot)
	steamRootEntry.Validator = func(s string) error {
		if len(steamRootEntry.Text) > 0 {
			if !golib.Exists(steamRootEntry.Text) {
				return errors.New("Path does not exist")
			}
			fp := filepath.Join(steamRootEntry.Text, platform.SteamRootValidationFile)
			if !golib.Exists(fp) {
				return errors.Errorf("Could not find %s inside, invalid steam root", platform.SteamRootValidationFile)
			}
			if tf2RootEntry.Text == "" {
				dp := filepath.Join(steamRootEntry.Text, "steamapps\\common\\Team Fortress 2\\tf")
				if golib.Exists(dp) {
					tf2RootEntry.SetText(dp)
				}
			}
		}
		return nil
	}

	kickerEnabled := ui.settings.getBoundBoolDefault("KickerEnabled", true)
	kickerEnabledEntry := widget.NewCheckWithData("", kickerEnabled)

	chatWarningsEnabled := ui.settings.getBoundBoolDefault("ChatWarningsEnabled", false)
	chatWarningsEnabledEntry := widget.NewCheckWithData("", chatWarningsEnabled)

	partyWarningsEnabled := ui.settings.getBoundBoolDefault("PartyWarningsEnabled", true)
	partyWarningsEnabledEntry := widget.NewCheckWithData("", partyWarningsEnabled)

	settingsForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Kicker Enabled", Widget: kickerEnabledEntry},
			{Text: "Chat Warning Enabled", Widget: chatWarningsEnabledEntry},
			{Text: "Party Warning Enabled", Widget: partyWarningsEnabledEntry},
			{Text: "Steam API Key", Widget: apiKeyEntry},
			{Text: "Steam ID", Widget: steamIdEntry},
			{Text: "Steam Root", Widget: createSelectorRow("Select", theme.FileTextIcon(), steamRootEntry, "")},
			{Text: "TF2 Root", Widget: createSelectorRow("Select", theme.FileTextIcon(), tf2RootEntry, "")},
		},
		OnSubmit: func() {
			defer onClose()
			// Update it to our preferred format
			newSid, errSid := steamid.StringToSID64(steamIdEntry.Text)
			if errSid != nil {
				// Should never happen? was validated previously.
				log.Panicf("Steamid state invalid?: %v\n", errSid)
			}
			steamIdEntry.SetText(newSid.String())

			ui.baseSettings.Lock()
			ui.baseSettings.ApiKey = apiKeyEntry.Text
			ui.baseSettings.SteamRoot = steamRootEntry.Text
			ui.baseSettings.TF2Root = tf2RootEntry.Text
			ui.baseSettings.SteamId = newSid.String()
			ui.baseSettings.KickerEnabled = kickerEnabledEntry.Checked
			ui.baseSettings.ChatWarningsEnabled = chatWarningsEnabledEntry.Checked
			ui.baseSettings.PartyWarningsEnabled = partyWarningsEnabledEntry.Checked
			ui.baseSettings.Unlock()
			if apiKeyOriginal != apiKeyEntry.Text {
				if errSetKey := steamweb.SetKey(apiKeyEntry.Text); errSetKey != nil {
					log.Printf("Failed to set new steam key: %v\n", errSetKey)
				}
			}
			ui.settingsDialog.Hide()
		},
	}

	settingsWindow := dialog.NewCustom("Settings", "Cancel", container.NewVScroll(settingsForm), parent)
	settingsWindow.Resize(fyne.NewSize(900, 500))
	return settingsWindow
}
