package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/pkg/errors"
	"log"
	"sync"
)

type baseListWidget struct {
	list              *widget.List
	boundList         binding.ExternalUntypedList
	content           fyne.CanvasObject
	objectMu          sync.RWMutex
	boundListMu       sync.RWMutex
	messageCount      binding.Int
	autoScrollEnabled binding.Bool
}

func (l *baseListWidget) Widget() fyne.CanvasObject {
	return l.content
}

func (l *baseListWidget) Reload(rr []any) error {
	l.boundListMu.Lock()
	defer l.boundListMu.Unlock()
	if errSet := l.boundList.Set(rr); errSet != nil {
		log.Printf("failed to set list: %v\n", errSet)
	}
	if errReload := l.boundList.Reload(); errReload != nil {
		return errReload
	}
	if errSet := l.messageCount.Set(l.boundList.Length()); errSet != nil {
		return errors.Wrapf(errSet, "Failed to set count")
	}
	l.scroll()
	l.list.Refresh()
	return nil
}

func (l *baseListWidget) scroll() {
	b, _ := l.autoScrollEnabled.Get()
	if b {
		l.list.ScrollToBottom()
	}
}

func (l *baseListWidget) Append(msg any) error {
	l.boundListMu.Lock()
	defer l.boundListMu.Unlock()
	if errSet := l.boundList.Append(msg); errSet != nil {
		log.Printf("failed to append item: %v\n", errSet)
	}
	if errSet := l.messageCount.Set(l.boundList.Length()); errSet != nil {
		return errors.Wrapf(errSet, "Failed to set count")
	}
	if errReload := l.boundList.Reload(); errReload != nil {
		log.Printf("Failed to update list: %v\n", errReload)
	}
	l.scroll()
	return nil
}

func newBaseListWidget() *baseListWidget {
	return &baseListWidget{
		boundList:         binding.BindUntypedList(&[]interface{}{}),
		autoScrollEnabled: binding.NewBool(),
		messageCount:      binding.NewInt(),
	}
}

func (l *baseListWidget) SetupList(createItem func() fyne.CanvasObject, updateItem func(i binding.DataItem, o fyne.CanvasObject)) {
	l.list = widget.NewListWithData(l.boundList, createItem, updateItem)
	l.SetContent(container.NewVScroll(l.list))
}

func (l *baseListWidget) SetContent(o fyne.CanvasObject) {
	l.content = o
}

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
