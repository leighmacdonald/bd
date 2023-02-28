package ui

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/bd/translations"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"log"
	"sync"
	"time"
)

type userChatWindow struct {
	fyne.Window

	app               fyne.App
	list              *widget.List
	boundList         binding.UntypedList
	objectMu          sync.RWMutex
	messageCount      binding.Int
	autoScrollEnabled binding.Bool
	queryFunc         model.QueryUserMessagesFunc
}

func newUserChatWindow(ctx context.Context, app fyne.App, queryFunc model.QueryUserMessagesFunc, sid64 steamid.SID64) *userChatWindow {
	appWindow := app.NewWindow(translations.Tr(&i18n.Message{ID: string(translations.WindowChatHistoryUser)},
		1, map[string]interface{}{
			"SteamId": sid64,
		}))

	window := userChatWindow{
		Window:            appWindow,
		app:               app,
		boundList:         binding.BindUntypedList(&[]interface{}{}),
		autoScrollEnabled: binding.NewBool(),
		messageCount:      binding.NewInt(),
		queryFunc:         queryFunc,
	}

	window.Canvas().AddShortcut(
		&desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: fyne.KeyModifierControl},
		func(shortcut fyne.Shortcut) {
			window.Hide()
		})
	if errSet := window.autoScrollEnabled.Set(true); errSet != nil {
		log.Printf("Failed to set default autoscroll: %v\n", errSet)
	}

	window.list = widget.NewListWithData(window.boundList, func() fyne.CanvasObject {
		return container.NewBorder(
			nil,
			nil,
			widget.NewLabel(""),
			nil,
			widget.NewRichTextWithText(""))
	}, func(i binding.DataItem, o fyne.CanvasObject) {
		value := i.(binding.Untyped)
		obj, _ := value.Get()
		um := obj.(model.UserMessage)
		window.objectMu.Lock()
		rootContainer := o.(*fyne.Container)
		timeStamp := rootContainer.Objects[1].(*widget.Label)
		timeStamp.SetText(um.Created.Format(time.RFC822))
		messageRichText := rootContainer.Objects[0].(*widget.RichText)
		messageRichText.Segments[0] = &widget.TextSegment{
			Style: widget.RichTextStyleInline,
			Text:  um.Message,
		}
		messageRichText.Refresh()
		window.objectMu.Unlock()
	})
	window.SetContent(container.NewBorder(
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				widget.NewCheckWithData(translations.One(translations.LabelAutoScroll), window.autoScrollEnabled),
				widget.NewButtonWithIcon(translations.One(translations.LabelBottom), theme.MoveDownIcon(), window.list.ScrollToBottom),
				widget.NewButtonWithIcon(translations.One(translations.LabelClear), theme.ContentClearIcon(), func() {
					if errReload := window.boundList.Set(nil); errReload != nil {
						log.Printf("Failed to clear chat: %v\n", errReload)
					}
				}),
			),
			widget.NewLabelWithData(binding.IntToStringWithFormat(window.messageCount, fmt.Sprintf("%s%%d", translations.One(translations.LabelMessageCount)))),
			widget.NewLabel(""),
		),
		nil,
		nil,
		nil,
		container.NewVScroll(window.list)))

	messages, errMessage := queryFunc(ctx, sid64)
	if errMessage != nil {
		messages = append(messages, model.UserMessage{
			MessageId: 0,
			Team:      0,
			Player:    "Bot Detector",
			PlayerSID: 0,
			UserId:    69,
			Message:   "No messages",
			Created:   time.Now(),
			Dead:      false,
			TeamOnly:  false,
		})
	}
	if errSet := window.boundList.Set(messages.AsAny()); errSet != nil {
		log.Printf("Failed to set messages: %v\n", errSet)
	}
	window.Resize(fyne.NewSize(600, 600))
	window.Show()
	return &window
}
