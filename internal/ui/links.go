package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/tr"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.uber.org/zap"
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

func newLinksDialog(parent fyne.Window, logger *zap.Logger, settings *detector.UserSettings) *linksConfigDialog {
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
		linkConfig := obj.(*detector.LinkConfig)

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
			enabledEntry := widget.NewCheckWithData("", enabledBinding)
			formatEntry := widget.NewSelectEntry(lcd.selectOpts)
			formatEntry.Bind(binding.BindString(&linkConfig.IdFormat))

			msgName := &i18n.Message{ID: "links_label_name", Other: "Name"}
			msgUrl := &i18n.Message{ID: "links_label_url", Other: "URL"}
			msgFormat := &i18n.Message{ID: "links_label_format", Other: "Format"}
			msgEnabled := &i18n.Message{ID: "links_label_enabled", Other: "Enabled"}

			form := widget.NewForm([]*widget.FormItem{
				{Text: tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: msgName}), Widget: nameEntry},
				{Text: tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: msgUrl}), Widget: urlEntry},
				{Text: tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: msgFormat}), Widget: formatEntry},
				{Text: tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: msgEnabled}), Widget: enabledEntry},
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
	delButton := widget.NewButtonWithIcon(
		tr.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{ID: "links_label_delete", Other: "Delete"}}),
		theme.ContentRemoveIcon(),
		func() {})

	lcd.list.OnSelected = func(id widget.ListItemID) {
		selectedId = id
		delButton.Enable()
	}
	lcd.list.OnUnselected = func(id widget.ListItemID) {
		delButton.Disable()
		selectedId = -1
	}
	addMsg := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "gamechat_label_add", Other: "Add"}})
	addButton := widget.NewButtonWithIcon(addMsg, theme.ContentAddIcon(), func() {
		lcd.boundListMu.Lock()
		newLinks := settings.GetLinks()
		newLinks = append(newLinks, &detector.LinkConfig{
			IdFormat: string(detector.Steam64),
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
		title := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "links_title_delete", Other: "Delete Link"}})
		msg := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "links_label_confirm", Other: "Are you sure?"}})
		confirm := dialog.NewConfirm(title, msg, func(b bool) {
			if !b {
				return
			}
			var updatedLinks detector.LinkConfigCollection
			for idx, link := range settings.GetLinks() {
				if idx == selectedId {
					continue
				}
				updatedLinks = append(updatedLinks, link)
			}
			settings.SetLinks(updatedLinks)
			lcd.boundListMu.Lock()
			if errReload := lcd.boundList.Set(settings.GetLinks().AsAny()); errReload != nil {
				logger.Error("Failed to reload links list", zap.Error(errReload))
			}
			lcd.boundListMu.Unlock()
			lcd.list.Refresh()

		}, parent)
		confirm.Show()
	}
	delButton.Refresh()

	title := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "links_title_edit", Other: "Edit Link"}})
	closeLabel := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "links_button_close", Other: "Close"}})
	lcd.Dialog = dialog.NewCustom(title, closeLabel,
		container.NewBorder(container.NewHBox(addButton, delButton), nil, nil, nil, lcd.list), parent)

	lcd.Resize(fyne.NewSize(sizeDialogueWidth, sizeDialogueHeight))
	return &lcd
}
