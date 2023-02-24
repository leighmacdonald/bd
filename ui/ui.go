// Package ui provides a simple, cross-platform interface to the bot detector tool
//
// TODO
// - Use external data map/struct(?) for table data updates
// - Remove old players from state on configurable delay
package ui

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
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/bd/platform"
	"github.com/leighmacdonald/bd/translations"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"log"
	"net/url"
	"sort"
	"strings"
)

const (
	AppId   = "com.github.leighmacdonald.bd"
	urlHome = "https://github.com/leighmacdonald/bd"
	urlHelp = "https://github.com/leighmacdonald/bd/wiki"
)

type UserInterface interface {
	Refresh()
	Start()
	SetOnLaunchTF2(func())
	SetOnMark(model.MarkFunc)
	SetOnKick(kickFunc model.KickFunc)
	SetFetchMessageHistory(messagesFunc model.QueryUserMessagesFunc)
	SetFetchNameHistory(namesFunc model.QueryNamesFunc)
	UpdateServerState(state model.ServerState)
	UpdateTitle(string)
	UpdatePlayerState([]model.PlayerState)
	AddUserMessage(message model.UserMessage)
	UpdateAttributes([]string)
}

type Ui struct {
	application           fyne.App
	rootWindow            fyne.Window
	chatWindow            fyne.Window
	settingsDialog        dialog.Dialog
	aboutDialog           dialog.Dialog
	boundSettings         boundSettings
	settings              *model.Settings
	playerList            *PlayerList
	userMessageList       *userMessageList
	knownAttributes       []string
	launcher              func()
	markFn                model.MarkFunc
	kickFn                model.KickFunc
	queryNamesFunc        model.QueryNamesFunc
	queryUserMessagesFunc model.QueryUserMessagesFunc
	labelHostname         *widget.RichText
	labelMap              *widget.RichText
	chatHistoryWindows    map[steamid.SID64]fyne.Window
	nameHistoryWindows    map[steamid.SID64]fyne.Window
	playerSortDir         playerSortType
}

func New(settings *model.Settings) UserInterface {
	application := app.NewWithID(AppId)
	application.Settings().SetTheme(&bdTheme{})
	application.SetIcon(resourceIconPng)
	rootWindow := application.NewWindow("Bot Detector")

	ui := Ui{
		application:        application,
		rootWindow:         rootWindow,
		boundSettings:      boundSettings{binding.BindStruct(settings)},
		settings:           settings,
		chatHistoryWindows: map[steamid.SID64]fyne.Window{},
		nameHistoryWindows: map[steamid.SID64]fyne.Window{},
		playerSortDir:      playerSortStatus,
	}

	ui.settingsDialog = ui.newSettingsDialog(rootWindow, func() {
		if errSave := settings.Save(); errSave != nil {
			log.Printf("Failed to save config file: %v\n", errSave)
			return
		}
		log.Println("Settings saved successfully")
	})
	ui.aboutDialog = createAboutDialog(rootWindow)
	ui.playerList = ui.createPlayerList()
	ui.userMessageList = ui.createGameChatMessageList()
	ui.chatWindow = ui.createChatWidget(ui.userMessageList)

	rootWindow.Resize(fyne.NewSize(800, 1000))
	ui.rootWindow.SetCloseIntercept(func() {
		application.Quit()
	})

	ui.configureTray(func() {
		ui.rootWindow.Show()
	})

	toolbar := ui.newToolbar(func() {
		ui.chatWindow.Show()
	}, func() {
		ui.settingsDialog.Show()
	}, func() {
		ui.aboutDialog.Show()
	})
	ui.labelHostname = widget.NewRichText(
		&widget.TextSegment{Text: "Hostname: ", Style: widget.RichTextStyleInline},
		&widget.TextSegment{Text: "n/a", Style: widget.RichTextStyleStrong},
	)
	ui.labelMap = widget.NewRichText(
		&widget.TextSegment{Text: "Map: ", Style: widget.RichTextStyleInline},
		&widget.TextSegment{Text: "n/a", Style: widget.RichTextStyleStrong},
	)

	statPanel := container.NewHBox(ui.labelMap, ui.labelHostname)
	var dirNames []string
	for _, dir := range sortDirections {
		dirNames = append(dirNames, string(dir))
	}
	sortSelect := widget.NewSelect(dirNames, func(s string) {
		ui.playerSortDir = playerSortType(s)
		v, _ := ui.playerList.boundList.Get()
		var sorted []model.PlayerState
		for _, p := range v {
			sorted = append(sorted, p.(model.PlayerState))
		}
		ui.UpdatePlayerState(sorted)
	})
	sortSelect.PlaceHolder = "Sort By..."
	heading := container.NewBorder(nil, nil, toolbar, sortSelect, container.NewCenter(widget.NewLabel("")))

	rootWindow.SetContent(container.NewBorder(
		heading,
		statPanel,
		nil,
		nil,
		ui.playerList.Widget(),
	))
	rootWindow.SetMainMenu(ui.newMainMenu())
	return &ui
}

func (ui *Ui) SetFetchMessageHistory(messagesFunc model.QueryUserMessagesFunc) {
	ui.queryUserMessagesFunc = messagesFunc
}

func (ui *Ui) SetFetchNameHistory(namesFunc model.QueryNamesFunc) {
	ui.queryNamesFunc = namesFunc
}

func (ui *Ui) SetOnMark(fn model.MarkFunc) {
	ui.markFn = fn
}

func (ui *Ui) SetOnKick(fn model.KickFunc) {
	ui.kickFn = fn
}

func (ui *Ui) Refresh() {
	ui.userMessageList.Widget().Refresh()
	ui.playerList.Widget().Refresh()
}

func (ui *Ui) UpdateAttributes(attrs []string) {
	ui.knownAttributes = attrs
}

func (ui *Ui) UpdateTitle(title string) {
	ui.rootWindow.SetTitle(title)
}

func (ui *Ui) UpdateServerState(state model.ServerState) {
	ui.labelHostname.Segments = []widget.RichTextSegment{
		&widget.TextSegment{Text: "Hostname: ", Style: widget.RichTextStyleInline},
		&widget.TextSegment{Text: state.ServerName, Style: widget.RichTextStyleStrong},
	}
	ui.labelHostname.Refresh()
	ui.labelMap.Segments = []widget.RichTextSegment{
		&widget.TextSegment{Text: "Map: ", Style: widget.RichTextStyleInline},
		&widget.TextSegment{Text: state.CurrentMap, Style: widget.RichTextStyleStrong},
	}
	ui.labelMap.Refresh()
}

type playerSortType string

const (
	playerSortName   playerSortType = "Name"
	playerSortKills  playerSortType = "Kills"
	playerSortKD     playerSortType = "K:D"
	playerSortStatus playerSortType = "Status"
)

var sortDirections = []playerSortType{playerSortName, playerSortKills, playerSortKD, playerSortStatus}

func (ui *Ui) UpdatePlayerState(players []model.PlayerState) {
	switch ui.playerSortDir {
	case playerSortKills:
		sort.Slice(players, func(i, j int) bool {
			return players[i].Kills > players[j].Kills
		})
	case playerSortStatus:
		sort.Slice(players, func(i, j int) bool {
			l := players[i]
			r := players[j]
			if l.NumberOfVACBans > r.NumberOfVACBans {
				return true
			} else if l.NumberOfGameBans > r.NumberOfGameBans {
				return true
			} else if l.CommunityBanned && !r.CommunityBanned {
				return true
			} else if l.EconomyBan && !r.EconomyBan {
				return true
			}
			return false
		})
	case playerSortKD:
		sort.Slice(players, func(i, j int) bool {
			l, r := 0.0, 0.0
			lk := players[i].Kills
			ld := players[i].Deaths
			if ld > 0 {
				l = float64(lk) / float64(ld)
			} else {
				l = float64(lk)
			}
			rk := players[j].Kills
			rd := players[j].Deaths
			if rd > 0 {
				r = float64(rk) / float64(rd)
			} else {
				r = float64(rk)
			}

			return l > r
		})
	default:
		sort.Slice(players, func(i, j int) bool {
			return strings.ToLower(players[i].Name) < strings.ToLower(players[j].Name)
		})
	}
	if errReboot := ui.playerList.Reload(players); errReboot != nil {
		log.Printf("Faile to reboot data: %v\n", errReboot)
	}
}

func (ui *Ui) AddUserMessage(msg model.UserMessage) {
	if errAppend := ui.userMessageList.Append(msg); errAppend != nil {
		log.Printf("Failed to append user message: %v", errAppend)
	}
	ui.userMessageList.Widget().Refresh()
}

func (ui *Ui) createChatHistoryWindow(sid64 steamid.SID64) error {
	_, found := ui.chatHistoryWindows[sid64]
	if found {
		ui.chatHistoryWindows[sid64].Show()
	} else {
		window := ui.application.NewWindow(fmt.Sprintf("Chat History: %d", sid64))
		window.SetOnClosed(func() {
			delete(ui.chatHistoryWindows, sid64)
		})
		messages, errMessage := ui.queryUserMessagesFunc(sid64)
		if errMessage != nil {
			return errors.Wrap(errMessage, "Failed to fetch user message history")
		}
		msgList := ui.createUserHistoryMessageList()
		if errReload := msgList.Reload(messages); errReload != nil {
			return errors.Wrap(errMessage, "Failed to reload user message history")
		}
		window.SetContent(msgList.Widget())
		window.Resize(fyne.NewSize(600, 600))
		window.Show()
		ui.chatHistoryWindows[sid64] = window
	}
	return nil
}

func (ui *Ui) createNameHistoryWindow(sid64 steamid.SID64) error {
	_, found := ui.nameHistoryWindows[sid64]
	if found {
		ui.nameHistoryWindows[sid64].Show()
	} else {
		window := ui.application.NewWindow(fmt.Sprintf("Name History: %d", sid64))
		window.SetOnClosed(func() {
			delete(ui.nameHistoryWindows, sid64)
		})
		names, errMessage := ui.queryNamesFunc(sid64)
		if errMessage != nil {
			return errors.Wrap(errMessage, "Failed to fetch user message history")
		}
		msgList := ui.createUserNameList()
		if errReload := msgList.Reload(names); errReload != nil {
			return errors.Wrap(errMessage, "Failed to reload user message history")
		}
		window.SetContent(msgList.Widget())
		window.Resize(fyne.NewSize(600, 600))
		window.Show()
		ui.nameHistoryWindows[sid64] = window
	}
	return nil
}

func (ui *Ui) newMainMenu() *fyne.MainMenu {
	wikiUrl, _ := url.Parse(urlHelp)
	fm := fyne.NewMenu("Bot Detector",
		&fyne.MenuItem{
			Shortcut: &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl},
			Label:    "Settings",
			Action: func() {
				ui.settingsDialog.Show()
			},
			Icon: theme.SettingsIcon(),
		},
		fyne.NewMenuItemSeparator(),
		&fyne.MenuItem{
			Icon:     theme.ContentUndoIcon(),
			Shortcut: &desktop.CustomShortcut{KeyName: fyne.KeyX, Modifier: fyne.KeyModifierControl},
			Label:    "Exit",
			IsQuit:   true,
			Action: func() {
				ui.application.Quit()
			},
		},
	)
	hm := fyne.NewMenu("Help",
		&fyne.MenuItem{
			Label:    "Help",
			Shortcut: &desktop.CustomShortcut{KeyName: fyne.KeyF1},
			Icon:     theme.HelpIcon(),
			Action: func() {
				if errOpenHelp := ui.application.OpenURL(wikiUrl); errOpenHelp != nil {
					log.Printf("Failed to open help url: %v\n", errOpenHelp)
				}
			}},
		&fyne.MenuItem{
			Label:    "About",
			Shortcut: &desktop.CustomShortcut{KeyName: fyne.KeyF10},
			Icon:     theme.InfoIcon(),
			Action: func() {
				ui.aboutDialog.Show()
			}},
	)
	return fyne.NewMainMenu(fm, hm)
}

func (ui *Ui) SetOnLaunchTF2(fn func()) {
	ui.launcher = fn
}

func (ui *Ui) Start() {
	ui.rootWindow.Show()
	ui.application.Run()
}

func (ui *Ui) OnDisconnect(sid64 steamid.SID64) {
	go func(sid steamid.SID64) {
		//time.Sleep(time.Second * 60)
		//
	}(sid64)
	log.Printf("Player disconnected: %d", sid64.Int64())
}

func (ui *Ui) Run() {
	ui.rootWindow.Show()
	ui.application.Run()
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

func showUserError(msg string, parent fyne.Window) {
	d := dialog.NewError(errors.New(msg), parent)
	d.Show()
}

func (ui *Ui) newToolbar(chatFunc func(), settingsFunc func(), aboutFunc func()) *widget.Toolbar {
	wikiUrl, _ := url.Parse(urlHelp)
	toolBar := widget.NewToolbar(
		widget.NewToolbarAction(resourceTf2Png, func() {
			log.Println("Launching game")
			if !ui.settings.GetSteamId().Valid() {
				showUserError("Must configure your steamid", ui.rootWindow)
			} else {
				ui.launcher()
			}
		}),
		widget.NewToolbarAction(theme.MailComposeIcon(), chatFunc),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.SettingsIcon(), settingsFunc),
		widget.NewToolbarAction(theme.FolderOpenIcon(), func() {
			platform.OpenFolder(ui.settings.ConfigRoot())
		}),
		widget.NewToolbarAction(theme.HelpIcon(), func() {
			if errOpenHelp := ui.application.OpenURL(wikiUrl); errOpenHelp != nil {
				log.Printf("Failed to open help url: %v\n", errOpenHelp)
			}
		}),
		widget.NewToolbarAction(theme.InfoIcon(), aboutFunc),
	)
	return toolBar
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
