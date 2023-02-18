package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/model"
	"log"
	"net/url"
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

func (playerList *PlayerList) Reboot(rr []model.PlayerState) (err error) {

	defer func() {
		if err != nil {
			err = fmt.Errorf("options.Contact.Reboot: %w", err)
		}
	}()

	bl := make([]interface{}, len(rr))
	for i, r := range rr {
		bl[i] = r
	}

	playerList.boundListMu.Lock()
	defer playerList.boundListMu.Unlock()

	if errSet := playerList.boundList.Set(bl); errSet != nil {
		log.Printf("failed to set player list: %v\n", errSet)
	}
	if err = playerList.boundList.Reload(); err != nil {
		return
	}
	playerList.list.Refresh()
	return
}

// Widget returns the actual select list widget.
func (playerList *PlayerList) Widget() (w *widget.List) {
	w = playerList.list
	return
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

// ┌──────┬────────────────────────┬─────────────┬────────────┐
// │      │profile name            │ kills: 0    │ deaths: 0  │
// │  X   ├────────────────────────┼─────────────┼────────────┤
// │      │                        │ ping: 0     │┼┼┼┼┼┼┼┼┼┼┤►│
// └──────┴────────────────────────┴─────────────┴────────────┘
func newPlayerList(urlOpener func(url *url.URL) error, clipboard fyne.Clipboard) *PlayerList {
	iconSize := fyne.NewSize(64, 64)
	pl := &PlayerList{}
	boundList := binding.BindUntypedList(&[]interface{}{})
	playerListWidget := widget.NewListWithData(
		boundList,
		func() fyne.CanvasObject {
			rootContainer := container.NewVBox()

			lowerContainer := container.NewHBox()
			icon := widget.NewIcon(resourceUiResourcesDefaultavatarJpg)
			icon.Resize(iconSize)
			icon.Refresh()
			upperContainer := container.NewHBox(
				icon,
				widget.NewRichTextWithText(""),
				widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{}),
			)
			upperContainer.Resize(upperContainer.MinSize())
			lowerContainer.Add(newMenuButton(fyne.NewMenu("")))
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

			if ps.Avatar != nil {
				icon := upperContainer.Objects[0].(*widget.Icon)
				icon.SetResource(ps.Avatar)
				icon.Refresh()
				//iconContainer.Refresh()
			}

			profileLabel := upperContainer.Objects[1].(*widget.RichText)
			stl := widget.RichTextStyleStrong
			stl.ColorName = theme.ColorNameError
			profileLabel.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: ps.Name, Style: stl}}

			var vacState []string
			if ps.NumberOfVACBans > 0 {
				vacState = append(vacState, fmt.Sprintf("VB: %d", ps.NumberOfGameBans))
			}
			if ps.NumberOfGameBans > 0 {
				vacState = append(vacState, fmt.Sprintf("GB: %d (%d days)", ps.NumberOfVACBans, ps.DaysSinceLastBan))
			}
			if ps.CommunityBanned {
				vacState = append(vacState, "CB: Y")
			}
			if ps.EconomyBan {
				vacState = append(vacState, "EB: Y")
			}
			vacLabel := upperContainer.Objects[2].(*widget.Label)
			vacLabel.SetText(strings.Join(vacState, ", "))

			btn := lowerContainer.Objects[0].(*menuButton)
			btn.menu = generateUserMenu(ps.SteamId, urlOpener, clipboard)

			lowerContainer.Objects[1].(*widget.Label).SetText(fmt.Sprintf("K: %d", ps.Kills))
			lowerContainer.Objects[2].(*widget.Label).SetText(fmt.Sprintf("D: %d", ps.Deaths))
			lowerContainer.Objects[3].(*widget.Label).SetText(fmt.Sprintf("TKA: %d", ps.KillsOn))
			lowerContainer.Objects[4].(*widget.Label).SetText(fmt.Sprintf("TKB: %d", ps.DeathsBy))
			lowerContainer.Objects[5].(*widget.Label).SetText(fmt.Sprintf("Ping: %d", ps.Ping))

			upperContainer.Resize(upperContainer.MinSize())
			lowerContainer.Resize(lowerContainer.MinSize())
			rootContainer.Refresh()
			o.Resize(o.MinSize())
			o.Refresh()
			pl.objectMu.Unlock()

		})
	pl.list = playerListWidget
	pl.boundList = boundList
	pl.content = container.NewVScroll(playerListWidget)
	return pl
}
