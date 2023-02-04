// Package ui provides a simple, cross-platform interface to the bot detector tool
//
// TODO
// - Use external data map/struct(?) for table data updates
// - Remove old players from state on configurable delay
package ui

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
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
	"net/url"
	"path/filepath"
	"time"
)

const (
	AppId   = "com.github.leighmacdonald.bd"
	urlHome = "https://github.com/leighmacdonald/bd"
	urlHelp = "https://github.com/leighmacdonald/bd/wiki"
)

type UserInterface interface {
	OnUserMessage(value model.EvtUserMessage)
	OnServerState(value *model.ServerState)
	OnDisconnect(sid64 steamid.SID64)
	Start()
	OnLaunchTF2(func())
}

type Ui struct {
	ctx            context.Context
	application    fyne.App
	rootWindow     fyne.Window
	chatWindow     fyne.Window
	settingsDialog dialog.Dialog
	PlayerTable    *widget.Table
	aboutDialog    dialog.Dialog
	serverName     binding.String
	currentMap     binding.String
	messages       binding.StringList
	playerData     binding.UntypedList
	settings       boundSettings
	baseSettings   *model.Settings
	launcher       func()
}

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

func readIcon(path string) fyne.Resource {
	r, re := fyne.LoadResourceFromPath(path)
	if re != nil {
		log.Println(re.Error())
		// Fallback
		return theme.InfoIcon()
	}
	return r
}

func New(ctx context.Context, settings *model.Settings) UserInterface {
	application := app.NewWithID(AppId)
	application.Settings().SetTheme(&bdTheme{})
	application.SetIcon(readIcon("ui/resources/Icon.png"))
	rootWindow := application.NewWindow("Bot Detector")

	ui := Ui{
		ctx:          ctx,
		application:  application,
		rootWindow:   rootWindow,
		messages:     binding.NewStringList(),
		currentMap:   binding.NewString(),
		serverName:   binding.NewString(),
		playerData:   binding.NewUntypedList(),
		settings:     boundSettings{binding.BindStruct(settings)},
		baseSettings: settings,
	}
	ui.settingsDialog = ui.newSettingsDialog(rootWindow, func() {
		if errSave := settings.Save(); errSave != nil {
			log.Printf("Failed to save config file: %v\n", errSave)
			return
		}
		log.Println("Settings saved successfully")
	})
	ui.aboutDialog = createAboutDialog(rootWindow)
	ui.chatWindow = ui.newChatWidget()
	table := ui.newPlayerTableWidget()
	playerTable := container.NewVScroll(table)
	ui.PlayerTable = table
	//ui.rootWindow.SetCloseIntercept(func() {
	//	ui.rootWindow.Hide()
	//})
	rootWindow.Resize(fyne.NewSize(750, 1000))

	ui.configureTray(func() {
		rootWindow.Show()
	})

	toolbar := ui.newToolbar(func() {
		ui.chatWindow.Show()
	}, func() {
		ui.settingsDialog.Show()
	}, func() {
		ui.aboutDialog.Show()
	})

	rootWindow.SetContent(container.NewBorder(
		toolbar,
		nil,
		nil,
		nil,
		playerTable,
	))
	rootWindow.SetMainMenu(ui.newMainMenu())
	return &ui
}

func (ui *Ui) newMainMenu() *fyne.MainMenu {
	wikiUrl, _ := url.Parse(urlHelp)
	fm := fyne.NewMenu("Bot Detector",
		fyne.NewMenuItem("Settings", func() {
			ui.settingsDialog.Show()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() {
			ui.application.Quit()
		}),
	)
	am := fyne.NewMenu("Actions",
		fyne.NewMenuItem("Clear", func() {
			if errSet := ui.messages.Set([]string{}); errSet != nil {
				log.Println("Failed to clear chat messages")
			}
		}),
	)
	hm := fyne.NewMenu("Help",
		fyne.NewMenuItem("Help", func() {
			if errOpenHelp := ui.application.OpenURL(wikiUrl); errOpenHelp != nil {
				log.Printf("Failed to open help url: %v\n", errOpenHelp)
			}

		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("About", func() {
			ui.aboutDialog.Show()
		}),
	)
	return fyne.NewMainMenu(fm, am, hm)
}

func (ui *Ui) OnLaunchTF2(fn func()) {
	ui.launcher = fn
}

func (ui *Ui) Start() {
	ui.rootWindow.Show()
	ui.application.Run()
}

func (ui *Ui) OnDisconnect(sid64 steamid.SID64) {
	go func(sid steamid.SID64) {
		time.Sleep(time.Second * 60)
		//
	}(sid64)
	log.Printf("Player disconnected: %d", sid64.Int64())
}

func (ui *Ui) OnUserMessage(value model.EvtUserMessage) {
	teamMsg := "blu"
	if value.Team == model.Red {
		teamMsg = "red"
	}
	outMsg := fmt.Sprintf("[%s] %s: %s", teamMsg, value.Player, value.Message)
	if errAppend := ui.messages.Append(outMsg); errAppend != nil {
		log.Printf("Failed to add message: %v\n", errAppend)
	}
	ui.chatWindow.Content().(*widget.List).ScrollToBottom()
}

func (ui *Ui) OnServerState(value *model.ServerState) {
	if errSetServer := ui.serverName.Set(value.Server); errSetServer != nil {
		log.Printf("Failed to update server name: %v", errSetServer)
	}
	if errSetCurrentMap := ui.currentMap.Set(value.CurrentMap); errSetCurrentMap != nil {
		log.Printf("Failed to update current map: %v", errSetCurrentMap)
	}
	var players []any
	for _, x := range value.Players {
		players = append(players, x)
	}
	if errPlayerState := ui.playerData.Set(players); errPlayerState != nil {
		log.Printf("Failed to update player state: %v", errPlayerState)
	}
	ui.PlayerTable.Refresh()
}

func (ui *Ui) Run() {
	ui.rootWindow.Show()
	ui.application.Run()
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

func (ui *Ui) configureTray(showFunc func()) {
	launchLabel := translations.Tr(&i18n.Message{
		ID:  "LaunchButton",
		One: "Launch TF2",
	}, 1, nil)

	if desk, ok := ui.application.(desktop.App); ok {
		m := fyne.NewMenu(ui.application.Preferences().StringWithFallback("appName", "Bot Detector"),
			fyne.NewMenuItem("Show", showFunc),
			fyne.NewMenuItem(launchLabel, ui.launcher))
		desk.SetSystemTrayMenu(m)
		ui.application.SetIcon(theme.InfoIcon())
	}
}

func (ui *Ui) newToolbar(chatFunc func(), settingsFunc func(), aboutFunc func()) *widget.Toolbar {
	wikiUrl, _ := url.Parse(urlHelp)
	toolBar := widget.NewToolbar(
		widget.NewToolbarAction(theme.MediaPlayIcon(), func() {
			log.Println("Launching game")
			ui.launcher()
		}),
		widget.NewToolbarAction(theme.DocumentIcon(), chatFunc),
		//widget.NewToolbarSeparator(),
		//widget.NewToolbarAction(theme.MediaPlayIcon(), chatFunc),
		//widget.NewToolbarAction(theme.FileTextIcon(), chatFunc),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.SettingsIcon(), settingsFunc),
		widget.NewToolbarAction(theme.HelpIcon(), func() {
			if errOpenHelp := ui.application.OpenURL(wikiUrl); errOpenHelp != nil {
				log.Printf("Failed to open help url: %v\n", errOpenHelp)
			}
		}),
		widget.NewToolbarAction(theme.InfoIcon(), aboutFunc),
	)
	return toolBar
}

//func formatMsgDate(msg string) string {
//	return fmt.Sprintf("%s: %s", time.Now().Format("15:04:05"), msg)
//}

func (ui *Ui) newChatWidget() fyne.Window {
	chatWidget := widget.NewListWithData(ui.messages, func() fyne.CanvasObject {
		return newContextMenuLabel("template")
	}, func(item binding.DataItem, object fyne.CanvasObject) {
		object.(*contextMenuLabel).Bind(item.(binding.String))
	})
	chatWindow := ui.application.NewWindow("Chat")
	chatWindow.SetContent(chatWidget)
	chatWindow.Resize(fyne.NewSize(1000, 500))
	chatWindow.SetCloseIntercept(func() {
		chatWindow.Hide()
	})

	return chatWindow
}

// newPlayerTableWidget will configure and return a new player table widget.
// TODO: Investigate if its worth it to bother with binding this, external binding
// may be better?
func (ui *Ui) newPlayerTableWidget() *widget.Table {
	table := widget.NewTable(func() (int, int) {
		return ui.playerData.Length(), 6
	}, func() fyne.CanvasObject {
		return container.NewMax(widget.NewLabel(""), newTableButtonLabel(""))
	}, func(id widget.TableCellID, object fyne.CanvasObject) {
		label := object.(*fyne.Container).Objects[0].(*widget.Label)
		icon := object.(*fyne.Container).Objects[1].(*tableButtonLabel)
		label.Show()
		icon.Hide()
		if ui.playerData == nil || id.Row+1 > ui.playerData.Length() {
			object.(*widget.Label).SetText("")
			return
		}
		if id.Row > ui.playerData.Length()-1 {
			object.(*widget.Label).SetText("no value")
			return
		}
		value, valueErr := ui.playerData.GetValue(id.Row)
		if valueErr != nil {
			object.(*widget.Label).SetText("err")
			return
		}
		rv, ok := value.(model.PlayerState)
		if !ok {
			object.(*widget.Label).SetText("cast err")
			return
		}
		switch id.Col {
		case 0:
			label.TextStyle.Symbol = true
			label.TextStyle.Monospace = true
			label.SetText(fmt.Sprintf("%04d", rv.UserId))
		case 1:
			label.TextStyle.Monospace = true
			label.SetText(rv.SteamId.String())
		case 2:
			label.TextStyle.Bold = true
			label.SetText(rv.Name)
		case 3:
			label.Hide()
			icon.Show()
			icon.SetResource(theme.AccountIcon())
		case 4:
			label.SetText("0")
		}
	})
	for i, v := range []float32{50, 200, 300, 24, 50} {
		table.SetColumnWidth(i, v)
		//table.SetRowHeight(i, 24)
	}
	return table
}

func createAboutDialog(parent fyne.Window) dialog.Dialog {
	u, _ := url.Parse(urlHome)
	aboutMsg := fmt.Sprintf("%s\n\nVersion: %s\nCommit: %s\nDate: %s\n", AppId, model.BuildVersion, model.BuildCommit, model.BuildDate)
	c := container.NewVBox(
		widget.NewLabel(aboutMsg),
		widget.NewHyperlink(urlHome, u),
	)

	return dialog.NewCustom("About", "Close", c, parent)
}
