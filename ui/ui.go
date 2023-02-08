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
	"fyne.io/fyne/v2/canvas"
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
	"image/color"
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
	Refresh()
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
	gameState      *model.GameState
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

func New(ctx context.Context, settings *model.Settings, gameState *model.GameState) UserInterface {
	application := app.NewWithID(AppId)
	application.Settings().SetTheme(&bdTheme{})
	application.SetIcon(readIcon("ui/resources/Icon.png"))
	rootWindow := application.NewWindow("Bot Detector")
	ui := Ui{
		ctx:          ctx,
		application:  application,
		rootWindow:   rootWindow,
		gameState:    gameState,
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

func (ui *Ui) Refresh() {
	cw := ui.chatWindow.Content().(*chatListWidget)
	cw.ScrollToBottom()
	cw.Refresh()
	ui.PlayerTable.Refresh()
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
			(*ui.gameState).Messages = nil
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
		widget.NewToolbarAction(theme.MediaPlayIcon(), func() {
			ui.Refresh()
		}),
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

type chatListWidget struct {
	widget.List
}

func (ui *Ui) newChatListWidget() *chatListWidget {
	cl := &chatListWidget{
		List: widget.List{
			Length: func() int {
				ui.gameState.RLock()
				defer ui.gameState.RUnlock()
				return len(ui.gameState.Messages)
			},
			CreateItem: func() fyne.CanvasObject {
				return container.NewHSplit(widget.NewLabel(""), widget.NewLabel(""))
			},
			UpdateItem: func(id widget.ListItemID, item fyne.CanvasObject) {
				ui.gameState.RLock()
				defer ui.gameState.RUnlock()
				if id+1 > len(ui.gameState.Messages) {
					return
				}
				itm := ui.gameState.Messages[id]
				cnt := item.(*container.Split)
				a := cnt.Leading.(*widget.Label)
				a.SetText(itm.Created.Format("3:04PM"))
				b := cnt.Trailing.(*widget.Label)
				b.SetText(itm.Message)
			},
			OnSelected: func(id widget.ListItemID) {
				log.Printf("Selected row: %v\n", id)

			},
			OnUnselected: func(id widget.ListItemID) {
				log.Printf("Unselected row: %v\n", id)
			},
		},
	}
	cl.ExtendBaseWidget(cl)
	return cl
}

func (ui *Ui) newChatWidget() fyne.Window {
	chatWidget := ui.newChatListWidget()
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
	defText := color.RGBA{
		R: 100,
		G: 100,
		B: 100,
		A: 50,
	}
	columns := []float32{50, 24, 300, 50, 50, 50}
	table := widget.NewTable(func() (int, int) {
		ui.gameState.RLock()
		defer ui.gameState.RUnlock()
		return len(ui.gameState.Players), len(columns)
	}, func() fyne.CanvasObject {
		return container.NewMax(canvas.NewText("", defText), ui.newTableButtonLabel())
	}, func(id widget.TableCellID, object fyne.CanvasObject) {
		label := object.(*fyne.Container).Objects[0].(*canvas.Text)
		icon := object.(*fyne.Container).Objects[1].(*tableButtonLabel)
		label.Show()
		icon.Hide()
		ui.gameState.RLock()
		defer ui.gameState.RUnlock()
		if ui.gameState.Players == nil || id.Row+1 > len(ui.gameState.Players) {
			label.Text = ""
			//label.Refresh()
			return
		}
		if id.Row > len(ui.gameState.Players)-1 {
			label.Text = "no value"
			//label.Refresh()
			return
		}
		value := ui.gameState.Players[id.Row]

		if icon.menu == nil {
			icon.menu = generateUserMenu(value.SteamId, ui.application.OpenURL, ui.rootWindow.Clipboard())
		}

		switch id.Col {
		case 0:
			label.TextStyle.Symbol = false
			label.TextStyle.Monospace = true
			label.Alignment = fyne.TextAlignCenter
			label.Text = fmt.Sprintf("%04d", value.UserId)
			//label.Refresh()
		case 1:
			label.Hide()
			icon.Show()
		case 2:
			label.TextStyle.Bold = true
			label.Text = value.Name
			label.Color = color.RGBA{
				R: 200,
				G: 20,
				B: 20,
				A: 255,
			}
		case 3:
			label.Alignment = fyne.TextAlignCenter
			label.Text = fmt.Sprintf("%d", value.Ping)
		case 4:
			label.Alignment = fyne.TextAlignCenter
			label.Text = fmt.Sprintf("%d", value.KillsOn)
		case 5:
			label.Alignment = fyne.TextAlignCenter
			label.Text = fmt.Sprintf("%d", value.DeathsBy)
		}
	})
	for i, v := range columns {
		table.SetColumnWidth(i, v)
	}

	return table
}

func createAboutDialog(parent fyne.Window) dialog.Dialog {
	u, _ := url.Parse(urlHome)
	aboutMsg := fmt.Sprintf("%s\n\nVersion: %s\nCommit: %s\nDate: %s\n", AppId, model.BuildVersion, model.BuildCommit, model.BuildDate)
	vbox := container.NewVBox(
		widget.NewLabel(aboutMsg),
		widget.NewHyperlink(urlHome, u),
	)
	return dialog.NewCustom("About", "Close", vbox, parent)
}
