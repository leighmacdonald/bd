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
	var selectedId widget.ListItemID
	_ = lcd.boundList.Set(links)
	lcd.list = widget.NewListWithData(lcd.boundList, func() fyne.CanvasObject {
		return container.NewBorder(
			nil,
			nil,
			widget.NewCheck("", func(b bool) {}),
			container.NewHBox(
				widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {})),
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
		btnContainer := rootContainer.Objects[2].(*fyne.Container)
		editButton := btnContainer.Objects[0].(*widget.Button)

		enabledCheck := rootContainer.Objects[1].(*widget.Check)

		enabledBinding := binding.BindBool(&linkConfig.Enabled)
		enabledCheck.Bind(enabledBinding)

		urlEntry := widget.NewEntryWithData(binding.BindString(&linkConfig.URL))

		nameBinding := binding.BindString(&linkConfig.Name)
		labelName.Bind(nameBinding)
		nameEntry := widget.NewEntryWithData(nameBinding)
		editButton.OnTapped = func() {
			enabledEntry := widget.NewCheckWithData(translations.One(translations.LabelEnabled), enabledBinding)
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
			sz.Width = sizeDialogueWidth
			d.Resize(sz)
			d.Show()
		}
		editButton.Refresh()
	})
	count := 1
	delButton := widget.NewButtonWithIcon(translations.One(translations.LabelDelete), theme.ContentRemoveIcon(), func() {})

	lcd.list.OnSelected = func(id widget.ListItemID) {
		selectedId = id
		delButton.Enable()
	}
	lcd.list.OnUnselected = func(id widget.ListItemID) {
		delButton.Disable()
		selectedId = -1
	}

	addButton := widget.NewButtonWithIcon(translations.One(translations.LabelAdd), theme.ContentAddIcon(), func() {
		lcd.boundListMu.Lock()
		newLinks := settings.GetLinks()
		newLinks = append(newLinks, &model.LinkConfig{
			IdFormat: string(model.Steam64),
			Enabled:  true,
			Name:     fmt.Sprintf("New Link %d", count)})
		settings.SetLinks(newLinks)
		showUserError(lcd.boundList.Set(settings.GetLinks().AsAny()),
			parent)
		lcd.boundListMu.Unlock()
		lcd.list.Refresh()
		count++
	})

	delButton.OnTapped = func() {
		msg := translations.Tr(&i18n.Message{ID: string(translations.LabelConfirmDeleteList)},
			1, map[string]interface{}{"Name": ""})
		confirm := dialog.NewConfirm(translations.One(translations.TitleDeleteConfirm), msg, func(b bool) {
			if !b {
				return
			}
			var updatedLinks model.LinkConfigCollection
			for idx, link := range settings.GetLinks() {
				if idx == selectedId {
					continue
				}
				updatedLinks = append(updatedLinks, link)
			}
			settings.SetLinks(updatedLinks)
			lcd.boundListMu.Lock()
			if errReload := lcd.boundList.Set(settings.GetLinks().AsAny()); errReload != nil {
				log.Printf("Failed to reload: %v\n", errReload)
			}
			lcd.boundListMu.Unlock()
			lcd.list.Refresh()

		}, parent)
		confirm.Show()
	}
	delButton.Refresh()
	lcd.Dialog = dialog.NewCustom("Edit Links", translations.One(translations.LabelClose),
		container.NewBorder(container.NewHBox(addButton, delButton), nil, nil, nil, lcd.list), parent)

	lcd.Resize(fyne.NewSize(sizeDialogueWidth, sizeDialogueHeight))
	return &lcd
}
