package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/bd/translations"
	"log"
	"time"
)

func (ui *Ui) createGameChatMessageList() *baseListWidget {
	uml := newBaseListWidget()
	uml.SetupList(func() fyne.CanvasObject {
		return container.NewBorder(
			nil,
			nil,
			container.NewHBox(widget.NewLabel(""), newContextMenuRichText(nil)),
			nil,
			widget.NewRichTextWithText(""))
	},
		func(i binding.DataItem, o fyne.CanvasObject) {
			value := i.(binding.Untyped)
			obj, errObj := value.Get()
			if errObj != nil {
				log.Printf("Failed to get bound value: %v", errObj)
				return
			}
			um := obj.(model.UserMessage)
			uml.objectMu.Lock()
			rootContainer := o.(*fyne.Container)
			timeAndProfileContainer := rootContainer.Objects[1].(*fyne.Container)
			timeStamp := timeAndProfileContainer.Objects[0].(*widget.Label)
			profileButton := timeAndProfileContainer.Objects[1].(*contextMenuRichText)
			messageRichText := rootContainer.Objects[0].(*widget.RichText)

			timeStamp.SetText(um.Created.Format(time.Kitchen))
			profileButton.SetText(um.Player)
			sz := profileButton.Size()
			sz.Width = 200
			profileButton.Resize(sz)
			profileButton.menu = ui.generateUserMenu(um.PlayerSID, um.UserId)
			profileButton.menu.Refresh()
			profileButton.Refresh()
			nameStyle := widget.RichTextStyleInline
			if um.Team == model.Red {
				nameStyle.ColorName = theme.ColorNameError
			} else {
				nameStyle.ColorName = theme.ColorNamePrimary
			}
			messageRichText.Segments[0] = &widget.TextSegment{
				Style: nameStyle,
				Text:  um.Formatted(),
			}
			messageRichText.Refresh()

			uml.objectMu.Unlock()
		})

	uml.SetContent(container.NewBorder(
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				widget.NewCheckWithData(translations.One(translations.LabelAutoScroll), uml.autoScrollEnabled),
				widget.NewButtonWithIcon(translations.One(translations.LabelBottom), theme.MoveDownIcon(), uml.list.ScrollToBottom),
				widget.NewButtonWithIcon(translations.One(translations.LabelClear), theme.ContentClearIcon(), func() {
					if errReload := uml.Reload(nil); errReload != nil {
						log.Printf("Failed to clear chat: %v\n", errReload)
					}
				}),
			),
			widget.NewLabelWithData(binding.IntToStringWithFormat(uml.messageCount, fmt.Sprintf("%s%%d", translations.One(translations.LabelMessageCount)))),
			widget.NewLabel(""),
		),
		nil,
		nil,
		nil,
		container.NewVScroll(uml.list)))

	uml.content.Refresh()
	return uml
}

func (ui *Ui) createUserHistoryMessageList() *baseListWidget {
	uml := newBaseListWidget()
	uml.SetupList(
		func() fyne.CanvasObject {
			return container.NewBorder(
				nil,
				nil,
				widget.NewLabel(""),
				nil,
				widget.NewRichTextWithText(""))
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			value := i.(binding.Untyped)
			obj, _ := value.Get()
			um := obj.(model.UserMessage)
			uml.objectMu.Lock()
			rootContainer := o.(*fyne.Container)
			timeStamp := rootContainer.Objects[1].(*widget.Label)
			timeStamp.SetText(um.Created.Format(time.RFC822))
			messageRichText := rootContainer.Objects[0].(*widget.RichText)
			messageRichText.Segments[0] = &widget.TextSegment{
				Style: widget.RichTextStyleInline,
				Text:  um.Message,
			}
			messageRichText.Refresh()
			uml.objectMu.Unlock()
		})
	uml.SetContent(container.NewBorder(
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				widget.NewCheckWithData(translations.One(translations.LabelAutoScroll), uml.autoScrollEnabled),
				widget.NewButtonWithIcon(translations.One(translations.LabelBottom), theme.MoveDownIcon(), uml.list.ScrollToBottom),
				widget.NewButtonWithIcon(translations.One(translations.LabelClear), theme.ContentClearIcon(), func() {
					if errReload := uml.Reload(nil); errReload != nil {
						log.Printf("Failed to clear chat: %v\n", errReload)
					}
				}),
			),
			widget.NewLabelWithData(binding.IntToStringWithFormat(uml.messageCount, fmt.Sprintf("%s%%d", translations.One(translations.LabelMessageCount)))),
			widget.NewLabel(""),
		),
		nil,
		nil,
		nil,
		container.NewVScroll(uml.list)))
	return uml
}

func (ui *Ui) createChatWidget(msgList *baseListWidget) fyne.Window {
	chatWindow := ui.application.NewWindow(translations.One(translations.WindowChatHistoryGame))
	chatWindow.SetIcon(resourceIconPng)
	chatWindow.SetContent(msgList.Widget())
	chatWindow.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		chatWindow.Hide()
	})
	chatWindow.Resize(fyne.NewSize(1000, 500))
	chatWindow.SetCloseIntercept(chatWindow.Hide)
	return chatWindow
}
