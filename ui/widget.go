package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

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
