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
	"github.com/leighmacdonald/bd/internal/tr"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.uber.org/zap"
	"sync"
	"time"
)

type userChatWindow struct {
	fyne.Window
	list              *widget.List
	boundList         binding.UntypedList
	objectMu          sync.RWMutex
	messageCount      binding.Int
	autoScrollEnabled binding.Bool
	queryFunc         model.QueryUserMessagesFunc

	logger *zap.Logger
}

func newUserChatWindow(ctx context.Context, queryFunc model.QueryUserMessagesFunc, sid64 steamid.SID64) *userChatWindow {
	appWindow := application.NewWindow(tr.Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "userchat_title",
			Other: "User Chat History: {{ .SteamId }}"},
		TemplateData: map[string]interface{}{
			"SteamId": sid64,
		}}))
	appWindow.SetCloseIntercept(func() {
		appWindow.Hide()
	})
	window := userChatWindow{
		Window:            appWindow,
		logger:            logger.Named("user_chat"),
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
		logger.Error("Failed to set default autoscroll", zap.Error(errSet))
	}

	window.list = widget.NewListWithData(window.boundList, func() fyne.CanvasObject {
		return container.NewBorder(
			nil,
			nil,
			widget.NewLabel(""),
			nil,
			widget.NewRichTextWithText(""))
	}, func(i binding.DataItem, o fyne.CanvasObject) {
		window.objectMu.Lock()
		defer window.objectMu.Unlock()
		value := i.(binding.Untyped)
		obj, _ := value.Get()
		um := obj.(model.UserMessage)
		rootContainer := o.(*fyne.Container)
		timeStamp := rootContainer.Objects[1].(*widget.Label)
		timeStamp.SetText(um.Created.Format(time.RFC822))
		messageRichText := rootContainer.Objects[0].(*widget.RichText)
		messageRichText.Segments[0] = &widget.TextSegment{
			Style: widget.RichTextStyleInline,
			Text:  um.Message,
		}
		messageRichText.Refresh()
	})
	labelAutoScroll := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "chatuser_check_auto_scroll", Other: "Auto-Scroll"}})
	buttonBottom := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "links_button_bottom", Other: "Bottom"}})
	buttonClear := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "links_button_clear", Other: "Clear"}})
	labelMessageCount := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "links_label_message_count", Other: "Messages: "}})
	window.SetContent(container.NewBorder(
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				widget.NewCheckWithData(labelAutoScroll, window.autoScrollEnabled),
				widget.NewButtonWithIcon(buttonBottom, theme.MoveDownIcon(), window.list.ScrollToBottom),
				widget.NewButtonWithIcon(buttonClear, theme.ContentClearIcon(), func() {
					if errReload := window.boundList.Set(nil); errReload != nil {
						logger.Error("Failed to clear chat", zap.Error(errReload))
					}
				}),
			),
			widget.NewLabelWithData(binding.IntToStringWithFormat(window.messageCount, fmt.Sprintf("%s%%d", labelMessageCount))),
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
			UserId:    -1,
			Message:   "No messages",
			Created:   time.Now(),
			Dead:      false,
			TeamOnly:  false,
		})
	}
	if errSet := window.boundList.Set(messages.AsAny()); errSet != nil {
		logger.Error("Failed to set messages", zap.Error(errSet))
	}
	_ = window.messageCount.Set(window.boundList.Length())
	if ase, errASE := window.autoScrollEnabled.Get(); errASE == nil && ase {
		window.list.ScrollToBottom()
	}
	window.Resize(fyne.NewSize(sizeWindowChatWidth, sizeWindowChatHeight))
	return &window
}
