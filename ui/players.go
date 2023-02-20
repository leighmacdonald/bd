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
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"log"
	"net/url"
	"sort"
	"strings"
	"sync"
)

type PlayerList struct {
	list        *widget.List
	boundList   binding.ExternalUntypedList
	content     fyne.CanvasObject
	objectMu    sync.RWMutex
	boundListMu sync.RWMutex
}

func (playerList *PlayerList) Reload(rr []model.PlayerState) error {
	bl := make([]interface{}, len(rr))
	for i, r := range rr {
		bl[i] = r
	}
	playerList.boundListMu.Lock()
	defer playerList.boundListMu.Unlock()
	if errSet := playerList.boundList.Set(bl); errSet != nil {
		log.Printf("failed to set player list: %v\n", errSet)
	}
	if errReload := playerList.boundList.Reload(); errReload != nil {
		return errReload
	}
	playerList.list.Refresh()
	return nil
}

// Widget returns the actual select list widget.
func (playerList *PlayerList) Widget() *widget.List {
	return playerList.list
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
	attrMenu := fyne.NewMenu("Mark As")
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
	fi := widget.NewFormItem("Attribute Name", entry)

	attrMenu.Items = append(attrMenu.Items, fyne.NewMenuItem(newItemLabel, func() {
		w := dialog.NewForm("Mark with custom attribute", "Confirm", "Dismiss", []*widget.FormItem{fi}, func(b bool) {
			if b {
				log.Printf("item: %v\n", b)
				if errMark := ui.markFn(sid64, []string{entry.Text}); errMark != nil {
					log.Printf("Failed to mark player: %v\n", errMark)
				}
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

func (ui *Ui) generateKickMenu(userId int64) *fyne.Menu {
	m := fyne.NewMenu("Call Vote",
		fyne.NewMenuItem("Kick", func() {
			log.Printf("Kicking user_id %d (cheating)\n", userId)
			if errKick := ui.kickFn(userId); errKick != nil {
				log.Printf("Error trying to call kick: %v\n", errKick)
			}
		}),
	)
	return m
}

func (ui *Ui) generateUserMenu(steamId steamid.SID64, userId int64) *fyne.Menu {
	menu := fyne.NewMenu("User Actions",
		&fyne.MenuItem{
			Icon:      theme.CheckButtonCheckedIcon(),
			ChildMenu: ui.generateKickMenu(userId),
			Label:     "Call Vote..."},
		&fyne.MenuItem{
			Icon:      theme.ZoomFitIcon(),
			ChildMenu: ui.generateAttributeMenu(steamId, ui.knownAttributes),
			Label:     "Mark As..."},
		&fyne.MenuItem{
			Icon:      theme.SearchIcon(),
			ChildMenu: generateExternalLinksMenu(steamId, ui.baseSettings.GetLinks(), ui.application.OpenURL),
			Label:     "Open External..."},
		&fyne.MenuItem{
			Icon:      theme.ContentCopyIcon(),
			ChildMenu: ui.generateSteamIdMenu(steamId),
			Label:     "Copy SteamID..."},
	)
	return menu
}

// ┌─────┬───────────────────────────────────────────────────┐
// │  X  │ profile name                          │   Vac..   │
// │─────────────────────────────────────────────────────────┤
// │ K: 10  A: 66                                            │
// └─────────────────────────────────────────────────────────┘
func (ui *Ui) createPlayerList() *PlayerList {
	//iconSize := fyne.NewSize(64, 64)
	pl := &PlayerList{}
	boundList := binding.BindUntypedList(&[]interface{}{})
	playerListWidget := widget.NewListWithData(
		boundList,
		func() fyne.CanvasObject {
			rootContainer := container.NewVBox()
			lowerContainer := container.NewHBox()

			menuBtn := newMenuButton(fyne.NewMenu(""))
			menuBtn.Icon = resourceUiResourcesDefaultavatarJpg
			menuBtn.IconPlacement = widget.ButtonIconTrailingText
			menuBtn.Refresh()

			upperContainer := container.NewBorder(
				nil,
				nil,
				menuBtn,
				widget.NewRichText(),
				widget.NewRichText(),
			)
			upperContainer.Resize(upperContainer.MinSize())
			lowerContainer.Add(widget.NewLabel(""))
			lowerContainer.Add(widget.NewLabel(""))
			lowerContainer.Add(widget.NewLabel(""))
			lowerContainer.Add(widget.NewLabel(""))
			lowerContainer.Add(widget.NewLabel(""))
			lowerContainer.Resize(lowerContainer.MinSize())

			rootContainer.Add(upperContainer)
			rootContainer.Add(lowerContainer)

			rootContainer.Refresh()

			return rootContainer
		}, func(i binding.DataItem, o fyne.CanvasObject) {
			value := i.(binding.Untyped)
			obj, _ := value.Get()
			ps := obj.(model.PlayerState)
			pl.objectMu.Lock()
			rootContainer := o.(*fyne.Container)
			upperContainer := rootContainer.Objects[0].(*fyne.Container)
			lowerContainer := rootContainer.Objects[1].(*fyne.Container)

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
			profileLabel.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: ps.Name, Style: stlOk}}
			profileLabel.Refresh()
			var vacState []string
			if ps.NumberOfVACBans > 0 {
				vacState = append(vacState, fmt.Sprintf("VB: %d", ps.NumberOfGameBans))
			}
			if ps.NumberOfGameBans > 0 {
				vacState = append(vacState, fmt.Sprintf("GB: %d (%d days)", ps.NumberOfVACBans, ps.DaysSinceLastBan))
			}
			if ps.CommunityBanned {
				vacState = append(vacState, "CB: ✓")
			}
			if ps.EconomyBan {
				vacState = append(vacState, "EB: ✓")
			}
			vacStyle := stlBad
			if len(vacState) == 0 {
				vacState = append(vacState, "✓")
				vacStyle = stlOk

			}
			vacLabel := upperContainer.Objects[2].(*widget.RichText)
			vacLabel.Segments = []widget.RichTextSegment{
				&widget.TextSegment{Text: strings.Join(vacState, ", "), Style: vacStyle},
			}
			vacLabel.Refresh()

			lowerContainer.Objects[0].(*widget.Label).SetText(fmt.Sprintf("K: %d", ps.Kills))
			lowerContainer.Objects[1].(*widget.Label).SetText(fmt.Sprintf("D: %d", ps.Deaths))
			lowerContainer.Objects[2].(*widget.Label).SetText(fmt.Sprintf("TKA: %d", ps.KillsOn))
			lowerContainer.Objects[3].(*widget.Label).SetText(fmt.Sprintf("TKB: %d", ps.DeathsBy))
			lowerContainer.Objects[4].(*widget.Label).SetText(fmt.Sprintf("Ping: %d", ps.Ping))

			pl.objectMu.Unlock()

		})
	pl.list = playerListWidget
	pl.boundList = boundList
	pl.content = container.NewVScroll(playerListWidget)
	return pl
}
