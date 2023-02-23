package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/model"
	"log"
	"sync"
	"time"
)

type userNameList struct {
	list        *widget.List
	boundList   binding.ExternalUntypedList
	content     fyne.CanvasObject
	objectMu    sync.RWMutex
	boundListMu sync.RWMutex
}

func (nameList *userNameList) Reload(rr []model.UserNameHistory) error {
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
func (nameList *userNameList) Widget() *widget.List {
	return nameList.list
}

func (ui *Ui) createUserNameList() *userNameList {
	unl := &userNameList{}
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
	return unl
}
