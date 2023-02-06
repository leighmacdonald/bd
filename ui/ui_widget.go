package ui

import (
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
	widget.ShowPopUpMenuAtPosition(b.menu, fyne.CurrentApp().Driver().CanvasForObject(b), e.AbsolutePosition)
}

type externalUrl struct {
	title  string
	url    string
	format string
}

func (ui *Ui) generateExternalLinksMenu(steamId steamid.SID64) []*fyne.MenuItem {
	links := []externalUrl{
		{title: "RGL", url: "https://rgl.gg/Public/PlayerProfile.aspx?p=%d", format: "steam64"},
	}
	var items []*fyne.MenuItem
	for _, link := range links {
		items = append(items, fyne.NewMenuItem(link.title, func() {
			ul, urlErr := url.Parse(link.url)
			if urlErr != nil {
				log.Printf("Failed to create link: %v", urlErr)
				return
			}
			if errOpen := ui.application.OpenURL(ul); errOpen != nil {
				log.Printf("Failed to open external link: %v", errOpen)
			}
		}))
	}
	return items
}

func (ui *Ui) newTableButtonLabel(steamId steamid.SID64) *tableButtonLabel {
	menuItem1 := fyne.NewMenuItem("Copy SteamID", nil)

	subMenu := fyne.NewMenu("Sub Menu",
		fyne.NewMenuItem("Racist", nil),
		fyne.NewMenuItem("Extra Racist", nil),
		fyne.NewMenuItem("Ultra Racist", nil),
	)

	externalSubMenu := fyne.NewMenu("Sub Menu", ui.generateExternalLinksMenu(steamId)...)

	menu := fyne.NewMenu("Actions",
		&fyne.MenuItem{ChildMenu: subMenu, Label: "Mark As..."},
		&fyne.MenuItem{ChildMenu: externalSubMenu, Label: "Open External..."},
		menuItem1,
	)

	l := widget.NewIcon(theme.CheckButtonIcon())
	l.ExtendBaseWidget(l)
	return &tableButtonLabel{
		Icon: l,
		menu: menu,
	}
}
