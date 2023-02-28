package ui

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
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
	content           fyne.CanvasObject
	objectMu          sync.RWMutex
	boundListMu       sync.RWMutex
	messageCount      binding.Int
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

	unl := &userNameWindow{
		Window:            appWindow,
		boundList:         binding.BindUntypedList(&[]interface{}{}),
		autoScrollEnabled: binding.NewBool(),
		messageCount:      binding.NewInt(),
	}
	boundList := binding.BindUntypedList(&[]interface{}{})
	userMessageListWidget := widget.NewListWithData(
		boundList,
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
	unl.boundList = boundList
	unl.content = container.NewVScroll(userMessageListWidget)
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
	return unl
}
