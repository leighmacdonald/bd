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
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/internal/translations"
	"github.com/pkg/errors"
	"log"
	"sync"
	"time"
)

type gameChatWindow struct {
	window            fyne.Window
	ctx               context.Context
	app               fyne.App
	list              *widget.List
	boundList         binding.UntypedList
	objectMu          *sync.RWMutex
	boundListMu       *sync.RWMutex
	messageCount      binding.Int
	autoScrollEnabled binding.Bool
	avatarCache       *avatarCache
	cb                callBacks
}

func newGameChatWindow(ctx context.Context, app fyne.App, cb callBacks, attrs binding.StringList, settings *model.Settings, cache *avatarCache) *gameChatWindow {
	window := app.NewWindow(translations.One(translations.WindowChatHistoryGame))
	window.Canvas().AddShortcut(
		&desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: fyne.KeyModifierControl},
		func(shortcut fyne.Shortcut) {
			window.Hide()
		})
	window.SetCloseIntercept(func() {
		window.Hide()
	})
	gcw := gameChatWindow{
		window:            window,
		ctx:               ctx,
		app:               app,
		boundList:         binding.BindUntypedList(&[]interface{}{}),
		autoScrollEnabled: binding.NewBool(),
		messageCount:      binding.NewInt(),
		boundListMu:       &sync.RWMutex{},
		objectMu:          &sync.RWMutex{},
		avatarCache:       cache,
		cb:                cb,
	}

	if errSet := gcw.autoScrollEnabled.Set(true); errSet != nil {
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
		gcw.objectMu.Lock()
		rootContainer := o.(*fyne.Container)
		timeAndProfileContainer := rootContainer.Objects[1].(*fyne.Container)
		timeStamp := timeAndProfileContainer.Objects[0].(*widget.Label)
		profileButton := timeAndProfileContainer.Objects[1].(*contextMenuRichText)
		messageRichText := rootContainer.Objects[0].(*widget.RichText)

		timeStamp.SetText(um.Created.Format(time.Kitchen))
		profileButton.SetText(um.Player)
		profileButton.SetIcon(gcw.avatarCache.GetAvatar(um.PlayerSID))
		profileButton.menu = generateUserMenu(gcw.ctx, app, gcw.window, um.PlayerSID, um.UserId, cb, attrs, settings.GetLinks())
		//profileButton.menu.Refresh()
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

		gcw.objectMu.Unlock()
	}
	gcw.list = widget.NewListWithData(gcw.boundList, createFunc, updateFunc)
	selected := "all"
	chatTypeEntry := widget.NewSelect([]string{
		string(model.ChatDestAll),
		string(model.ChatDestTeam),
		string(model.ChatDestParty),
	}, func(s string) {
		selected = s
	})
	chatTypeEntry.PlaceHolder = "Message..."
	chatTypeEntry.SetSelectedIndex(0)
	chatTypeEntry.Refresh()
	sz := chatTypeEntry.Size()
	sz.Width = 150
	chatTypeEntry.Resize(sz)
	chatEntryData := binding.NewString()
	messageEntry := widget.NewEntryWithData(chatEntryData)
	messageEntry.OnSubmitted = func(s string) {
		showUserError(gcw.cb.chatFunc(ctx, model.ChatDest(selected), s), gcw.window)
		_ = chatEntryData.Set("")
	}
	bottomContainer := container.NewBorder(
		nil,
		nil,
		nil,
		container.NewHBox(
			chatTypeEntry,
			widget.NewButtonWithIcon("Send", theme.MailSendIcon(), func() {
				msg, err := chatEntryData.Get()
				if err != nil {
					return
				}
				showUserError(gcw.cb.chatFunc(ctx, model.ChatDest(selected), msg), gcw.window)
				_ = chatEntryData.Set("")
			})),
		messageEntry)

	gcw.window.SetContent(container.NewBorder(
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				widget.NewCheckWithData(translations.One(translations.LabelAutoScroll), gcw.autoScrollEnabled),
				widget.NewButtonWithIcon(translations.One(translations.LabelBottom), theme.MoveDownIcon(), gcw.list.ScrollToBottom),
				widget.NewButtonWithIcon(translations.One(translations.LabelClear), theme.ContentClearIcon(), func() {
					if errReload := gcw.boundList.Set(nil); errReload != nil {
						log.Printf("Failed to clear chat: %v\n", errReload)
					}
				}),
			),
			widget.NewLabelWithData(binding.IntToStringWithFormat(gcw.messageCount, fmt.Sprintf("%s%%d", translations.One(translations.LabelMessageCount)))),
			widget.NewLabel(""),
		),
		bottomContainer,
		nil,
		nil,
		container.NewVScroll(gcw.list)))
	gcw.window.Resize(fyne.NewSize(sizeWindowChatWidth, sizeWindowChatHeight))
	return &gcw
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
