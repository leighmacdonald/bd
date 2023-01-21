package main

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
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"log"
	"time"
)

const (
	settingKeySteamId = "steamId"
)

type BotDetector struct {
	application    fyne.App
	rootWindow     fyne.Window
	settingsDialog dialog.Dialog
	logChan        chan string
	messages       binding.StringList
	chatWindow     fyne.Window
	serverState    serverState
	ctx            context.Context
	playerLists    []TF2BDSchema
}

func New() BotDetector {
	application := app.NewWithID(AppId)
	rootApp := BotDetector{
		application: application,
		logChan:     make(chan string),
		messages:    binding.NewStringList(),
		serverState: serverState{
			players: map[steamid.SID64]*player{},
		},
		ctx: context.Background(),
	}
	rootApp.createRootWindow()
	go func() {
		for {
			msg := <-rootApp.logChan
			if errAppend := rootApp.messages.Append(msg); errAppend != nil {
				log.Printf("Failed to add message: %v\n", errAppend)
			}
			rootApp.chatWindow.Content().(*widget.List).ScrollToBottom()
		}
	}()
	go func() {
		tick := time.NewTicker(time.Second * 10)
		for {
			<-tick.C
			updatePlayerState(rootApp.ctx, &rootApp.serverState)
			for _, v := range rootApp.serverState.players {
				log.Printf("%d, %d, %s, %v, %s\n", v.userId, v.steamId, v.name, v.team, v.connectedTime)
			}
		}
	}()
	go func() {
		rootApp.playerLists = downloadPlayerLists(rootApp.ctx)
		tick := time.NewTicker(1 * time.Hour)
		for {
			<-tick.C
			rootApp.playerLists = downloadPlayerLists(rootApp.ctx)
		}
	}()
	go func() {
		count := 0
		t := time.NewTicker(time.Second)
		for {
			<-t.C
			rootApp.logChan <- formatMsg(fmt.Sprintf("Test message #%d", count))
			count++
		}
	}()
	return rootApp
}

func (bd *BotDetector) start() {
	bd.rootWindow.Show()
	bd.application.Run()
}

func (bd *BotDetector) newSettingsDialog() dialog.Dialog {
	defaultSteamId := bd.application.Preferences().StringWithFallback(settingKeySteamId, "")
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

			bd.application.Preferences().SetString(settingKeySteamId, sid.String())
			bd.rootWindow.Close()
		},
	}
	settingsWindow := dialog.NewCustom("Settings", "Dismiss", form, bd.rootWindow)
	settingsWindow.Resize(fyne.NewSize(500, 500))
	return settingsWindow
}

func (bd *BotDetector) configureTray() {
	if desk, ok := bd.application.(desktop.App); ok {
		m := fyne.NewMenu(AppName,
			fyne.NewMenuItem("Show", func() {
				bd.rootWindow.Show()
			}),
			fyne.NewMenuItem("Launch TF2", func() {
				launchTF2()
			}))
		desk.SetSystemTrayMenu(m)
	}
}

func (bd *BotDetector) newToolbar() *widget.Toolbar {
	toolBar := widget.NewToolbar(
		widget.NewToolbarAction(theme.MediaPlayIcon(), launchTF2),
		widget.NewToolbarAction(theme.FileTextIcon(), func() {
			bd.chatWindow.Show()
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.SettingsIcon(), func() {
			bd.settingsDialog.Show()
		}),
		widget.NewToolbarAction(theme.HelpIcon(), func() {
			log.Println("Display help")
		}),
	)
	return toolBar
}

func formatMsg(msg string) string {
	return fmt.Sprintf("%s: %s", time.Now().Format("15:04:05"), msg)
}

func (bd *BotDetector) newChatWidget() fyne.Window {
	chatWidget := widget.NewListWithData(bd.messages, func() fyne.CanvasObject {
		return widget.NewLabel("template")
	}, func(item binding.DataItem, object fyne.CanvasObject) {
		object.(*widget.Label).Bind(item.(binding.String))
	})
	chatWindow := bd.application.NewWindow("Chat")
	chatWindow.SetContent(chatWidget)
	chatWindow.Resize(fyne.NewSize(1000, 500))
	chatWindow.SetCloseIntercept(func() {
		chatWindow.Hide()
	})

	return chatWindow
}

func (bd *BotDetector) newPlayerTable() *widget.Table {
	keys := []string{"userId", "steamId", "name", ""}
	var bindings []binding.DataMap
	for _, p := range bd.serverState.players {
		bindings = append(bindings, binding.BindStruct(&p))
	}
	table := widget.NewTable(func() (int, int) {
		return 24, 6
	}, func() fyne.CanvasObject {
		return widget.NewLabel("wide content")
	}, func(id widget.TableCellID, object fyne.CanvasObject) {
		if id.Row > len(bd.serverState.players)-1 {
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

func (bd *BotDetector) createRootWindow() {
	bd.rootWindow = bd.application.NewWindow(AppName)
	bd.settingsDialog = bd.newSettingsDialog()
	bd.configureTray()
	bd.chatWindow = bd.newChatWidget()
	playerTable := container.NewVScroll(bd.newPlayerTable())

	centerContainer := container.NewBorder(
		bd.newToolbar(),
		nil,
		nil,
		nil,
		playerTable,
	)

	bd.rootWindow.SetContent(centerContainer)

	//bd.rootWindow.SetCloseIntercept(func() {
	//	bd.rootWindow.Hide()
	//})
	bd.rootWindow.Resize(fyne.NewSize(750, 1000))
}
