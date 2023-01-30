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
	"github.com/leighmacdonald/bd/translations"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"log"
	"time"
)

const (
	settingKeySteamId = "steamId"

	AppId = "com.github.leighmacdonald.bd"
)

type UserInterface interface {
	OnUserMessage(value model.EvtUserMessage)
	OnServerState(value model.ServerState)
	Start()
	OnLaunchTF2(func())
}

type Ui struct {
	ctx            context.Context
	Application    fyne.App
	RootWindow     fyne.Window
	ChatWindow     fyne.Window
	SettingsDialog dialog.Dialog
	PlayerTable    *widget.Table
	AboutDialog    dialog.Dialog
	serverName     binding.String
	currentMap     binding.String
	messages       binding.StringList
	playerData     binding.UntypedList
	launcher       func()
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

func New(ctx context.Context) UserInterface {
	application := app.NewWithID(AppId)
	application.SetIcon(readIcon("./Icon.png"))
	rootWindow := application.NewWindow("Bot Detector")
	settingsDialog := newSettingsDialog(application, rootWindow, func() {
		rootWindow.Close()
	})
	aboutDialog := createAboutDialog(rootWindow)

	//var bindings []binding.DataMap
	//for _, p := range serverState.players {
	//	bindings = append(bindings, binding.BindStruct(&p))
	//}

	ui := Ui{
		ctx:            ctx,
		Application:    application,
		RootWindow:     rootWindow,
		SettingsDialog: settingsDialog,
		AboutDialog:    aboutDialog,
		messages:       binding.NewStringList(),
		currentMap:     binding.NewString(),
		serverName:     binding.NewString(),
		playerData:     binding.NewUntypedList(),
	}

	ui.ChatWindow = ui.newChatWidget()

	table := ui.newPlayerTable()
	playerTable := container.NewVScroll(table)
	ui.PlayerTable = table
	//ui.RootWindow.SetCloseIntercept(func() {
	//	ui.RootWindow.Hide()
	//})
	rootWindow.Resize(fyne.NewSize(750, 1000))

	ui.configureTray(func() {
		rootWindow.Show()
	})

	toolbar := ui.newToolbar(func() {
		ui.ChatWindow.Show()
	}, func() {
		settingsDialog.Show()
	}, func() {
		aboutDialog.Show()
	})

	rootWindow.SetContent(container.NewBorder(
		toolbar,
		nil,
		nil,
		nil,
		playerTable,
	))
	return &ui
}

func (ui *Ui) OnLaunchTF2(fn func()) {
	ui.launcher = fn
}

func (ui *Ui) Start() {
	ui.RootWindow.Show()
	go func() {
		time.Sleep(time.Second * 2)
		ui.Application.SendNotification(fyne.NewNotification("New Notification", "This is the content of a "+
			"test message"))
	}()
	ui.Application.Run()
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
	ui.ChatWindow.Content().(*widget.List).ScrollToBottom()
}

func (ui *Ui) OnServerState(value model.ServerState) {
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
	ui.RootWindow.Show()
	ui.Application.Run()
}

func newSettingsDialog(application fyne.App, parent fyne.Window, onClose func()) dialog.Dialog {
	defaultSteamId := application.Preferences().StringWithFallback(settingKeySteamId, "")
	settingSteamId := binding.BindString(&defaultSteamId)
	entry := widget.NewEntryWithData(settingSteamId)

	entry.Validator = func(s string) error {
		_, sidErr := steamid.SID64FromString(entry.Text)
		if sidErr != nil {
			return errors.New("Invalid steam64")
		}
		return nil
	}

	form := &widget.Form{
		Items: []*widget.FormItem{ // we can specify items in the constructor
			{Text: "Steam ID (steam64)", Widget: entry}},
		OnSubmit: func() {
			sid, sidErr := steamid.SID64FromString(entry.Text)
			if sidErr != nil {
				log.Println(sidErr)
				return
			}

			application.Preferences().SetString(settingKeySteamId, sid.String())
			onClose()
		},
	}
	settingsWindow := dialog.NewCustom("Settings", "Dismiss", form, parent)
	settingsWindow.Resize(fyne.NewSize(500, 500))
	return settingsWindow
}

func (ui *Ui) configureTray(showFunc func()) {
	launchLabel := translations.Tr(&i18n.Message{
		ID:  "LaunchButton",
		One: "Launch TF2",
	}, 1, nil)

	if desk, ok := ui.Application.(desktop.App); ok {
		m := fyne.NewMenu(ui.Application.Preferences().StringWithFallback("appName", "Bot Detector"),
			fyne.NewMenuItem("Show", showFunc),
			fyne.NewMenuItem(launchLabel, ui.launcher))
		desk.SetSystemTrayMenu(m)
		ui.Application.SetIcon(theme.InfoIcon())
	}
}

func (ui *Ui) newToolbar(chatFunc func(), settingsFunc func(), aboutFunc func()) *widget.Toolbar {
	toolBar := widget.NewToolbar(
		widget.NewToolbarAction(theme.MediaPlayIcon(), func() {
			log.Println("Launching game")
			ui.launcher()
		}),
		widget.NewToolbarAction(theme.DocumentIcon(), chatFunc),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.SettingsIcon(), settingsFunc),
		widget.NewToolbarAction(theme.HelpIcon(), func() {
			log.Println("Display help")
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
		return widget.NewLabel("template")
	}, func(item binding.DataItem, object fyne.CanvasObject) {
		object.(*widget.Label).Bind(item.(binding.String))
	})
	chatWindow := ui.Application.NewWindow("Chat")
	chatWindow.SetContent(chatWidget)
	chatWindow.Resize(fyne.NewSize(1000, 500))
	chatWindow.SetCloseIntercept(func() {
		chatWindow.Hide()
	})

	return chatWindow
}

func (ui *Ui) newPlayerTable() *widget.Table {
	//keys := []string{"userId", "steamId", "name", ""}

	table := widget.NewTable(func() (int, int) {
		return ui.playerData.Length(), 6
	}, func() fyne.CanvasObject {
		return widget.NewLabel("wide content")
	}, func(id widget.TableCellID, object fyne.CanvasObject) {
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
			object.(*widget.Label).SetText(fmt.Sprintf("%d", rv.UserId))
		case 1:
			object.(*widget.Label).SetText(rv.Name)
		case 2:
			object.(*widget.Label).SetText(rv.SteamId.String())
		case 3:
			object.(*widget.Label).SetText("0")
		case 4:
			object.(*widget.Label).SetText("0")
		case 5:
			object.(*widget.Label).SetText("0")

		}
	})
	for i, v := range []float32{50, 250, 200, 50, 50, 50} {
		table.SetColumnWidth(i, v)
	}
	return table
}

func createAboutDialog(parent fyne.Window) dialog.Dialog {
	aboutMsg := fmt.Sprintf("%s\n\nVersion: %s\nCommit: %s\nDate: %s\n", AppId, model.BuildVersion, model.BuildCommit, model.BuildDate)
	return dialog.NewInformation("About", aboutMsg, parent)
}
