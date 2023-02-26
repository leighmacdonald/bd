// Package ui provides a simple, cross-platform interface to the bot detector tool
//
// TODO
// - Use external data map/struct(?) for table data updates
// - Remove old players from state on configurable delay
package ui

import (
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
	"runtime"
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
	SetBuildInfo(version string, commit string, date string, builtBy string)
	SetOnLaunchTF2(func())
	SetOnMark(model.MarkFunc)
	SetOnKick(kickFunc model.KickFunc)
	SetFetchMessageHistory(messagesFunc model.QueryUserMessagesFunc)
	SetFetchNameHistory(namesFunc model.QueryNamesFunc)
	UpdateServerState(state model.Server)
	UpdateTitle(string)
	UpdatePlayerState(collection model.PlayerCollection)
	AddUserMessage(message model.UserMessage)
	UpdateAttributes([]string)
}

type Ui struct {
	application           fyne.App
	rootWindow            fyne.Window
	chatWindow            fyne.Window
	settingsDialog        dialog.Dialog
	listsDialog           dialog.Dialog
	aboutDialog           dialog.Dialog
	boundSettings         boundSettings
	settings              *model.Settings
	playerList            *baseListWidget
	userMessageList       *baseListWidget
	knownAttributes       []string
	launcher              func()
	markFn                model.MarkFunc
	kickFn                model.KickFunc
	queryNamesFunc        model.QueryNamesFunc
	queryUserMessagesFunc model.QueryUserMessagesFunc
	labelHostname         *widget.RichText
	labelMap              *widget.RichText
	labelBuiltBy          *widget.RichText
	labelDate             *widget.RichText
	labelVersion          *widget.RichText
	labelCommit           *widget.RichText
	labelGo               *widget.RichText
	chatHistoryWindows    map[steamid.SID64]*userChatContainer
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
		chatHistoryWindows: map[steamid.SID64]*userChatContainer{},
		nameHistoryWindows: map[steamid.SID64]fyne.Window{},
		labelBuiltBy:       widget.NewRichTextWithText("Built By: "),
		labelDate:          widget.NewRichTextWithText("Build Date: "),
		labelVersion:       widget.NewRichTextWithText("Version: "),
		labelCommit:        widget.NewRichTextWithText("Commit: "),
		labelGo: widget.NewRichText(
			&widget.TextSegment{Text: "Go ", Style: widget.RichTextStyleInline},
			&widget.TextSegment{Text: runtime.Version(), Style: widget.RichTextStyleStrong},
		),
		playerSortDir: playerSortStatus,
	}

	saveFunc := func() {
		if errSave := settings.Save(); errSave != nil {
			log.Printf("Failed to save config file: %v\n", errSave)
			return
		}
		log.Println("Settings saved successfully")
	}
	ui.settingsDialog = ui.newSettingsDialog(rootWindow, saveFunc)
	ui.listsDialog = newRuleListConfigDialog(rootWindow, saveFunc, settings)
	ui.aboutDialog = ui.createAboutDialog(rootWindow)
	ui.playerList = ui.createPlayerList()
	ui.userMessageList = ui.createGameChatMessageList()
	ui.chatWindow = createChatWidget(ui.application, ui.userMessageList)

	rootWindow.Resize(fyne.NewSize(800, 1000))
	ui.rootWindow.SetCloseIntercept(func() {
		application.Quit()
	})

	toolbar := ui.newToolbar(func() {
		ui.chatWindow.Show()
	}, func() {
		ui.settingsDialog.Show()
	}, func() {
		ui.aboutDialog.Show()
	})

	ui.labelHostname = widget.NewRichText(
		&widget.TextSegment{Text: translations.One(translations.LabelHostname), Style: widget.RichTextStyleInline},
		&widget.TextSegment{Text: "n/a", Style: widget.RichTextStyleStrong},
	)
	ui.labelMap = widget.NewRichText(
		&widget.TextSegment{Text: translations.One(translations.LabelMap), Style: widget.RichTextStyleInline},
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
		var sorted []model.Player
		for _, p := range v {
			sorted = append(sorted, p.(model.Player))
		}
		ui.UpdatePlayerState(sorted)
	})
	sortSelect.PlaceHolder = translations.One(translations.LabelSortBy)
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

func (ui *Ui) SetBuildInfo(version string, commit string, date string, builtBy string) {
	if len(ui.labelVersion.Segments) == 1 {
		ui.labelVersion.Segments = append(ui.labelVersion.Segments, &widget.TextSegment{
			Style: widget.RichTextStyleStrong,
			Text:  version,
		})
		ui.labelVersion.Refresh()
	}
	if len(ui.labelCommit.Segments) == 1 {
		ui.labelCommit.Segments = append(ui.labelCommit.Segments, &widget.TextSegment{
			Style: widget.RichTextStyleStrong,
			Text:  commit,
		})
		ui.labelCommit.Refresh()
	}
	if len(ui.labelDate.Segments) == 1 {
		ui.labelDate.Segments = append(ui.labelDate.Segments, &widget.TextSegment{
			Style: widget.RichTextStyleStrong,
			Text:  date,
		})
		ui.labelDate.Refresh()
	}
	if len(ui.labelBuiltBy.Segments) == 1 {
		ui.labelBuiltBy.Segments = append(ui.labelBuiltBy.Segments, &widget.TextSegment{
			Style: widget.RichTextStyleStrong,
			Text:  builtBy,
		})
		ui.labelBuiltBy.Refresh()
	}
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

func (ui *Ui) UpdateServerState(state model.Server) {
	ui.labelHostname.Segments = []widget.RichTextSegment{
		&widget.TextSegment{Text: translations.One(translations.LabelHostname), Style: widget.RichTextStyleInline},
		&widget.TextSegment{Text: state.ServerName, Style: widget.RichTextStyleStrong},
	}
	ui.labelHostname.Refresh()
	ui.labelMap.Segments = []widget.RichTextSegment{
		&widget.TextSegment{Text: translations.One(translations.LabelMap), Style: widget.RichTextStyleInline},
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
	playerSortTeam   playerSortType = "Team"
	playerSortTime   playerSortType = "Time"
)

var sortDirections = []playerSortType{playerSortName, playerSortKills, playerSortKD, playerSortStatus, playerSortTeam, playerSortTime}

func (ui *Ui) UpdatePlayerState(players model.PlayerCollection) {
	// Sort by name first
	sort.Slice(players, func(i, j int) bool {
		return strings.ToLower(players[i].Name) < strings.ToLower(players[j].Name)
	})
	// Apply secondary ordering
	switch ui.playerSortDir {
	case playerSortKills:
		sort.SliceStable(players, func(i, j int) bool {
			return players[i].Kills > players[j].Kills
		})
	case playerSortStatus:
		sort.SliceStable(players, func(i, j int) bool {
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
	case playerSortTeam:
		sort.SliceStable(players, func(i, j int) bool {
			return players[i].Team < players[j].Team
		})
	case playerSortTime:
		sort.SliceStable(players, func(i, j int) bool {
			return players[i].Connected < players[j].Connected
		})
	case playerSortKD:
		sort.SliceStable(players, func(i, j int) bool {
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
	}
	if errReboot := ui.playerList.Reload(players.AsAny()); errReboot != nil {
		log.Printf("Faile to reboot data: %v\n", errReboot)
	}
}

func (ui *Ui) AddUserMessage(msg model.UserMessage) {
	if errAppend := ui.userMessageList.Append(msg); errAppend != nil {
		log.Printf("Failed to append game message: %v", errAppend)
	}
	ui.userMessageList.Widget().Refresh()

	if userChat, found := ui.chatHistoryWindows[msg.PlayerSID]; found {
		if errAppend := userChat.list.Append(msg); errAppend != nil {
			log.Printf("Failed to append user history message: %v", errAppend)
		}
		userChat.list.Widget().Refresh()
	}
}

type userChatContainer struct {
	fyne.Window
	list *baseListWidget
}

func (ui *Ui) createChatHistoryWindow(sid64 steamid.SID64) error {
	_, found := ui.chatHistoryWindows[sid64]
	if found {
		ui.chatHistoryWindows[sid64].Show()
	} else {
		windowTitle := translations.Tr(&i18n.Message{ID: string(translations.WindowChatHistoryUser)}, 1, map[string]interface{}{
			"SteamId": sid64,
		})
		window := ui.application.NewWindow(windowTitle)
		window.SetOnClosed(func() {
			delete(ui.chatHistoryWindows, sid64)
		})
		messages, errMessage := ui.queryUserMessagesFunc(sid64)
		if errMessage != nil {
			return errors.Wrap(errMessage, "Failed to fetch user message history")
		}
		msgList := ui.createUserHistoryMessageList()
		if errReload := msgList.Reload(messages.AsAny()); errReload != nil {
			return errors.Wrap(errMessage, "Failed to reload user message history")
		}
		window.SetContent(msgList.Widget())
		window.Resize(fyne.NewSize(600, 600))
		window.Show()
		ui.chatHistoryWindows[sid64] = &userChatContainer{Window: window, list: msgList}
	}
	return nil
}

func (ui *Ui) createNameHistoryWindow(sid64 steamid.SID64) error {
	_, found := ui.nameHistoryWindows[sid64]
	if found {
		ui.nameHistoryWindows[sid64].Show()
	} else {
		windowTitle := translations.Tr(&i18n.Message{ID: string(translations.WindowNameHistory)}, 1, map[string]interface{}{
			"SteamId": sid64,
		})
		window := ui.application.NewWindow(windowTitle)
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
	wikiUrl, errUrl := url.Parse(urlHelp)
	if errUrl != nil {
		log.Panicln("Failed to parse wiki url")
	}
	shortCutLaunch := &desktop.CustomShortcut{KeyName: fyne.KeyL, Modifier: fyne.KeyModifierControl}
	shortCutChat := &desktop.CustomShortcut{KeyName: fyne.KeyC, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}
	shortCutFolder := &desktop.CustomShortcut{KeyName: fyne.KeyE, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}
	shortCutSettings := &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl}
	shortCutLists := &desktop.CustomShortcut{KeyName: fyne.KeyL, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}
	shortCutQuit := &desktop.CustomShortcut{KeyName: fyne.KeyQ, Modifier: fyne.KeyModifierControl}
	shortCutHelp := &desktop.CustomShortcut{KeyName: fyne.KeyH, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}
	shortCutAbout := &desktop.CustomShortcut{KeyName: fyne.KeyA, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}

	ui.rootWindow.Canvas().AddShortcut(shortCutLaunch, func(shortcut fyne.Shortcut) {
		ui.launcher()
	})
	ui.rootWindow.Canvas().AddShortcut(shortCutChat, func(shortcut fyne.Shortcut) {
		ui.chatWindow.Show()
	})
	ui.rootWindow.Canvas().AddShortcut(shortCutFolder, func(shortcut fyne.Shortcut) {
		platform.OpenFolder(ui.settings.ConfigRoot())
	})
	ui.rootWindow.Canvas().AddShortcut(shortCutSettings, func(shortcut fyne.Shortcut) {
		ui.settingsDialog.Show()
	})
	ui.rootWindow.Canvas().AddShortcut(shortCutLaunch, func(shortcut fyne.Shortcut) {
		ui.listsDialog.Show()
	})
	ui.rootWindow.Canvas().AddShortcut(shortCutQuit, func(shortcut fyne.Shortcut) {
		ui.application.Quit()
	})
	ui.rootWindow.Canvas().AddShortcut(shortCutHelp, func(shortcut fyne.Shortcut) {
		if errOpenHelp := ui.application.OpenURL(wikiUrl); errOpenHelp != nil {
			log.Printf("Failed to open help url: %v\n", errOpenHelp)
		}
	})
	ui.rootWindow.Canvas().AddShortcut(shortCutAbout, func(shortcut fyne.Shortcut) {
		ui.aboutDialog.Show()
	})
	fm := fyne.NewMenu("Bot Detector",
		&fyne.MenuItem{
			Shortcut: shortCutLaunch,
			Label:    translations.One(translations.LabelLaunch),
			Action:   ui.launcher,
			Icon:     resourceTf2Png,
		},
		&fyne.MenuItem{
			Shortcut: shortCutChat,
			Label:    translations.One(translations.LabelChatLog),
			Action:   ui.chatWindow.Show,
			Icon:     theme.MailComposeIcon(),
		},
		&fyne.MenuItem{
			Shortcut: shortCutFolder,
			Label:    translations.One(translations.LabelConfigFolder),
			Action: func() {
				platform.OpenFolder(ui.settings.ConfigRoot())
			},
			Icon: theme.FolderOpenIcon(),
		},
		&fyne.MenuItem{
			Shortcut: shortCutSettings,
			Label:    translations.One(translations.LabelSettings),
			Action:   ui.settingsDialog.Show,
			Icon:     theme.SettingsIcon(),
		},
		&fyne.MenuItem{
			Shortcut: shortCutLists,
			Label:    translations.One(translations.LabelListConfig),
			Action:   ui.listsDialog.Show,
			Icon:     theme.StorageIcon(),
		},
		fyne.NewMenuItemSeparator(),
		&fyne.MenuItem{
			Icon:     theme.ContentUndoIcon(),
			Shortcut: shortCutQuit,
			Label:    translations.One(translations.LabelQuit),
			IsQuit:   true,
			Action:   ui.application.Quit,
		},
	)

	hm := fyne.NewMenu(translations.One(translations.LabelHelp),
		&fyne.MenuItem{
			Label:    translations.One(translations.LabelHelp),
			Shortcut: shortCutHelp,
			Icon:     theme.HelpIcon(),
			Action: func() {
				if errOpenHelp := ui.application.OpenURL(wikiUrl); errOpenHelp != nil {
					log.Printf("Failed to open help url: %v\n", errOpenHelp)
				}
			}},
		&fyne.MenuItem{
			Label:    translations.One(translations.LabelAbout),
			Shortcut: shortCutAbout,
			Icon:     theme.InfoIcon(),
			Action:   ui.aboutDialog.Show},
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
	log.Printf("Player disconnected: %d", sid64.Int64())
}

func (ui *Ui) Run() {
	ui.rootWindow.Show()
	ui.application.Run()
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
		widget.NewToolbarAction(theme.StorageIcon(), func() {
			ui.listsDialog.Show()
		}),
		widget.NewToolbarAction(theme.FolderOpenIcon(), func() {
			platform.OpenFolder(ui.settings.ConfigRoot())
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.HelpIcon(), func() {
			if errOpenHelp := ui.application.OpenURL(wikiUrl); errOpenHelp != nil {
				log.Printf("Failed to open help url: %v\n", errOpenHelp)
			}
		}),
		widget.NewToolbarAction(theme.InfoIcon(), aboutFunc),
	)
	return toolBar
}

func (ui *Ui) createAboutDialog(parent fyne.Window) dialog.Dialog {
	u, _ := url.Parse(urlHome)
	vbox := container.NewVBox(
		ui.labelVersion,
		ui.labelCommit,
		ui.labelDate,
		ui.labelBuiltBy,
		ui.labelGo,
		widget.NewHyperlink(urlHome, u),
	)
	return dialog.NewCustom(
		translations.One(translations.LabelAbout),
		translations.One(translations.LabelClose),
		vbox,
		parent,
	)
}
