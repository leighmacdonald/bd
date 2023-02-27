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
	"github.com/leighmacdonald/bd/translations"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"log"
	"net/url"
	"sort"
	"strings"
	"time"
)

type PlayerList struct {
	baseListWidget
}

type menuButton struct {
	widget.Button
	menu *fyne.Menu
}

func (m *menuButton) Tapped(event *fyne.PointEvent) {
	widget.ShowPopUpMenuAtPosition(m.menu, fyne.CurrentApp().Driver().CanvasForObject(m), event.AbsolutePosition)
}

func newMenuButton(menu *fyne.Menu) *menuButton {
	c := &menuButton{menu: menu}
	c.ExtendBaseWidget(c)
	c.SetIcon(theme.SettingsIcon())

	return c
}

func generateExternalLinksMenu(steamId steamid.SID64, links []model.LinkConfig, urlOpener func(url *url.URL) error) *fyne.Menu {
	lk := func(link model.LinkConfig, sid64 steamid.SID64, urlOpener func(url *url.URL) error) func() {
		clsLinkValue := link
		clsSteamId := sid64
		return func() {
			u := clsLinkValue.URL
			switch clsLinkValue.IdFormat {
			case model.Steam:
				u = fmt.Sprintf(u, steamid.SID64ToSID(clsSteamId))
			case model.Steam3:
				u = fmt.Sprintf(u, steamid.SID64ToSID3(clsSteamId))
			case model.Steam32:
				u = fmt.Sprintf(u, steamid.SID64ToSID32(clsSteamId))
			case model.Steam64:
				u = fmt.Sprintf(u, clsSteamId.Int64())
			default:
				log.Printf("Got unhandled steamid format, trying steam64: %v", clsLinkValue.IdFormat)
			}
			ul, urlErr := url.Parse(u)
			if urlErr != nil {
				log.Printf("Failed to create link: %v", urlErr)
				return
			}
			if errOpen := urlOpener(ul); errOpen != nil {
				log.Printf("Failed to open url: %v", errOpen)
			}
		}
	}

	var items []*fyne.MenuItem
	sort.Slice(links, func(i, j int) bool {
		return strings.ToLower(links[i].Name) < strings.ToLower(links[j].Name)
	})
	for _, link := range links {
		if !link.Enabled {
			continue
		}
		items = append(items, fyne.NewMenuItem(link.Name, lk(link, steamId, urlOpener)))
	}
	return fyne.NewMenu("Sub Menu", items...)
}

const newItemLabel = "New..."

func (ui *Ui) generateAttributeMenu(sid64 steamid.SID64, knownAttributes []string) *fyne.Menu {
	mkAttr := func(attrName string) func() {
		clsAttribute := attrName
		clsSteamId := sid64
		return func() {
			log.Printf("marking %d as %s", clsSteamId, clsAttribute)
			if errMark := ui.markFn(sid64, []string{clsAttribute}); errMark != nil {
				log.Printf("Failed to mark player: %v\n", errMark)
			}
		}
	}
	attrMenu := fyne.NewMenu(translations.One(translations.LabelMarkAs))
	sort.Slice(knownAttributes, func(i, j int) bool {
		return strings.ToLower(knownAttributes[i]) < strings.ToLower(knownAttributes[j])
	})
	for _, mi := range knownAttributes {
		attrMenu.Items = append(attrMenu.Items, fyne.NewMenuItem(mi, mkAttr(mi)))
	}
	entry := widget.NewEntry()
	entry.Validator = func(s string) error {
		if s == "" {
			return errors.New("Empty attribute name")
		}
		for _, knownAttr := range knownAttributes {
			if strings.EqualFold(knownAttr, s) {
				return errors.New("Duplicate attribute name")
			}
		}
		return nil
	}
	fi := widget.NewFormItem(translations.One(translations.LabelAttributeName), entry)

	attrMenu.Items = append(attrMenu.Items, fyne.NewMenuItem(newItemLabel, func() {
		w := dialog.NewForm(
			translations.One(translations.WindowMarkCustom),
			translations.One(translations.LabelApply),
			translations.One(translations.LabelClose),
			[]*widget.FormItem{fi}, func(success bool) {
				if !success {
					return
				}
				if errMark := ui.markFn(sid64, []string{entry.Text}); errMark != nil {
					log.Printf("Failed to mark player: %v\n", errMark)
				}
			}, ui.rootWindow)
		w.Show()
	}))
	attrMenu.Refresh()
	return attrMenu
}

func (ui *Ui) generateSteamIdMenu(steamId steamid.SID64) *fyne.Menu {
	m := fyne.NewMenu("Copy SteamID",
		fyne.NewMenuItem(fmt.Sprintf("%d", steamId), func() {
			ui.rootWindow.Clipboard().SetContent(fmt.Sprintf("%d", steamId))
		}),
		fyne.NewMenuItem(string(steamid.SID64ToSID(steamId)), func() {
			ui.rootWindow.Clipboard().SetContent(string(steamid.SID64ToSID(steamId)))
		}),
		fyne.NewMenuItem(string(steamid.SID64ToSID3(steamId)), func() {
			ui.rootWindow.Clipboard().SetContent(string(steamid.SID64ToSID3(steamId)))
		}),
		fyne.NewMenuItem(fmt.Sprintf("%d", steamid.SID64ToSID32(steamId)), func() {
			ui.rootWindow.Clipboard().SetContent(fmt.Sprintf("%d", steamid.SID64ToSID32(steamId)))
		}),
	)
	return m
}

func newMenuItem(key translations.Key, fn func()) *fyne.MenuItem {
	return &fyne.MenuItem{
		Label:  translations.One(key),
		Action: fn,
	}
}

func (ui *Ui) generateKickMenu(userId int64) *fyne.Menu {
	fn := func(reason model.KickReason) func() {
		return func() {
			log.Printf("Calling vote: %d %v", userId, reason)
			if errKick := ui.kickFn(userId, reason); errKick != nil {
				log.Printf("Error trying to call kick: %v\n", errKick)
			}
		}
	}
	return fyne.NewMenu(translations.One(translations.MenuCallVote),
		newMenuItem(translations.MenuVoteCheating, fn(model.KickReasonCheating)),
		newMenuItem(translations.MenuVoteIdle, fn(model.KickReasonIdle)),
		newMenuItem(translations.MenuVoteScamming, fn(model.KickReasonScamming)),
		newMenuItem(translations.MenuVoteOther, fn(model.KickReasonOther)),
	)
}

func (ui *Ui) generateUserMenu(steamId steamid.SID64, userId int64) *fyne.Menu {
	menu := fyne.NewMenu("User Actions",
		&fyne.MenuItem{
			Icon:      theme.CheckButtonCheckedIcon(),
			ChildMenu: ui.generateKickMenu(userId),
			Label:     translations.One(translations.MenuCallVote)},
		&fyne.MenuItem{
			Icon:      theme.ZoomFitIcon(),
			ChildMenu: ui.generateAttributeMenu(steamId, ui.knownAttributes),
			Label:     translations.One(translations.MenuMarkAs)},
		&fyne.MenuItem{
			Icon:      theme.SearchIcon(),
			ChildMenu: generateExternalLinksMenu(steamId, ui.settings.GetLinks(), ui.application.OpenURL),
			Label:     translations.One(translations.MenuOpenExternal)},
		&fyne.MenuItem{
			Icon:      theme.ContentCopyIcon(),
			ChildMenu: ui.generateSteamIdMenu(steamId),
			Label:     translations.One(translations.MenuCopySteamId)},
		&fyne.MenuItem{
			Icon: theme.ListIcon(),
			Action: func() {
				if errChat := ui.createChatHistoryWindow(steamId); errChat != nil {
					showUserError("Error trying to load chat: %v", ui.rootWindow)
				}
			},
			Label: translations.One(translations.MenuChatHistory)},
		&fyne.MenuItem{
			Icon: theme.VisibilityIcon(),
			Action: func() {
				if errChat := ui.createNameHistoryWindow(steamId); errChat != nil {
					showUserError("Error trying to load names: %v", ui.rootWindow)
				}
			},
			Label: translations.One(translations.MenuNameHistory)},
	)
	return menu
}

// ┌─────┬───────────────────────────────────────────────────┐
// │  X  │ profile name                          │   Vac..   │
// │─────────────────────────────────────────────────────────┤
// │ K: 10  A: 66                                            │
// └─────────────────────────────────────────────────────────┘
func (ui *Ui) createPlayerList(compact bool) *baseListWidget {
	const (
		symbolOk  = "✓"
		symbolBad = "✗"
	)
	newLower := func() *fyne.Container {
		cont := container.NewHBox()
		cont.Add(widget.NewLabel("x"))
		cont.Add(widget.NewLabel(""))
		cont.Add(widget.NewLabel(""))
		cont.Add(widget.NewLabel(""))
		cont.Add(widget.NewLabel(""))
		cont.Add(widget.NewLabel(""))

		return cont
	}
	pl := newBaseListWidget()
	_ = pl.autoScrollEnabled.Set(false)
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
		upperContainer.Resize(upperContainer.MinSize())

		rootContainer.Add(upperContainer)
		if !compact {
			rootContainer.Add(newLower())
		}

		rootContainer.Refresh()

		return rootContainer
	}
	updateItem := func(i binding.DataItem, o fyne.CanvasObject) {
		value := i.(binding.Untyped)
		obj, _ := value.Get()
		ps := obj.(*model.Player)

		pl.objectMu.Lock()
		rootContainer := o.(*fyne.Container)
		upperContainer := rootContainer.Objects[0].(*fyne.Container)

		if !compact {
			lowerContainer := rootContainer.Objects[1].(*fyne.Container)
			lowerContainer.Objects[0].(*widget.Label).SetText(fmt.Sprintf("K: %d", ps.Kills))
			lowerContainer.Objects[1].(*widget.Label).SetText(fmt.Sprintf("D: %d", ps.Deaths))
			lowerContainer.Objects[2].(*widget.Label).SetText(fmt.Sprintf("TKA: %d", ps.KillsOn))
			lowerContainer.Objects[3].(*widget.Label).SetText(fmt.Sprintf("TKB: %d", ps.DeathsBy))
			lowerContainer.Objects[4].(*widget.Label).SetText(fmt.Sprintf("Ping: %d", ps.Ping))
			lowerContainer.Objects[5].(*widget.Label).SetText(ps.Connected.String())
			lowerContainer.Refresh()
		}
		btn := upperContainer.Objects[1].(*menuButton)
		btn.menu = ui.generateUserMenu(ps.SteamId, ps.UserId)
		btn.menu.Refresh()
		if ps.Avatar != nil {
			btn.Icon = ps.Avatar
		}
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

		profileLabel.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: ps.Name, Style: nameStyle}}
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
			matchLabel.Segments = []widget.RichTextSegment{
				&widget.TextSegment{Text: fmt.Sprintf("Match: %s [%s]", ps.Match.Origin, ps.Match.MatcherType), Style: vacStyle},
			}
		}
		matchLabel.Refresh()
		vacLabel := lc.Objects[1].(*widget.RichText)
		vacLabel.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: vacMsgFull, Style: vacStyle},
		}
		vacLabel.Refresh()
		upperContainer.Refresh()
		rootContainer.Refresh()
		pl.objectMu.Unlock()

	}
	pl.SetupList(createItem, updateItem)
	return pl
}
