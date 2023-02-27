package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/bd/translations"
	"github.com/pkg/errors"
	"log"
	"net/url"
)

func newRuleListConfigDialog(parent fyne.Window, saveFn func(), settings *model.Settings) dialog.Dialog {
	l := newBaseListWidget()
	l.SetupList(func() fyne.CanvasObject {
		return container.NewBorder(
			nil,
			nil,
			nil,
			container.NewHBox(widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {

			}), widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {

			})),
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
		urlEntry.Validator = func(s string) error {
			_, e := url.Parse(s)
			return e
		}
		btnContainer := rootContainer.Objects[1].(*fyne.Container)

		editButton := btnContainer.Objects[0].(*widget.Button)
		editButton.OnTapped = func() {
			nameEntry := widget.NewEntryWithData(binding.BindString(&lc.Name))
			enabledEntry := widget.NewCheckWithData("Enabled", binding.BindBool(&lc.Enabled))
			d := dialog.NewForm("Edit item", "Confirm", "Dismiss", []*widget.FormItem{
				{Text: "Name", Widget: nameEntry},
				{Text: "Url", Widget: urlEntry},
				{Text: "Enabled", Widget: enabledEntry},
			}, func(valid bool) {
				if !valid {
					return
				}
				saveFn()
				name.SetText(nameEntry.Text)
			}, parent)
			sz := d.MinSize()
			sz.Width = defaultDialogueWidth
			d.Resize(sz)
			d.Show()
		}
		deleteButton := btnContainer.Objects[1].(*widget.Button)
		deleteButton.OnTapped = func() {
			confirm := dialog.NewConfirm("Delete Confirmation", fmt.Sprintf("Are you are you want to delete the list?: %s", lc.Name), func(b bool) {
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
				if errReload := l.Reload(settings.Lists.AsAny()); errReload != nil {
					log.Printf("Failed to reload: %v\n", errReload)
				}
				saveFn()
			}, parent)
			confirm.Show()
		}

	})

	toolBar := container.NewBorder(
		nil,
		nil, nil, container.NewHBox(
			widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() {
				newNameEntry := widget.NewEntryWithData(binding.NewString())
				newName := widget.NewFormItem("Name", newNameEntry)
				newNameEntry.Validator = func(s string) error {
					if len(s) == 0 {
						return errors.New("Name cannot be empty")
					}
					return nil
				}
				newUrlEntry := widget.NewEntryWithData(binding.NewString())
				newUrlEntry.Validator = func(s string) error {
					_, e := url.Parse(s)
					if e != nil {
						return errors.New("Invalid URL")
					}
					return nil
				}
				newUrl := widget.NewFormItem("Update URL", newUrlEntry)
				newEnabledEntry := widget.NewCheckWithData("", binding.NewBool())
				newEnabled := widget.NewFormItem("Enabled", newEnabledEntry)
				inputForm := dialog.NewForm("Import URL", "Confirm", "Cancel", []*widget.FormItem{
					newName, newUrl, newEnabled,
				}, func(b bool) {
					if !b {
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
					if errAppend := l.boundList.Append(lc); errAppend != nil {
						log.Printf("Failed to update config list: %v", errAppend)
					}
					saveFn()
				}, parent)
				sz := inputForm.MinSize()
				sz.Width = defaultDialogueWidth
				inputForm.Resize(sz)
				inputForm.Show()
			})),
		container.NewHBox())
	l.SetContent(container.NewBorder(toolBar, nil, nil, nil, l.Widget()))
	if errSet := l.Reload(settings.Lists.AsAny()); errSet != nil {
		log.Printf("failed to load lists")
	}
	settingsWindow := dialog.NewCustom("List Config", translations.One(translations.LabelClose), l.Widget(), parent)
	sz := settingsWindow.MinSize()
	sz.Width = defaultDialogueWidth
	sz.Height = 500
	settingsWindow.Resize(sz)
	//settingsWindow.Resize(fyne.NewSize(5050, 700))
	return settingsWindow
}
