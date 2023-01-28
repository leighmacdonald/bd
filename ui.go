package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/translations"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"log"
)

const (
	settingKeySteamId = "steamId"

	AppId = "com.github.leighmacdonald.bd"
)

type UserInterface interface {
	onNewMessage(value evtUserMessageData)
}

type Ui struct {
	Application    fyne.App
	RootWindow     fyne.Window
	ChatWindow     fyne.Window
	SettingsDialog dialog.Dialog
	AboutDialog    dialog.Dialog
	messages       binding.StringList
	GameLauncher   func()
}

func (ui *Ui) onNewMessage(value evtUserMessageData) {
	teamMsg := "blu"
	if value.team == red {
		teamMsg = "red"
	}
	outMsg := fmt.Sprintf("[%s] %s: %s", teamMsg, value.player, value.message)
	if errAppend := ui.messages.Append(outMsg); errAppend != nil {
		log.Printf("Failed to add message: %v\n", errAppend)
	}
	ui.ChatWindow.Content().(*widget.List).ScrollToBottom()
}

func newUi(state *serverState) *Ui {
	application := app.NewWithID(AppId)
	rootWindow := application.NewWindow("")
	settingsDialog := newSettingsDialog(application, rootWindow, func() {
		rootWindow.Close()
	})
	aboutDialog := createAboutDialog(rootWindow)
	messages := binding.NewStringList()
	chatWindow := newChatWidget(application, messages)
	var bindings []binding.DataMap
	//for _, p := range serverState.players {
	//	bindings = append(bindings, binding.BindStruct(&p))
	//}
	playerTable := container.NewVScroll(newPlayerTable(state, bindings))

	//ui.RootWindow.SetCloseIntercept(func() {
	//	ui.RootWindow.Hide()
	//})
	rootWindow.Resize(fyne.NewSize(750, 1000))

	ui := Ui{
		Application:    application,
		RootWindow:     rootWindow,
		SettingsDialog: settingsDialog,
		AboutDialog:    aboutDialog,
		ChatWindow:     chatWindow,
	}

	configureTray(application, ui.GameLauncher, func() {
		rootWindow.Show()
	})

	toolbar := newToolbar(ui.GameLauncher, func() {
		chatWindow.Show()
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

func configureTray(application fyne.App, launchTf2Fn func(), showFunc func()) {
	launchLabel := translations.Tr(&i18n.Message{
		ID:  "LaunchButton",
		One: "Launch TF2",
	}, 1, nil)

	if desk, ok := application.(desktop.App); ok {
		m := fyne.NewMenu(application.Preferences().StringWithFallback("appName", "Bot Detector"),
			fyne.NewMenuItem("Show", showFunc),
			fyne.NewMenuItem(launchLabel, launchTf2Fn))
		desk.SetSystemTrayMenu(m)
		application.SetIcon(theme.InfoIcon())
	}
}

func newToolbar(tf2LaunchFn func(), chatFunc func(), settingsFunc func(), aboutFunc func()) *widget.Toolbar {
	toolBar := widget.NewToolbar(
		widget.NewToolbarAction(theme.MediaPlayIcon(), tf2LaunchFn),
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

func newChatWidget(application fyne.App, messages binding.StringList) fyne.Window {
	chatWidget := widget.NewListWithData(messages, func() fyne.CanvasObject {
		return widget.NewLabel("template")
	}, func(item binding.DataItem, object fyne.CanvasObject) {
		object.(*widget.Label).Bind(item.(binding.String))
	})
	chatWindow := application.NewWindow("Chat")
	chatWindow.SetContent(chatWidget)
	chatWindow.Resize(fyne.NewSize(1000, 500))
	chatWindow.SetCloseIntercept(func() {
		chatWindow.Hide()
	})

	return chatWindow
}

func newPlayerTable(serverState *serverState, bindings []binding.DataMap) *widget.Table {
	keys := []string{"userId", "steamId", "name", ""}

	table := widget.NewTable(func() (int, int) {
		return 24, 6
	}, func() fyne.CanvasObject {
		return widget.NewLabel("wide content")
	}, func(id widget.TableCellID, object fyne.CanvasObject) {
		if id.Row > len(serverState.players)-1 {
			object.(*widget.Label).SetText("")
			return
		}
		if bindings == nil {
			object.(*widget.Label).SetText("")
			return
		}
		value := bindings[id.Row]

		//found := playerState[id.Row]
		label := object.(*widget.Label)
		newValue, err := value.GetItem(keys[id.Col])
		if err != nil {
			log.Println(err)
			label.SetText(err.Error())
			return
		}
		label.Bind(newValue.(binding.String))
	})
	for i, v := range []float32{50, 250, 75, 75, 200} {
		table.SetColumnWidth(i, v)
	}
	return table
}

func createAboutDialog(parent fyne.Window) dialog.Dialog {
	aboutMsg := fmt.Sprintf("%s\n\nVersion: %s\nCommit: %s\nDate: %s\n", AppId, version, commit, date)
	return dialog.NewInformation("About", aboutMsg, parent)
}
