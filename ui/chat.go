package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

type chatListWidget struct {
	list *widget.List
}

func (ui *Ui) newChatListWidget() *chatListWidget {
	boundList := binding.BindUntypedList(&[]interface{}{})
	userMessageListWidget := widget.NewListWithData(
		boundList,
		func() fyne.CanvasObject {
			return container.NewHSplit(widget.NewLabel(""), widget.NewLabel(""))
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			//if id+1 > len(ui.messages) {
			//	return
			//}
			//itm := ui.messages[id]
			//cnt := item.(*container.Split)
			//a := cnt.Leading.(*widget.Label)
			//a.SetText(itm.Created.Format("3:04PM"))
			//b := cnt.Trailing.(*widget.Label)
			//b.SetText(itm.Message)
		})

	return &chatListWidget{
		list: userMessageListWidget,
	}
}

func (ui *Ui) createChatWidget() fyne.Window {
	//chatWidget := ui.newChatListWidget()
	chatWindow := ui.application.NewWindow("Chat")
	//chatWindow.SetContent(chatWidget)
	chatWindow.Resize(fyne.NewSize(1000, 500))
	chatWindow.SetCloseIntercept(func() {
		chatWindow.Hide()
	})

	return chatWindow
}
