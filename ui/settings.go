package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/model"
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

	steamId := boundSettings.getBoundStringDefault("SteamID", "")
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

	tf2Dir := boundSettings.getBoundStringDefault("TF2Dir", platform.DefaultTF2Root)
	tf2RootEntry := widget.NewEntryWithData(tf2Dir)
	validateSteamDir := func(s string) error {
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
	tf2RootEntry.Validator = validateSteamDir

	steamDir := boundSettings.getBoundStringDefault("SteamDir", platform.DefaultSteamRoot)
	steamDirEntry := widget.NewEntryWithData(steamDir)
	steamDirEntry.Validator = func(s string) error {
		if len(steamDirEntry.Text) > 0 {
			if !golib.Exists(steamDirEntry.Text) {
				return errors.New("Path does not exist")
			}
			userDataDir := filepath.Join(steamDirEntry.Text, "userdata")
			if !golib.Exists(userDataDir) {
				return errors.New("THe userdata folder not found in steam dir")
			}
			if tf2RootEntry.Text == "" {
				dp := filepath.Join(steamDirEntry.Text, "steamapps/common/Team Fortress 2/tf")
				if errValid := validateSteamDir(dp); errValid == nil && golib.Exists(dp) {
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
	rconModeStaticEntry := widget.NewCheckWithData("Static", rconModeStatic)

	staticConfig := model.NewRconConfig(true)

	settingsForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Vote Kicker", Widget: kickerEnabledEntry, HintText: "Enable vote kick functionality in-game"},
			{Text: "Chat Warnings", Widget: chatWarningsEnabledEntry, HintText: "Show warning message using in-game chat"},
			{Text: "Party Warnings", Widget: partyWarningsEnabledEntry, HintText: "Show lobby only warning messages"},
			{Text: "Discord Presence", Widget: discordPresenceEnabledEntry, HintText: "Enables discord rich presence if discord is running"},
			{Text: "Steam API Key", Widget: apiKeyEntry, HintText: "Steam web api key. https://steamcommunity.com/dev/apikey"},
			{Text: "Steam ID", Widget: steamIdEntry, HintText: "Your steam id in any of the following formats: steam,steam3,steam32,steam64"},
			{Text: "Steam Root", Widget: createSelectorRow("Select", theme.FileTextIcon(), steamDirEntry, ""),
				HintText: "Location of your steam install directory containing a userdata folder."},
			{Text: "TF2 Root", Widget: createSelectorRow("Select", theme.FileTextIcon(), tf2RootEntry, ""),
				HintText: "Path to your steamapps/common/Team Fortress 2/tf folder"},
			{Text: "RCON Mode", Widget: rconModeStaticEntry,
				HintText: fmt.Sprintf("Static: Port: %d, Password: %s", staticConfig.Port(), staticConfig.Password())},
		},
		OnSubmit: func() {
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
		},
	}

	settingsWindow := dialog.NewCustom("Settings", "Cancel", container.NewVScroll(settingsForm), parent)
	settingsWindow.Resize(fyne.NewSize(750, 700))
	return settingsWindow
}
