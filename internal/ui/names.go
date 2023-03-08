package ui

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/internal/tr"
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
	title := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{ID: "names_title", Other: "Username History: {{ .SteamID }}"},
		TemplateData: map[string]interface{}{
			"SteamId": sid64,
		}})
	appWindow := app.NewWindow(title)
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
		msg := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{ID: "error_names_empty", Other: "Names not found for: {{ .SteamID }}"},
			TemplateData: map[string]interface{}{
				"SteamId": sid64,
			}})
		names = append(names, model.UserNameHistory{Name: msg, FirstSeen: time.Now()})
	}
	if errSet := unl.boundList.Set(names.AsAny()); errSet != nil {
		log.Printf("Failed to set names list: %v\n", errSet)
	}
	if errSetCount := unl.nameCount.Set(unl.boundList.Length()); errSetCount != nil {
		log.Printf("Failed to set name count: %v", errSetCount)
	}
	labelAutoScroll := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "names_check_autoscroll", Other: "Auto-Scroll"}})
	labelBottom := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "names_button_bottom", Other: "Bottom"}})
	labelCount := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "names_label_count", Other: "Count: "}})
	unl.SetContent(container.NewBorder(
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				widget.NewCheckWithData(labelAutoScroll, unl.autoScrollEnabled),
				widget.NewButtonWithIcon(labelBottom, theme.MoveDownIcon(), unl.list.ScrollToBottom),
			),
			widget.NewLabelWithData(binding.IntToStringWithFormat(unl.nameCount, fmt.Sprintf("%s%%d", labelCount))),
			widget.NewLabel(""),
		),
		nil,
		nil,
		nil,
		container.NewVScroll(unl.list)))
	unl.Resize(fyne.NewSize(sizeDialogueWidth, sizeDialogueHeight))
	unl.Show()
	return unl
}
