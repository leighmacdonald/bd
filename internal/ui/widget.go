package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const defaultDialogueWidth = 600

type contextMenuRichText struct {
	*widget.Button
	menu *fyne.Menu
}

func (b *contextMenuRichText) Tapped(e *fyne.PointEvent) {
	widget.ShowPopUpMenuAtPosition(b.menu, fyne.CurrentApp().Driver().CanvasForObject(b), e.AbsolutePosition)
}

func newContextMenuRichText(menu *fyne.Menu) *contextMenuRichText {
	return &contextMenuRichText{
		Button: widget.NewButtonWithIcon("", theme.AccountIcon(), func() {

		}),
		menu: menu,
	}
}

type contextMenuIcon struct {
	*widget.Icon
	menu *fyne.Menu
}

func (b *contextMenuIcon) Tapped(e *fyne.PointEvent) {
	widget.ShowPopUpMenuAtPosition(b.menu, fyne.CurrentApp().Driver().CanvasForObject(b), e.AbsolutePosition)
}

func newContextMenuIcon() *contextMenuIcon {
	return &contextMenuIcon{
		Icon: widget.NewIcon(theme.SettingsIcon()),
		menu: fyne.NewMenu(""),
	}
}

//
//type clickableIcon struct {
//	*widget.Icon
//	onClicked func()
//}
//
//func (b *clickableIcon) Tapped(e *fyne.PointEvent) {
//	b.onClicked()
//}
//
//func newClickableIcon(icon fyne.Resource, clickHandler func()) *clickableIcon {
//	return &clickableIcon{
//		Icon:      widget.NewIcon(icon),
//		onClicked: clickHandler,
//	}
//}
