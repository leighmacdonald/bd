package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"log"
)

type contextMenuLabel struct {
	*widget.Label
	menu *fyne.Menu
}

func (b *contextMenuLabel) Tapped(e *fyne.PointEvent) {
	log.Println("Got click")
	widget.ShowPopUpMenuAtPosition(b.menu, fyne.CurrentApp().Driver().CanvasForObject(b), e.AbsolutePosition)
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

func newTableButtonLabel(text string) *tableButtonLabel {
	menuItem1 := fyne.NewMenuItem("Copy SteamID", nil)

	subMenu := fyne.NewMenu("Sub Menu",
		fyne.NewMenuItem("Racist", nil),
		fyne.NewMenuItem("Extra Racist", nil),
		fyne.NewMenuItem("Ultra Racist", nil),
	)

	externalSubMenu := fyne.NewMenu("Sub Menu",
		fyne.NewMenuItem("RGL", nil),
		fyne.NewMenuItem("Steamid.io", nil),
		fyne.NewMenuItem("ESEA", nil),
	)

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
