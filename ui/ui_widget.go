package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"log"
	"net/url"
)

type contextMenuLabel struct {
	*widget.Label
	menu *fyne.Menu
}

func (b *contextMenuLabel) Tapped(e *fyne.PointEvent) {
	log.Println("Got click")
	widget.ShowPopUpMenuAtPosition(b.menu, fyne.CurrentApp().Driver().CanvasForObject(b), e.AbsolutePosition)
}

type chatRow struct {
	*container.Split
	message model.UserMessage
}

func newContextMenuLabel(text string) *contextMenuLabel {
	menuItem1 := fyne.NewMenuItem("A", nil)
	menuItem2 := fyne.NewMenuItem("B", nil)

	subMenu := fyne.NewMenu("Sub Menu",
		fyne.NewMenuItem("A", nil),
		fyne.NewMenuItem("A", nil),
	)

	menuItem3 := fyne.NewMenuItem("Mark As", nil)
	menu := fyne.NewMenu("File", &fyne.MenuItem{ChildMenu: subMenu, Label: "Mark As..."}, menuItem1, menuItem2, menuItem3)

	l := &widget.Label{
		Text:      text,
		Alignment: fyne.TextAlignLeading,
		TextStyle: fyne.TextStyle{},
	}

	l.ExtendBaseWidget(nil)

	return &contextMenuLabel{
		Label: l,
		menu:  menu,
	}
}

type tableButtonLabel struct {
	*widget.Icon
	menu *fyne.Menu
}

func (b *tableButtonLabel) Tapped(e *fyne.PointEvent) {
	log.Println("Got click")
	if b.menu != nil {
		widget.ShowPopUpMenuAtPosition(b.menu, fyne.CurrentApp().Driver().CanvasForObject(b), e.AbsolutePosition)
	}
}

type externalUrl struct {
	title  string
	url    string
	format string
}

func generateExternalLinksMenu(steamId steamid.SID64, urlOpener func(url *url.URL) error) []*fyne.MenuItem {
	links := []externalUrl{
		{title: "RGL", url: "https://rgl.gg/Public/PlayerProfile.aspx?p=%d", format: "steam64"},
	}
	var items []*fyne.MenuItem
	for _, link := range links {
		items = append(items, fyne.NewMenuItem(link.title, func() {
			u := link.url
			switch link.format {
			case "steam64":
				u = fmt.Sprintf(u, steamId.Int64())
			}
			ul, urlErr := url.Parse(u)
			if urlErr != nil {
				log.Printf("Failed to create link: %v", urlErr)
				return
			}
			if errOpen := urlOpener(ul); errOpen != nil {
				log.Printf("Failed to open url: %v", errOpen)
			}
		}))
	}
	return items
}

func generateUserMenu(steamId steamid.SID64, urlOpener func(url *url.URL) error, clipboard fyne.Clipboard) *fyne.Menu {
	copySteamIdSubMenu := fyne.NewMenu("Copy SteamID",
		fyne.NewMenuItem(fmt.Sprintf("%d", steamId), func() {
			clipboard.SetContent(fmt.Sprintf("%d", steamId))
		}),
		fyne.NewMenuItem(string(steamid.SID64ToSID(steamId)), func() {
			clipboard.SetContent(string(steamid.SID64ToSID(steamId)))
		}),
		fyne.NewMenuItem(string(steamid.SID64ToSID3(steamId)), func() {
			clipboard.SetContent(string(steamid.SID64ToSID3(steamId)))
		}),
		fyne.NewMenuItem(fmt.Sprintf("%d", steamid.SID64ToSID32(steamId)), func() {
			clipboard.SetContent(fmt.Sprintf("%d", steamid.SID64ToSID32(steamId)))
		}),
	)

	markAsSubMenu := fyne.NewMenu("Mark As",
		fyne.NewMenuItem("Racist", nil),
		fyne.NewMenuItem("Extra Racist", nil),
		fyne.NewMenuItem("Ultra Racist", nil),
	)

	externalSubMenu := fyne.NewMenu("Sub Menu", generateExternalLinksMenu(steamId, urlOpener)...)

	menu := fyne.NewMenu("User Actions",
		&fyne.MenuItem{ChildMenu: markAsSubMenu, Label: "Mark As..."},
		&fyne.MenuItem{ChildMenu: externalSubMenu, Label: "Open External..."},
		&fyne.MenuItem{ChildMenu: copySteamIdSubMenu, Label: "Copy SteamID"},
	)
	return menu
}

func (ui *Ui) newTableButtonLabel() *tableButtonLabel {
	l := widget.NewIcon(theme.SettingsIcon())
	l.ExtendBaseWidget(l)
	return &tableButtonLabel{
		Icon: l,
		menu: nil,
	}
}
