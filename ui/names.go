package ui

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
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

type userNameWindow struct {
	fyne.Window

	list              *widget.List
	boundList         binding.ExternalUntypedList
	objectMu          sync.RWMutex
	boundListMu       sync.RWMutex
	nameCount         binding.Int
	autoScrollEnabled binding.Bool
}

func (nameList *userNameWindow) Reload(rr []model.UserNameHistory) error {
	bl := make([]interface{}, len(rr))
	for i, r := range rr {
		bl[i] = r
	}
	nameList.boundListMu.Lock()
	defer nameList.boundListMu.Unlock()
	if errSet := nameList.boundList.Set(bl); errSet != nil {
		log.Printf("failed to set player list: %v\n", errSet)
	}
	if errReload := nameList.boundList.Reload(); errReload != nil {
		return errReload
	}
	nameList.list.ScrollToBottom()
	return nil
}

// Widget returns the actual select list widget.
func (nameList *userNameWindow) Widget() *widget.List {
	return nameList.list
}

func newUserNameWindow(ctx context.Context, app fyne.App, namesFunc model.QueryNamesFunc, sid64 steamid.SID64) *userNameWindow {
	appWindow := app.NewWindow(translations.Tr(&i18n.Message{ID: string(translations.WindowNameHistory)}, 1, map[string]interface{}{
		"SteamId": sid64,
	}))
	appWindow.SetCloseIntercept(func() {
		appWindow.Hide()
	})
	unl := &userNameWindow{
		Window:            appWindow,
		boundList:         binding.BindUntypedList(&[]interface{}{}),
		autoScrollEnabled: binding.NewBool(),
		nameCount:         binding.NewInt(),
	}
	if errSet := unl.autoScrollEnabled.Set(true); errSet != nil {
		log.Printf("Failed to set default autoscroll: %v\n", errSet)
	}

	userMessageListWidget := widget.NewListWithData(
		unl.boundList,
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
			um := obj.(model.UserNameHistory)
			unl.objectMu.Lock()
			rootContainer := o.(*fyne.Container)
			timeStamp := rootContainer.Objects[1].(*widget.Label)
			timeStamp.SetText(um.FirstSeen.Format(time.RFC822))
			messageRichText := rootContainer.Objects[0].(*widget.RichText)
			messageRichText.Segments[0] = &widget.TextSegment{
				Style: widget.RichTextStyleInline,
				Text:  um.Name,
			}
			messageRichText.Refresh()
			unl.objectMu.Unlock()
		})
	unl.list = userMessageListWidget

	names, err := namesFunc(ctx, sid64)
	if err != nil {
		names = append(names, model.UserNameHistory{
			NameId:    0,
			Name:      fmt.Sprintf("No names found for steamid: %d", sid64),
			FirstSeen: time.Now(),
		})
	}
	if errSet := unl.boundList.Set(names.AsAny()); errSet != nil {
		log.Printf("Failed to set names list: %v\n", errSet)
	}
	if errSetCount := unl.nameCount.Set(unl.boundList.Length()); errSetCount != nil {
		log.Printf("Failed to set name count: %v", errSetCount)
	}
	unl.SetContent(container.NewBorder(
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				widget.NewCheckWithData(translations.One(translations.LabelAutoScroll), unl.autoScrollEnabled),
				widget.NewButtonWithIcon(translations.One(translations.LabelBottom), theme.MoveDownIcon(), unl.list.ScrollToBottom),
			),
			widget.NewLabelWithData(binding.IntToStringWithFormat(unl.nameCount, fmt.Sprintf("%s%%d", translations.One(translations.LabelMessageCount)))),
			widget.NewLabel(""),
		),
		nil,
		nil,
		nil,
		container.NewVScroll(unl.list)))
	unl.Resize(fyne.NewSize(600, 600))
	unl.Show()
	return unl
}
