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
	"github.com/pkg/errors"
	"log"
	"sync"
	"time"
)

type userMessageList struct {
	list              *widget.List
	boundList         binding.ExternalUntypedList
	content           fyne.CanvasObject
	objectMu          sync.RWMutex
	boundListMu       sync.RWMutex
	messageCount      binding.Int
	autoScrollEnabled binding.Bool
}

func (chatList *userMessageList) Reload(rr []model.UserMessage) error {
	bl := make([]interface{}, len(rr))
	for i, r := range rr {
		bl[i] = r
	}
	chatList.boundListMu.Lock()
	defer chatList.boundListMu.Unlock()
	if errSet := chatList.boundList.Set(bl); errSet != nil {
		log.Printf("failed to set player list: %v\n", errSet)
	}
	if errSet := chatList.messageCount.Set(chatList.boundList.Length()); errSet != nil {
		return errors.Wrapf(errSet, "Failed to set message count")
	}
	if errReload := chatList.boundList.Reload(); errReload != nil {
		return errReload
	}
	chatList.list.ScrollToBottom()
	return nil
}

func (chatList *userMessageList) Append(msg model.UserMessage) error {
	chatList.boundListMu.Lock()
	defer chatList.boundListMu.Unlock()
	if errSet := chatList.boundList.Append(msg); errSet != nil {
		log.Printf("failed to append message: %v\n", errSet)
	}
	if errSet := chatList.messageCount.Set(chatList.boundList.Length()); errSet != nil {
		return errors.Wrapf(errSet, "Failed to set message count")
	}
	if errReload := chatList.boundList.Reload(); errReload != nil {
		log.Printf("Failed to update chat list: %v\n", errReload)
	}
	b, _ := chatList.autoScrollEnabled.Get()
	if b {
		chatList.list.ScrollToBottom()
	}
	return nil
}

// Widget returns the actual select list widget.
func (chatList *userMessageList) Widget() fyne.CanvasObject {
	return chatList.content
}

func (ui *Ui) createGameChatMessageList() *userMessageList {
	uml := ui.newMessageList()
	boundList := binding.BindUntypedList(&[]interface{}{})
	userMessageListWidget := widget.NewListWithData(
		boundList,
		func() fyne.CanvasObject {
			return container.NewBorder(
				nil,
				nil,
				container.NewHBox(widget.NewLabel(""), newContextMenuRichText(nil)),
				nil,
				widget.NewRichTextWithText(""))
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			value := i.(binding.Untyped)
			obj, _ := value.Get()
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

			messageRichText.Segments[0] = &widget.TextSegment{
				Style: widget.RichTextStyleInline,
				Text:  um.Message,
			}
			messageRichText.Refresh()

			uml.objectMu.Unlock()
		})
	uml.boundList = boundList
	uml.list = userMessageListWidget
	uml.content = container.NewBorder(
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				widget.NewCheckWithData(translations.One(translations.LabelAutoScroll), uml.autoScrollEnabled),
				widget.NewButtonWithIcon(translations.One(translations.LabelBottom), theme.MoveDownIcon(), func() {
					uml.list.ScrollToBottom()
				}),
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
		container.NewVScroll(userMessageListWidget))
	uml.content.Refresh()
	return uml
}

func (ui *Ui) newMessageList() *userMessageList {
	uml := userMessageList{
		autoScrollEnabled: binding.NewBool(),
		messageCount:      binding.NewInt(),
		boundList:         binding.BindUntypedList(&[]interface{}{}),
	}
	if errSetAS := uml.autoScrollEnabled.Set(true); errSetAS != nil {
		log.Printf("Failed to set auto-scroll preference: %v", errSetAS)
	}
	return &uml
}

func (ui *Ui) createUserHistoryMessageList() *userMessageList {
	uml := ui.newMessageList()
	userMessageListWidget := widget.NewListWithData(
		uml.boundList,
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
	uml.list = userMessageListWidget
	uml.content = container.NewBorder(
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				widget.NewCheckWithData(translations.One(translations.LabelAutoScroll), uml.autoScrollEnabled),
				widget.NewButtonWithIcon(translations.One(translations.LabelBottom), theme.MoveDownIcon(), func() {
					uml.list.ScrollToBottom()
				}),
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
		container.NewVScroll(userMessageListWidget))
	return uml
}

func (ui *Ui) createChatWidget(msgList *userMessageList) fyne.Window {
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
