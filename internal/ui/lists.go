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
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"log"
)

type ruleListConfigDialog struct {
	dialog.Dialog

	list      *widget.List
	boundList binding.UntypedList
	settings  *model.Settings
}

func newRuleListConfigDialog(parent fyne.Window, saveFn func() error, settings *model.Settings) dialog.Dialog {
	boundList := binding.BindUntypedList(&[]interface{}{})
	list := widget.NewListWithData(boundList, func() fyne.CanvasObject {
		return container.NewBorder(
			nil,
			nil,
			nil,
			container.NewHBox(
				widget.NewButtonWithIcon(translations.One(translations.LabelEdit),
					theme.DocumentCreateIcon(), func() {}),
				widget.NewButtonWithIcon(translations.One(translations.LabelDelete),
					theme.DeleteIcon(), func() {})),
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

		btnContainer := rootContainer.Objects[1].(*fyne.Container)
		editButton := btnContainer.Objects[0].(*widget.Button)
		editButton.OnTapped = func() {
			nameEntry := widget.NewEntryWithData(binding.BindString(&lc.Name))
			enabledEntry := widget.NewCheckWithData(translations.One(translations.LabelEnabled), binding.BindBool(&lc.Enabled))
			d := dialog.NewForm(
				translations.One(translations.LabelEdit),
				translations.One(translations.LabelApply),
				translations.One(translations.LabelClose),
				[]*widget.FormItem{
					{Text: translations.One(translations.LabelName), Widget: nameEntry},
					{Text: translations.One(translations.LabelURL), Widget: urlEntry},
					{Text: translations.One(translations.LabelEnabled), Widget: enabledEntry},
				}, func(valid bool) {
					if !valid {
						return
					}
					if errSave := saveFn(); errSave != nil {
						log.Printf("Failed to save list settings")
					}
					name.SetText(nameEntry.Text)
				}, parent)
			sz := d.MinSize()
			sz.Width = defaultDialogueWidth
			d.Resize(sz)
			d.Show()
		}
		deleteButton := btnContainer.Objects[1].(*widget.Button)
		deleteButton.OnTapped = func() {
			msg := translations.Tr(&i18n.Message{ID: string(translations.LabelConfirmDeleteList)},
				1, map[string]interface{}{"Name": lc.Name})
			confirm := dialog.NewConfirm(translations.One(translations.TitleDeleteConfirm), msg, func(b bool) {
				if !b {
					return
				}
				settings.Lock()
				var lists model.ListConfigCollection
				for _, list := range settings.Lists {
					if list == lc {
						continue
					}
					lists = append(lists, list)
				}
				settings.Lists = lists
				settings.Unlock()
				if errReload := boundList.Set(settings.Lists.AsAny()); errReload != nil {
					log.Printf("Failed to reload: %v\n", errReload)
				}
				if errSave := saveFn(); errSave != nil {
					log.Printf("Failed to save list settings")
				}
			}, parent)
			confirm.Show()
		}

	})

	toolBar := container.NewBorder(
		nil,
		nil, nil, container.NewHBox(
			widget.NewButtonWithIcon(translations.One(translations.LabelAdd), theme.ContentAddIcon(), func() {
				newNameEntry := widget.NewEntryWithData(binding.NewString())
				newNameEntry.Validator = validateName
				newNameFormItem := widget.NewFormItem(translations.One(translations.LabelName), newNameEntry)
				newUrlEntry := widget.NewEntryWithData(binding.NewString())
				newUrlEntry.Validator = validateUrl

				newUrl := widget.NewFormItem(translations.One(translations.LabelURL), newUrlEntry)
				newEnabledEntry := widget.NewCheckWithData("", binding.NewBool())
				newEnabled := widget.NewFormItem(translations.One(translations.LabelEnabled), newEnabledEntry)
				inputForm := dialog.NewForm(
					translations.One(translations.TitleImportUrl),
					translations.One(translations.LabelApply),
					translations.One(translations.LabelClose),
					[]*widget.FormItem{
						newNameFormItem, newUrl, newEnabled,
					}, func(valid bool) {
						if !valid {
							return
						}
						lc := &model.ListConfig{
							ListType: "",
							Name:     newNameEntry.Text,
							Enabled:  newEnabledEntry.Checked,
							URL:      newUrlEntry.Text,
						}
						settings.Lock()
						settings.Lists = append(settings.Lists, lc)
						settings.Unlock()
						if errAppend := boundList.Append(lc); errAppend != nil {
							log.Printf("Failed to update config list: %v", errAppend)
						}
						if errSave := saveFn(); errSave != nil {
							log.Printf("Failed to save list settings")
						}
					}, parent)
				sz := inputForm.MinSize()
				sz.Width = defaultDialogueWidth
				inputForm.Resize(sz)
				inputForm.Show()
			})),
		container.NewHBox())

	if errSet := boundList.Set(settings.Lists.AsAny()); errSet != nil {
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

	sz := configDialog.MinSize()
	sz.Width = defaultDialogueWidth
	sz.Height = 500
	configDialog.Resize(sz)
	//settingsWindow.Resize(fyne.NewSize(5050, 700))
	return &configDialog
}
