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
	"github.com/pkg/errors"
	"log"
	"sync"
	"time"
)

type gameChatWindow struct {
	ctx               context.Context
	app               fyne.App
	window            fyne.Window
	list              *widget.List
	boundList         binding.UntypedList
	objectMu          sync.RWMutex
	boundListMu       sync.RWMutex
	messageCount      binding.Int
	autoScrollEnabled binding.Bool
}

func newGameChatWindow(ctx context.Context, app fyne.App, kickFunc model.KickFunc, attrs binding.StringList, markFunc model.MarkFunc,
	settings *model.Settings, createUserChat func(sid64 steamid.SID64), createNameHistory func(sid64 steamid.SID64)) *gameChatWindow {
	chatWindow := app.NewWindow(translations.One(translations.WindowChatHistoryGame))
	chatWindow.Canvas().AddShortcut(
		&desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: fyne.KeyModifierControl},
		func(shortcut fyne.Shortcut) {
			chatWindow.Hide()
		})

	window := gameChatWindow{
		ctx:               ctx,
		app:               app,
		window:            chatWindow,
		boundList:         binding.BindUntypedList(&[]interface{}{}),
		autoScrollEnabled: binding.NewBool(),
		messageCount:      binding.NewInt(),
	}
	if errSet := window.autoScrollEnabled.Set(true); errSet != nil {
		log.Printf("Failed to set default autoscroll: %v\n", errSet)
	}

	createFunc := func() fyne.CanvasObject {
		return container.NewBorder(
			nil,
			nil,
			container.NewHBox(widget.NewLabel(""), newContextMenuRichText(nil)),
			nil,
			widget.NewRichTextWithText(""))
	}
	updateFunc := func(i binding.DataItem, o fyne.CanvasObject) {
		value := i.(binding.Untyped)
		obj, errObj := value.Get()
		if errObj != nil {
			log.Printf("Failed to get bound value: %v", errObj)
			return
		}
		um := obj.(model.UserMessage)
		window.objectMu.Lock()
		rootContainer := o.(*fyne.Container)
		timeAndProfileContainer := rootContainer.Objects[1].(*fyne.Container)
		timeStamp := timeAndProfileContainer.Objects[0].(*widget.Label)
		profileButton := timeAndProfileContainer.Objects[1].(*contextMenuRichText)
		messageRichText := rootContainer.Objects[0].(*widget.RichText)

		timeStamp.SetText(um.Created.Format(time.Kitchen))
		profileButton.SetText(um.Player)
		profileButton.menu = generateUserMenu(window.ctx, app, window.window, um.PlayerSID, um.UserId,
			kickFunc, attrs, markFunc, settings.Links, createUserChat, createNameHistory)
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

		window.objectMu.Unlock()
	}
	window.list = widget.NewListWithData(window.boundList, createFunc, updateFunc)
	window.window.SetContent(container.NewBorder(
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
	chatWindow.Resize(fyne.NewSize(1000, 500))
	window.window.Content().Refresh()
	return &window
}

func (gcw *gameChatWindow) append(msg any) error {
	gcw.boundListMu.Lock()
	defer gcw.boundListMu.Unlock()
	if errSet := gcw.boundList.Append(msg); errSet != nil {
		log.Printf("failed to append item: %v\n", errSet)
	}
	if errSet := gcw.messageCount.Set(gcw.boundList.Length()); errSet != nil {
		return errors.Wrapf(errSet, "Failed to set count")
	}
	gcw.scroll()
	return nil
}

func (gcw *gameChatWindow) scroll() {
	b, _ := gcw.autoScrollEnabled.Get()
	if b {
		gcw.list.ScrollToBottom()
	}
}
