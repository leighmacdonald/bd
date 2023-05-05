package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/platform"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/internal/tr"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type playerWindow struct {
	logger      *zap.Logger
	window      fyne.Window
	list        *widget.List
	boundList   binding.ExternalUntypedList
	content     fyne.CanvasObject
	objectMu    sync.RWMutex
	boundListMu sync.RWMutex

	aboutDialog *aboutDialog

	labelHostnameLabel string
	labelMapLabel      string

	labelHostname       *widget.RichText
	labelMap            *widget.RichText
	labelPlayersHeading *widget.Label
	toolbar             *widget.Toolbar

	bindingPlayerCount binding.Int

	playerSortDir binding.String

	containerHeading   *fyne.Container
	containerStatPanel *fyne.Container

	menuCreator MenuCreator
	onReload    func(count int)
	avatarCache *avatarCache
}

func (screen *playerWindow) showSettings() {
	d := newSettingsDialog(screen.logger, screen.window)
	d.Show()
}

func (screen *playerWindow) updatePlayerState(players store.PlayerCollection) {
	// Sort by name first
	sort.Slice(players, func(i, j int) bool {
		return strings.ToLower(players[i].Name) < strings.ToLower(players[j].Name)
	})
	sortType, errGet := screen.playerSortDir.Get()
	if errGet != nil {
		screen.logger.Error("Failed to get sort dir: %v\n", zap.Error(errGet))
		sortType = string(playerSortTeam)
	}
	// Apply secondary ordering
	switch playerSortType(sortType) {
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
	if errReboot := screen.Reload(players); errReboot != nil {
		screen.logger.Error("Failed to reboot player list", zap.Error(errReboot))
	}
}

func (screen *playerWindow) UpdateServerState(state detector.Server) {
	serverName := "n/a"
	if state.ServerName != "" {
		serverName = state.ServerName
	}
	screen.labelHostname.Segments = []widget.RichTextSegment{
		&widget.TextSegment{Text: screen.labelHostnameLabel, Style: widget.RichTextStyleInline},
		&widget.TextSegment{Text: serverName, Style: widget.RichTextStyleStrong},
	}
	screen.labelHostname.Refresh()
	currentMap := "n/a"
	if state.CurrentMap != "" {
		currentMap = state.CurrentMap
	}
	screen.labelMap.Segments = []widget.RichTextSegment{
		&widget.TextSegment{Text: screen.labelMapLabel, Style: widget.RichTextStyleInline},
		&widget.TextSegment{Text: currentMap, Style: widget.RichTextStyleStrong},
	}
	screen.labelMap.Refresh()
}

func (screen *playerWindow) Reload(rr store.PlayerCollection) error {
	bl := make([]interface{}, len(rr))
	for i, r := range rr {
		bl[i] = r
	}
	screen.boundListMu.Lock()
	defer screen.boundListMu.Unlock()
	if errSet := screen.boundList.Set(bl); errSet != nil {
		screen.logger.Error("failed to set player list", zap.Error(errSet))
	}
	if errReload := screen.boundList.Reload(); errReload != nil {
		return errReload
	}

	screen.list.Refresh()
	screen.onReload(len(bl))
	return nil
}

func (screen *playerWindow) createMainMenu() {
	wikiUrl, errUrl := url.Parse(urlHelp)
	if errUrl != nil {
		screen.logger.Panic("Failed to parse wiki url")
	}
	shortCutLaunch := &desktop.CustomShortcut{KeyName: fyne.KeyL, Modifier: fyne.KeyModifierControl}
	shortCutChat := &desktop.CustomShortcut{KeyName: fyne.KeyC, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}
	shortCutFolder := &desktop.CustomShortcut{KeyName: fyne.KeyE, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}
	shortCutSettings := &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl}
	shortCutQuit := &desktop.CustomShortcut{KeyName: fyne.KeyQ, Modifier: fyne.KeyModifierControl}
	shortCutHelp := &desktop.CustomShortcut{KeyName: fyne.KeyH, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}
	shortCutAbout := &desktop.CustomShortcut{KeyName: fyne.KeyA, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}

	screen.window.Canvas().AddShortcut(shortCutLaunch, func(shortcut fyne.Shortcut) {
		go detector.LaunchGameAndWait()
	})

	screen.window.Canvas().AddShortcut(shortCutChat, func(shortcut fyne.Shortcut) {
		windows.chat.Show()
	})

	screen.window.Canvas().AddShortcut(shortCutFolder, func(shortcut fyne.Shortcut) {
		showUserError(platform.OpenFolder(detector.Settings().ConfigRoot()), screen.window)
	})

	screen.window.Canvas().AddShortcut(shortCutSettings, func(shortcut fyne.Shortcut) {
		screen.showSettings()
	})

	screen.window.Canvas().AddShortcut(shortCutQuit, func(shortcut fyne.Shortcut) {
		application.Quit()
	})

	screen.window.Canvas().AddShortcut(shortCutHelp, func(shortcut fyne.Shortcut) {
		if errOpenHelp := application.OpenURL(wikiUrl); errOpenHelp != nil {
			screen.logger.Error("Failed to open help url", zap.Error(errOpenHelp))
		}
	})

	screen.window.Canvas().AddShortcut(shortCutAbout, func(shortcut fyne.Shortcut) {
		screen.aboutDialog.Show()
	})

	labelMainMenu := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_menu_heading", Other: "Bot Detector"}})
	labelLaunch := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_menu_launch", Other: "Launch TF2"}})
	labelChatLog := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_menu_chat_log", Other: "Chat Log"}})
	labelConfigFolder := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_menu_config_folder", Other: "Open Config Folder"}})
	labelSettings := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_menu_settings", Other: "UserSettings"}})
	labelQuit := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_menu_quit", Other: "Quit"}})
	fm := fyne.NewMenu(labelMainMenu,
		&fyne.MenuItem{
			Shortcut: shortCutLaunch,
			Label:    labelLaunch,
			Action: func() {
				go detector.LaunchGameAndWait()
			},
			Icon: resourceTf2Png,
		},
		&fyne.MenuItem{
			Shortcut: shortCutChat,
			Label:    labelChatLog,
			Action:   windows.chat.Show,
			Icon:     theme.MailComposeIcon(),
		},
		&fyne.MenuItem{
			Shortcut: shortCutFolder,
			Label:    labelConfigFolder,
			Action: func() {
				showUserError(platform.OpenFolder(detector.Settings().ConfigRoot()), screen.window)
			},
			Icon: theme.FolderOpenIcon(),
		},
		&fyne.MenuItem{
			Shortcut: shortCutSettings,
			Label:    labelSettings,
			Action: func() {
				screen.showSettings()
			},
			Icon: theme.SettingsIcon(),
		},
		fyne.NewMenuItemSeparator(),
		&fyne.MenuItem{
			Icon:     theme.ContentUndoIcon(),
			Shortcut: shortCutQuit,
			Label:    labelQuit,
			IsQuit:   true,
			Action:   application.Quit,
		},
	)

	labelHelpMenuHeading := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "help_menu_heading", Other: "Help"}})
	labelHelpMenu := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "help_menu_help", Other: "Help"}})
	labelAboutMenu := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "help_menu_about", Other: "About"}})
	hm := fyne.NewMenu(labelHelpMenuHeading,
		&fyne.MenuItem{
			Label:    labelHelpMenu,
			Shortcut: shortCutHelp,
			Icon:     theme.HelpIcon(),
			Action: func() {
				if errOpenHelp := application.OpenURL(wikiUrl); errOpenHelp != nil {
					screen.logger.Error("Failed to open help url", zap.Error(errOpenHelp), zap.String("url", wikiUrl.String()))
				}
			}},
		&fyne.MenuItem{
			Label:    labelAboutMenu,
			Shortcut: shortCutAbout,
			Icon:     theme.InfoIcon(),
			Action:   screen.aboutDialog.Show},
	)
	screen.window.SetMainMenu(fyne.NewMainMenu(fm, hm))
}

const symbolBad = "x"

// ┌─────┬───────────────────────────────────────────────────┐
// │  P  │ profile name                          │   Vac..   │
// │─────────────────────────────────────────────────────────┤
func newPlayerWindow(logger *zap.Logger, menuCreator MenuCreator, version detector.Version) *playerWindow {
	hostname := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_label_hostname", Other: "Hostname: "}})
	mapName := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_label_map", Other: "Map: "}})
	screen := &playerWindow{
		logger:             logger,
		window:             application.NewWindow("Bot Detector"),
		boundList:          binding.BindUntypedList(&[]interface{}{}),
		bindingPlayerCount: binding.NewInt(),
		labelHostnameLabel: hostname,
		labelMapLabel:      mapName,
		menuCreator:        menuCreator,
		labelHostname: widget.NewRichText(
			&widget.TextSegment{Text: hostname, Style: widget.RichTextStyleInline},
			&widget.TextSegment{Text: "n/a", Style: widget.RichTextStyleStrong},
		),
		labelMap: widget.NewRichText(
			&widget.TextSegment{Text: mapName, Style: widget.RichTextStyleInline},
			&widget.TextSegment{Text: "n/a", Style: widget.RichTextStyleStrong},
		),
		playerSortDir: binding.BindPreferenceString("sort_dir", application.Preferences()),
	}
	if sortDir, getErr := screen.playerSortDir.Get(); getErr != nil && sortDir == "" {
		if errSetSort := screen.playerSortDir.Set(string(playerSortTeam)); errSetSort != nil {
			screen.logger.Error("Failed to set initial sort dir", zap.Error(errSetSort))
		}
	}
	screen.labelPlayersHeading = widget.NewLabelWithData(binding.IntToStringWithFormat(screen.bindingPlayerCount, "%d Players"))
	screen.aboutDialog = newAboutDialog(screen.window, version)
	screen.onReload = func(count int) {
		if errSet := screen.bindingPlayerCount.Set(count); errSet != nil {
			screen.logger.Error("Failed to update player count", zap.Error(errSet))
		}
	}
	screen.toolbar = newToolbar(
		application,
		screen.window,
		func() {
			windows.chat.Show()
		}, func() {
			screen.showSettings()
		}, func() {
			screen.aboutDialog.Show()
		},
		func() {
			go detector.LaunchGameAndWait()
		},
		func() {
			windows.search.Show()
		})

	var dirNames []string
	for _, dir := range sortDirections {
		dirNames = append(dirNames, string(dir))
	}
	sortSelect := widget.NewSelect(dirNames, func(s string) {
		showUserError(screen.playerSortDir.Set(s), screen.window)
		v, _ := screen.boundList.Get()
		var sorted store.PlayerCollection
		for _, p := range v {
			sorted = append(sorted, p.(*store.Player))
		}
		screen.updatePlayerState(sorted)
	})

	sortSelect.PlaceHolder = tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_label_sort_by", Other: "Sort By..."}})

	screen.createMainMenu()

	createItem := func() fyne.CanvasObject {
		rootContainer := container.NewVBox()

		menuBtn := newMenuButton(fyne.NewMenu(""))
		menuBtn.Icon = resourceDefaultavatarJpg
		menuBtn.IconPlacement = widget.ButtonIconTrailingText
		menuBtn.Refresh()

		upperContainer := container.NewBorder(
			nil,
			nil,
			menuBtn,
			container.NewHBox(widget.NewRichText()),
			widget.NewRichText(),
		)
		rootContainer.Add(upperContainer)
		rootContainer.Refresh()

		return rootContainer
	}
	updateItem := func(i binding.DataItem, o fyne.CanvasObject) {
		screen.objectMu.Lock()
		value := i.(binding.Untyped)
		obj, _ := value.Get()
		ps := obj.(*store.Player)
		//ps.RLock()

		rootContainer := o.(*fyne.Container)
		upperContainer := rootContainer.Objects[0].(*fyne.Container)

		btn := upperContainer.Objects[1].(*menuButton)
		btn.menu = screen.menuCreator(screen.window, ps.SteamId, ps.UserId)
		btn.Icon = screen.avatarCache.GetAvatar(ps.SteamId)
		btn.Refresh()

		profileLabel := upperContainer.Objects[0].(*widget.RichText)

		styleKD := calcKDStyle(ps.Kills, ps.Deaths)
		styleKDAllTIme := calcKDStyle(ps.KillsOn, ps.DeathsBy)

		profileLabel.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: ps.Name, Style: calcNameStyle(ps, detector.Settings().GetSteamId())},
			&widget.TextSegment{Text: fmt.Sprintf("  %d", ps.Kills), Style: styleKD},
			&widget.TextSegment{Text: ":", Style: styleKD},
			&widget.TextSegment{Text: fmt.Sprintf("%d", ps.Deaths), Style: styleKD},
			&widget.TextSegment{Text: fmt.Sprintf("  %d", ps.KillsOn), Style: styleKDAllTIme},
			&widget.TextSegment{Text: ":", Style: styleKDAllTIme},
			&widget.TextSegment{Text: fmt.Sprintf("%d", ps.DeathsBy), Style: styleKDAllTIme},
			&widget.TextSegment{Text: fmt.Sprintf("  %dms", ps.Ping), Style: calcPingStyle(ps.Ping)},
		}
		profileLabel.Refresh()

		lc := upperContainer.Objects[2].(*fyne.Container)
		rightLabels := lc.Objects[0].(*widget.RichText)
		rightLabels.Segments = []widget.RichTextSegment{}
		for _, s := range generateRightSegments(ps) {
			rightLabels.Segments = append(rightLabels.Segments, s)
		}
		rightLabels.Refresh()
		lc.Refresh()
		rootContainer.Refresh()
		//ps.RUnlock()
		screen.objectMu.Unlock()
	}
	screen.containerHeading = container.NewBorder(
		nil,
		nil,
		screen.toolbar,
		container.NewHBox(sortSelect),
		container.NewCenter(screen.labelPlayersHeading),
	)
	screen.containerStatPanel = container.NewHBox(
		screen.labelMap,
		screen.labelHostname,
	)
	screen.createMainMenu()
	screen.window.Resize(fyne.NewSize(sizeWindowMainWidth, sizeWindowMainHeight))
	screen.window.SetCloseIntercept(func() {
		application.Quit()
	})
	screen.list = widget.NewListWithData(screen.boundList, createItem, updateItem)
	screen.content = container.NewVScroll(screen.list)
	screen.window.SetContent(container.NewBorder(
		screen.containerHeading,
		screen.containerStatPanel,
		nil,
		nil,
		screen.content),
	)
	return screen
}

func generateRightSegments(ps *store.Player) []*widget.TextSegment {
	var rightSegments []*widget.TextSegment
	banStateMsg, banStateStyle := generateBanStateMsg(ps)
	if ps.Notes != "" {
		notesStyle := widget.RichTextStyleStrong
		notesStyle.ColorName = theme.ColorNameWarning
		rightSegments = append(rightSegments, &widget.TextSegment{Text: "[note]  ", Style: notesStyle})
	}
	if ps.IsMatched() {
		suffix := ""
		if ps.Whitelisted {
			suffix = " (WL)"
		}
		for _, match := range ps.Matches {
			rightSegments = append(rightSegments,
				&widget.TextSegment{Text: fmt.Sprintf("%s [%s] [%s]%s", match.Origin, match.MatcherType, strings.Join(match.Attributes, ","), suffix), Style: banStateStyle})
		}
	}
	if banStateMsg != "" {
		rightSegments = append(rightSegments, &widget.TextSegment{Text: banStateMsg, Style: banStateStyle})
	}

	return rightSegments
}

func generateBanStateMsg(ps *store.Player) (string, widget.RichTextStyle) {
	style := widget.RichTextStyleStrong
	style.ColorName = theme.ColorNameSuccess

	var vacState []string
	if ps.NumberOfVACBans > 0 {
		vacState = append(vacState, fmt.Sprintf("VB: %s", strings.Repeat(symbolBad, ps.NumberOfVACBans)))
	}
	if ps.NumberOfGameBans > 0 {
		vacState = append(vacState, fmt.Sprintf("GB: %s", strings.Repeat(symbolBad, ps.NumberOfGameBans)))
	}
	if ps.CommunityBanned {
		vacState = append(vacState, fmt.Sprintf("CB: %s", symbolBad))
	}
	if ps.EconomyBan {
		vacState = append(vacState, fmt.Sprintf("EB: %s", symbolBad))
	}

	if len(vacState) > 0 || ps.IsMatched() && !ps.Whitelisted {
		style.ColorName = theme.ColorNameError
	}
	vacMsg := strings.Join(vacState, ", ")
	vacMsgFull := vacMsg
	if ps.LastVACBanOn != nil {
		vacMsgFull = fmt.Sprintf("[%s] (%s - %d days)",
			vacMsg,
			ps.LastVACBanOn.Format("Mon Jan 02 2006"),
			int(time.Since(*ps.LastVACBanOn).Hours()/24),
		)
	}
	return vacMsgFull, style
}

func calcNameStyle(player *store.Player, ownSid steamid.SID64) widget.RichTextStyle {
	style := widget.RichTextStyleStrong
	style.ColorName = theme.ColorNameSuccess
	if player.GetSteamID() == ownSid {
		style.ColorName = theme.ColorNameSuccess
	} else if player.IsDisconnected() {
		style.ColorName = theme.ColorNameForeground
	} else if player.NumberOfVACBans > 0 {
		style.ColorName = theme.ColorNameWarning
	} else if player.NumberOfGameBans > 0 || player.CommunityBanned || player.EconomyBan {
		style.ColorName = theme.ColorNameWarning
	} else if player.Team == store.Red {
		style.ColorName = theme.ColorNameError
	} else {
		style.ColorName = theme.ColorNamePrimary
	}
	return style
}

func calcPingStyle(ping int) widget.RichTextStyle {
	style := widget.RichTextStyleInline
	style.Inline = true
	if ping > 150 {
		style.ColorName = theme.ColorNameError
	} else if ping > 100 {
		style.ColorName = theme.ColorNameWarning
	} else {
		style.ColorName = theme.ColorNameSuccess
	}
	return style
}

func calcKDStyle(kills int, deaths int) widget.RichTextStyle {
	style := widget.RichTextStyleInline
	style.Inline = true
	if kills > deaths {
		style.ColorName = theme.ColorNameSuccess
	} else if deaths > kills {
		style.ColorName = theme.ColorNameError
	}
	return style
}

func newToolbar(app fyne.App, parent fyne.Window, chatFunc func(), settingsFunc func(), aboutFunc func(), launchFunc func(), showSearchFunc func()) *widget.Toolbar {
	wikiUrl, _ := url.Parse(urlHelp)
	toolBar := widget.NewToolbar(
		widget.NewToolbarAction(resourceTf2Png, func() {
			sid := detector.Settings().GetSteamId()
			if !sid.Valid() {
				msg := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "error_steam_id_misconfigured", Other: "Invalid steamid configuration"}})
				showUserError(errors.New(msg), parent)
			} else {
				launchFunc()
			}
		}),
		widget.NewToolbarAction(theme.MailComposeIcon(), chatFunc),
		widget.NewToolbarAction(theme.SearchIcon(), showSearchFunc),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.SettingsIcon(), settingsFunc),
		widget.NewToolbarAction(theme.FolderOpenIcon(), func() {
			showUserError(platform.OpenFolder(detector.Settings().ConfigRoot()), parent)
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.HelpIcon(), func() {
			if errOpenHelp := app.OpenURL(wikiUrl); errOpenHelp != nil {
				logger.Error("Failed to open help url: %v\n", zap.Error(errOpenHelp))
			}
		}),
		widget.NewToolbarAction(theme.InfoIcon(), aboutFunc),
	)
	return toolBar
}
