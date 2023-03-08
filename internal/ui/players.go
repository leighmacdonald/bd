package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/internal/platform"
	"github.com/leighmacdonald/bd/internal/tr"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"log"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type playerWindow struct {
	app         fyne.App
	window      fyne.Window
	list        *widget.List
	boundList   binding.ExternalUntypedList
	content     fyne.CanvasObject
	objectMu    sync.RWMutex
	boundListMu sync.RWMutex
	settings    *model.Settings

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

	onShowChat   func()
	onShowSearch func()

	menuCreator MenuCreator
	onReload    func(count int)
	callBacks   callBacks
	avatarCache *avatarCache
}

func (screen *playerWindow) showSettings(settings *model.Settings) {
	d := newSettingsDialog(screen.window, settings)
	d.Show()
}

func (screen *playerWindow) updatePlayerState(players model.PlayerCollection) {
	// Sort by name first
	sort.Slice(players, func(i, j int) bool {
		return strings.ToLower(players[i].Name) < strings.ToLower(players[j].Name)
	})
	sortType, errGet := screen.playerSortDir.Get()
	if errGet != nil {
		log.Printf("Failed to get sort dir: %v\n", errGet)
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
		log.Printf("Faile to reboot data: %v\n", errReboot)
	}
}

func (screen *playerWindow) UpdateServerState(state model.Server) {
	screen.labelHostname.Segments = []widget.RichTextSegment{
		&widget.TextSegment{Text: screen.labelHostnameLabel, Style: widget.RichTextStyleInline},
		&widget.TextSegment{Text: state.ServerName, Style: widget.RichTextStyleStrong},
	}
	screen.labelHostname.Refresh()
	screen.labelMap.Segments = []widget.RichTextSegment{
		&widget.TextSegment{Text: screen.labelMapLabel, Style: widget.RichTextStyleInline},
		&widget.TextSegment{Text: state.CurrentMap, Style: widget.RichTextStyleStrong},
	}
	screen.labelMap.Refresh()
}

func (screen *playerWindow) Reload(rr model.PlayerCollection) error {
	bl := make([]interface{}, len(rr))
	for i, r := range rr {
		bl[i] = r
	}
	screen.boundListMu.Lock()
	defer screen.boundListMu.Unlock()
	if errSet := screen.boundList.Set(bl); errSet != nil {
		log.Printf("failed to set player list: %v\n", errSet)
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
		log.Panicln("Failed to parse wiki url")
	}
	shortCutLaunch := &desktop.CustomShortcut{KeyName: fyne.KeyL, Modifier: fyne.KeyModifierControl}
	shortCutChat := &desktop.CustomShortcut{KeyName: fyne.KeyC, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}
	shortCutFolder := &desktop.CustomShortcut{KeyName: fyne.KeyE, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}
	shortCutSettings := &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl}
	shortCutQuit := &desktop.CustomShortcut{KeyName: fyne.KeyQ, Modifier: fyne.KeyModifierControl}
	shortCutHelp := &desktop.CustomShortcut{KeyName: fyne.KeyH, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}
	shortCutAbout := &desktop.CustomShortcut{KeyName: fyne.KeyA, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}

	screen.window.Canvas().AddShortcut(shortCutLaunch, func(shortcut fyne.Shortcut) {
		screen.callBacks.gameLauncherFunc()
	})
	screen.window.Canvas().AddShortcut(shortCutChat, func(shortcut fyne.Shortcut) {
		screen.onShowChat()
	})
	screen.window.Canvas().AddShortcut(shortCutFolder, func(shortcut fyne.Shortcut) {
		platform.OpenFolder(screen.settings.ConfigRoot())
	})
	screen.window.Canvas().AddShortcut(shortCutSettings, func(shortcut fyne.Shortcut) {
		screen.showSettings(screen.settings)
	})
	screen.window.Canvas().AddShortcut(shortCutQuit, func(shortcut fyne.Shortcut) {
		screen.app.Quit()
	})
	screen.window.Canvas().AddShortcut(shortCutHelp, func(shortcut fyne.Shortcut) {
		if errOpenHelp := screen.app.OpenURL(wikiUrl); errOpenHelp != nil {
			log.Printf("Failed to open help url: %v\n", errOpenHelp)
		}
	})
	screen.window.Canvas().AddShortcut(shortCutAbout, func(shortcut fyne.Shortcut) {
		screen.aboutDialog.Show()
	})

	labelMainMenu := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_menu_heading", Other: "Bot Detector"}})
	labelLaunch := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_menu_launch", Other: "Launch TF2"}})
	labelChatLog := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_menu_chat_log", Other: "Chat Log"}})
	labelConfigFolder := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_menu_config_folder", Other: "Open Config Folder"}})
	labelSettings := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_menu_settings", Other: "Settings"}})
	labelQuit := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_menu_quit", Other: "Quit"}})
	fm := fyne.NewMenu(labelMainMenu,
		&fyne.MenuItem{
			Shortcut: shortCutLaunch,
			Label:    labelLaunch,
			Action: func() {
				go screen.callBacks.gameLauncherFunc()
			},
			Icon: resourceTf2Png,
		},
		&fyne.MenuItem{
			Shortcut: shortCutChat,
			Label:    labelChatLog,
			Action:   screen.onShowChat,
			Icon:     theme.MailComposeIcon(),
		},
		&fyne.MenuItem{
			Shortcut: shortCutFolder,
			Label:    labelConfigFolder,
			Action: func() {
				platform.OpenFolder(screen.settings.ConfigRoot())
			},
			Icon: theme.FolderOpenIcon(),
		},
		&fyne.MenuItem{
			Shortcut: shortCutSettings,
			Label:    labelSettings,
			Action: func() {
				screen.showSettings(screen.settings)
			},
			Icon: theme.SettingsIcon(),
		},
		fyne.NewMenuItemSeparator(),
		&fyne.MenuItem{
			Icon:     theme.ContentUndoIcon(),
			Shortcut: shortCutQuit,
			Label:    labelQuit,
			IsQuit:   true,
			Action:   screen.app.Quit,
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
				if errOpenHelp := screen.app.OpenURL(wikiUrl); errOpenHelp != nil {
					log.Printf("Failed to open help url: %v\n", errOpenHelp)
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

const symbolOk = "✓"
const symbolBad = "✗"

// ┌─────┬───────────────────────────────────────────────────┐
// │  P  │ profile name                          │   Vac..   │
// │─────────────────────────────────────────────────────────┤
func newPlayerWindow(app fyne.App, settings *model.Settings, showChatWindowFunc func(), showSearchWindowFunc func(),
	callbacks callBacks, menuCreator MenuCreator, cache *avatarCache, version model.Version) *playerWindow {

	hostname := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_label_hostname", Other: "Hostname: "}})
	mapName := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "main_label_map", Other: "Map: "}})

	screen := &playerWindow{
		app:                app,
		window:             app.NewWindow("Bot Detector"),
		boundList:          binding.BindUntypedList(&[]interface{}{}),
		bindingPlayerCount: binding.NewInt(),
		onShowChat:         showChatWindowFunc,
		onShowSearch:       showSearchWindowFunc,
		callBacks:          callbacks,
		labelHostnameLabel: hostname,
		labelMapLabel:      mapName,
		menuCreator:        menuCreator,
		avatarCache:        cache,
		settings:           settings,
		labelHostname: widget.NewRichText(
			&widget.TextSegment{Text: hostname, Style: widget.RichTextStyleInline},
			&widget.TextSegment{Text: "n/a", Style: widget.RichTextStyleStrong},
		),
		labelMap: widget.NewRichText(
			&widget.TextSegment{Text: mapName, Style: widget.RichTextStyleInline},
			&widget.TextSegment{Text: "n/a", Style: widget.RichTextStyleStrong},
		),
		playerSortDir: binding.BindPreferenceString("sort_dir", app.Preferences()),
	}
	if sortDir, getErr := screen.playerSortDir.Get(); getErr != nil && sortDir == "" {
		if errSetSort := screen.playerSortDir.Set(string(playerSortTeam)); errSetSort != nil {
			log.Printf("Failed to set initial sort dir: %s\n", errSetSort)
		}
	}
	screen.labelPlayersHeading = widget.NewLabelWithData(binding.IntToStringWithFormat(screen.bindingPlayerCount, "%d Players"))
	screen.aboutDialog = newAboutDialog(screen.window, version)
	screen.onReload = func(count int) {
		if errSet := screen.bindingPlayerCount.Set(count); errSet != nil {
			log.Printf("Failed to update player count: %v\n", errSet)
		}
	}
	screen.toolbar = newToolbar(
		app,
		screen.window,
		settings,
		func() {
			screen.onShowChat()
		}, func() {
			screen.showSettings(settings)
		}, func() {
			screen.aboutDialog.Show()
		},
		func() {
			go screen.callBacks.gameLauncherFunc()
		},
		func() {
			screen.onShowSearch()
		})

	var dirNames []string
	for _, dir := range sortDirections {
		dirNames = append(dirNames, string(dir))
	}
	sortSelect := widget.NewSelect(dirNames, func(s string) {
		if errSet := screen.playerSortDir.Set(s); errSet != nil {
			log.Printf("Failed to set sort dir: %v\n", errSet)
		}
		v, _ := screen.boundList.Get()
		var sorted model.PlayerCollection
		for _, p := range v {
			sorted = append(sorted, p.(*model.Player))
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
			container.NewHBox(widget.NewRichText(), widget.NewRichText()),
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
		ps := obj.(*model.Player)
		ps.RLock()

		rootContainer := o.(*fyne.Container)
		upperContainer := rootContainer.Objects[0].(*fyne.Container)

		btn := upperContainer.Objects[1].(*menuButton)
		btn.menu = screen.menuCreator(screen.window, ps.SteamId, ps.UserId)
		btn.Icon = screen.avatarCache.GetAvatar(ps.SteamId)
		btn.Refresh()

		profileLabel := upperContainer.Objects[0].(*widget.RichText)
		stlBad := widget.RichTextStyleStrong
		stlBad.ColorName = theme.ColorNameError

		stlOk := widget.RichTextStyleStrong
		stlOk.ColorName = theme.ColorNameSuccess

		nameStyle := stlOk
		if ps.NumberOfVACBans > 0 {
			nameStyle.ColorName = theme.ColorNameWarning
		} else if ps.NumberOfGameBans > 0 || ps.CommunityBanned || ps.EconomyBan {
			nameStyle.ColorName = theme.ColorNameWarning
		} else if ps.Team == model.Red {
			nameStyle.ColorName = theme.ColorNameError
		} else {
			nameStyle.ColorName = theme.ColorNamePrimary
		}
		stlKD := widget.RichTextStyleInline
		if ps.Kills > ps.Deaths {
			stlKD.ColorName = theme.ColorNameSuccess
		} else if ps.Deaths > ps.Kills {
			stlKD.ColorName = theme.ColorNameError
		}
		stlPing := widget.RichTextStyleInline
		if ps.Ping > 150 {
			stlPing.ColorName = theme.ColorNameError
		} else if ps.Ping > 100 {
			stlPing.ColorName = theme.ColorNameWarning
		} else {
			stlPing.ColorName = theme.ColorNameSuccess
		}
		profileLabel.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: ps.Name, Style: nameStyle},
			&widget.TextSegment{Text: fmt.Sprintf(" %d", ps.Kills), Style: stlKD},
			&widget.TextSegment{Text: ":", Style: widget.RichTextStyleInline},
			&widget.TextSegment{Text: fmt.Sprintf("%d", ps.Deaths), Style: stlKD},
			&widget.TextSegment{Text: fmt.Sprintf(" %dms", ps.Ping), Style: stlPing},
		}
		profileLabel.Refresh()
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
		vacStyle := stlBad
		if len(vacState) == 0 && !ps.IsMatched() {
			vacState = append(vacState, symbolOk)
			vacStyle = stlOk
		}
		vacMsg := strings.Join(vacState, ", ")
		vacMsgFull := ""
		if ps.LastVACBanOn != nil {
			vacMsgFull = fmt.Sprintf("[%s] (%s - %d days)",
				vacMsg,
				ps.LastVACBanOn.Format("Mon Jan 02 2006"),
				int(time.Since(*ps.LastVACBanOn).Hours()/24),
			)
		}
		lc := upperContainer.Objects[2].(*fyne.Container)
		matchLabel := lc.Objects[0].(*widget.RichText)
		if ps.IsMatched() {
			if ps.Whitelisted {
				matchLabel.Segments = []widget.RichTextSegment{
					&widget.TextSegment{Text: fmt.Sprintf("%s [%s] [%s] (WL)", ps.Match.Origin, ps.Match.MatcherType, strings.Join(ps.Match.Attributes, ",")),
						Style: stlOk},
				}
			} else {
				matchLabel.Segments = []widget.RichTextSegment{
					&widget.TextSegment{Text: fmt.Sprintf("%s [%s] [%s]", ps.Match.Origin, ps.Match.MatcherType, strings.Join(ps.Match.Attributes, ",")),
						Style: vacStyle},
				}
			}
		} else {
			matchLabel.Segments = nil
		}
		matchLabel.Refresh()
		vacLabel := lc.Objects[1].(*widget.RichText)
		vacLabel.Segments = []widget.RichTextSegment{}
		if vacMsg != "" {
			vacLabel.Segments = append(vacLabel.Segments, &widget.TextSegment{Text: vacMsgFull, Style: vacStyle})
		}
		if ps.Notes != "" {
			notesStyle := stlOk
			notesStyle.ColorName = theme.ColorNameWarning
			vacLabel.Segments = append(vacLabel.Segments, &widget.TextSegment{Text: "[note]", Style: notesStyle})
		}
		lc.Refresh()
		vacLabel.Refresh()
		rootContainer.Refresh()
		ps.RUnlock()
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
		screen.app.Quit()
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

func newToolbar(app fyne.App, parent fyne.Window, settings *model.Settings, chatFunc func(), settingsFunc func(), aboutFunc func(), launchFunc func(), showSearchFunc func()) *widget.Toolbar {
	wikiUrl, _ := url.Parse(urlHelp)
	toolBar := widget.NewToolbar(
		widget.NewToolbarAction(resourceTf2Png, func() {
			sid := settings.GetSteamId()
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
			platform.OpenFolder(settings.ConfigRoot())
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.HelpIcon(), func() {
			if errOpenHelp := app.OpenURL(wikiUrl); errOpenHelp != nil {
				log.Printf("Failed to open help url: %v\n", errOpenHelp)
			}
		}),
		widget.NewToolbarAction(theme.InfoIcon(), aboutFunc),
	)
	return toolBar
}
