package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/internal/translations"
	"sync"
)

type linksConfigDialog struct {
	dialog.Dialog

	list        *widget.List
	boundList   binding.UntypedList
	objectMu    *sync.RWMutex
	boundListMu *sync.RWMutex
	selectOpts  []string
}

func newLinksDialog(parent fyne.Window, settings *model.Settings) *linksConfigDialog {
	var links []any
	for _, l := range settings.GetLinks() {
		links = append(links, l)
	}
	lcd := linksConfigDialog{
		objectMu:    &sync.RWMutex{},
		boundListMu: &sync.RWMutex{},
		boundList:   binding.NewUntypedList(),
		selectOpts:  []string{"steam64", "steam32", "steam3", "steam"},
	}
	_ = lcd.boundList.Set(links)
	lcd.list = widget.NewListWithData(lcd.boundList, func() fyne.CanvasObject {
		return container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				widget.NewButtonWithIcon("Edit", theme.FolderNewIcon(), func() {}),
				widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {})),
			nil,
			widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: false}),
		)
	}, func(i binding.DataItem, object fyne.CanvasObject) {
		lcd.objectMu.Lock()
		defer lcd.objectMu.Unlock()
		value := i.(binding.Untyped)
		obj, _ := value.Get()
		linkConfig := obj.(*model.LinkConfig)

		rootContainer := object.(*fyne.Container)
		labelName := rootContainer.Objects[0].(*widget.Label)
		btnContainer := rootContainer.Objects[1].(*fyne.Container)
		editButton := btnContainer.Objects[0].(*widget.Button)
		//	deleteButton := btnContainer.Objects[1].(*widget.Button)

		urlEntry := widget.NewEntryWithData(binding.BindString(&linkConfig.URL))

		nameBinding := binding.BindString(&linkConfig.Name)
		labelName.Bind(nameBinding)
		nameEntry := widget.NewEntryWithData(nameBinding)
		editButton.OnTapped = func() {

			enabledEntry := widget.NewCheckWithData(translations.One(translations.LabelEnabled), binding.BindBool(&linkConfig.Enabled))
			formatEntry := widget.NewSelectEntry(lcd.selectOpts)
			formatEntry.Bind(binding.BindString(&linkConfig.IdFormat))

			form := widget.NewForm([]*widget.FormItem{
				{Text: translations.One(translations.LabelName), Widget: nameEntry},
				{Text: translations.One(translations.LabelURL), Widget: urlEntry},
				{Text: "Format", Widget: formatEntry},
				{Text: translations.One(translations.LabelEnabled), Widget: enabledEntry},
			}...)

			d := dialog.NewCustom("Edit Entry", "Close", container.NewMax(form), parent)
			sz := d.MinSize()
			sz.Width = defaultDialogueWidth
			d.Resize(sz)
			d.Show()
		}
		editButton.Refresh()
	})

	lcd.Dialog = dialog.NewCustom("Edit Links", "Dismiss",
		container.NewBorder(widget.NewToolbar(widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			showUserError(lcd.boundList.Append(&model.LinkConfig{IdFormat: string(model.Steam64), Enabled: true}), parent)
			lcd.list.Refresh()
		})), nil, nil, nil, lcd.list), parent)

	// func(b bool) {
	//			if !b {
	//				return
	//			}
	//			utLinks, errGet := lcd.boundList.Get()
	//			if errGet != nil {
	//				showUserError(errGet, parent)
	//				return
	//			}
	//			var linkConfigs []*model.LinkConfig
	//			for _, ut := range utLinks {
	//				link := ut.(*model.LinkConfig)
	//				if !link.Deleted {
	//					linkConfigs = append(linkConfigs, link)
	//				}
	//			}
	//			lcd.newLinkConfigSuccess = true
	//			lcd.newLinkConfig = linkConfigs
	//		}
	lcd.Resize(fyne.NewSize(800, 800))
	return &lcd
}
