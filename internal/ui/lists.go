package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/internal/translations"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"log"
)

type ruleListConfigDialog struct {
	dialog.Dialog

	list      *widget.List
	boundList binding.UntypedList
	settings  *model.Settings
}

func newRuleListConfigDialog(parent fyne.Window, settings *model.Settings) dialog.Dialog {
	boundList := binding.BindUntypedList(&[]interface{}{})
	list := widget.NewListWithData(boundList, func() fyne.CanvasObject {
		return container.NewBorder(
			nil,
			nil,
			widget.NewCheck("", func(b bool) {

			}),
			container.NewHBox(
				widget.NewButtonWithIcon(translations.One(translations.LabelEdit),
					theme.DocumentCreateIcon(), func() {}),
				widget.NewButtonWithIcon(translations.One(translations.LabelDelete),
					theme.DeleteIcon(), func() {}),
			),
			widget.NewLabel(""),
		)
	}, func(i binding.DataItem, o fyne.CanvasObject) {
		value := i.(binding.Untyped)
		obj, _ := value.Get()
		lc := obj.(*model.ListConfig)
		rootContainer := o.(*fyne.Container)

		name := rootContainer.Objects[0].(*widget.Label)
		name.Bind(binding.BindString(&lc.Name))

		urlEntry := widget.NewEntryWithData(binding.BindString(&lc.URL))
		urlEntry.Validator = validateUrl

		btnContainer := rootContainer.Objects[2].(*fyne.Container)
		editButton := btnContainer.Objects[0].(*widget.Button)
		deleteButton := btnContainer.Objects[1].(*widget.Button)
		enabledCheck := rootContainer.Objects[1].(*widget.Check)

		enabledCheck.Bind(binding.BindBool(&lc.Enabled))
		editButton.OnTapped = func() {
			nameEntry := widget.NewEntryWithData(binding.BindString(&lc.Name))
			//enabledEntry := widget.NewCheckWithData(translations.One(translations.LabelEnabled), binding.BindBool(&lc.Enabled))
			form := widget.NewForm([]*widget.FormItem{
				{Text: translations.One(translations.LabelName), Widget: nameEntry},
				{Text: translations.One(translations.LabelURL), Widget: urlEntry},
				{Text: translations.One(translations.LabelEnabled), Widget: enabledCheck},
			}...)

			d := dialog.NewCustom(
				translations.One(translations.LabelEdit),
				translations.One(translations.LabelClose),
				container.NewVScroll(container.NewMax(form)),
				parent)
			sz := d.MinSize()
			sz.Width = sizeDialogueWidth
			sz.Height *= 3
			d.Resize(sz)
			d.Show()
		}

		deleteButton.OnTapped = func() {
			msg := translations.Tr(&i18n.Message{ID: string(translations.LabelConfirmDeleteList)},
				1, map[string]interface{}{"Name": lc.Name})
			confirm := dialog.NewConfirm(translations.One(translations.TitleDeleteConfirm), msg, func(b bool) {
				if !b {
					return
				}
				var lists model.ListConfigCollection
				for _, list := range settings.GetLists() {
					if list == lc {
						continue
					}
					lists = append(lists, list)
				}
				settings.SetLists(lists)
				if errReload := boundList.Set(settings.GetLists().AsAny()); errReload != nil {
					log.Printf("Failed to reload: %v\n", errReload)
				}

			}, parent)
			confirm.Show()
		}

	})
	listCount := 1
	toolBar := container.NewBorder(
		nil,
		nil, container.NewHBox(
			widget.NewButtonWithIcon(translations.One(translations.LabelAdd), theme.ContentAddIcon(), func() {
				newLists := settings.GetLists()
				newLists = append(newLists, &model.ListConfig{
					ListType: model.ListTypeTF2BDPlayerList,
					Name:     fmt.Sprintf("New List %d", listCount),
					Enabled:  false,
					URL:      "",
				})
				settings.SetLists(newLists)
				if errAppend := boundList.Set(settings.GetLists().AsAny()); errAppend != nil {
					log.Printf("Failed to update config list: %v", errAppend)
				}
				list.Refresh()
			})), nil,
		container.NewHBox())

	if errSet := boundList.Set(settings.GetLists().AsAny()); errSet != nil {
		log.Printf("failed to load lists")
	}

	configDialog := ruleListConfigDialog{
		Dialog: dialog.NewCustom(
			translations.One(translations.TitleListConfig),
			translations.One(translations.LabelClose),
			container.NewBorder(toolBar, nil, nil, nil, list),
			parent,
		),
		list:      list,
		boundList: nil,
		settings:  settings,
	}

	configDialog.Resize(fyne.NewSize(sizeDialogueWidth, sizeDialogueWidth))
	return &configDialog
}
