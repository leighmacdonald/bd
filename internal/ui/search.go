package ui

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/internal/translations"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type searchWindow struct {
	fyne.Window
	ctx         context.Context
	app         fyne.App
	list        *widget.List
	boundList   binding.ExternalUntypedList
	queryString binding.String
	objectMu    *sync.RWMutex
	boundListMu *sync.RWMutex
	resultCount binding.Int
	avatarCache *avatarCache
	queryEntry  *widget.Entry
	cb          callBacks
}

func (screen *searchWindow) Reload(results model.PlayerCollection) error {
	bl := results.AsAny()
	screen.boundListMu.Lock()
	if errSet := screen.boundList.Set(bl); errSet != nil {
		return errors.Wrapf(errSet, "failed to set player results")
	}
	if errReload := screen.boundList.Reload(); errReload != nil {
		return errors.Wrap(errReload, "Failed to reload results")
	}
	if errSet := screen.resultCount.Set(len(bl)); errSet != nil {
		return errors.Wrap(errSet, "Failed to set result count")
	}
	screen.boundListMu.Unlock()
	screen.list.Refresh()
	return nil
}

func newSearchWindow(ctx context.Context, app fyne.App, cb callBacks, attrs binding.StringList, settings *model.Settings, cache *avatarCache) *searchWindow {
	window := app.NewWindow(translations.One(translations.WindowPlayerSearch))
	window.Canvas().AddShortcut(
		&desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: fyne.KeyModifierControl},
		func(shortcut fyne.Shortcut) {
			window.Hide()
		})
	window.SetCloseIntercept(func() {
		window.Hide()
	})

	sw := searchWindow{
		Window:      window,
		ctx:         ctx,
		app:         app,
		list:        nil,
		boundList:   binding.BindUntypedList(&[]interface{}{}),
		objectMu:    &sync.RWMutex{},
		boundListMu: &sync.RWMutex{},
		avatarCache: cache,
		queryString: binding.NewString(),
		cb:          cb,
		resultCount: binding.NewInt(),
	}

	sw.list = widget.NewListWithData(sw.boundList, func() fyne.CanvasObject {
		return container.NewBorder(
			nil,
			nil,
			widget.NewLabel("Timestamp"),
			nil,
			newContextMenuRichText(nil))
	}, func(i binding.DataItem, o fyne.CanvasObject) {
		value := i.(binding.Untyped)
		obj, _ := value.Get()
		pl := obj.(*model.Player)
		sw.objectMu.Lock()

		rootContainer := o.(*fyne.Container)
		timeStamp := rootContainer.Objects[1].(*widget.Label)
		timeStamp.SetText(pl.UpdatedOn.Format(time.RFC822))

		profileButton := rootContainer.Objects[0].(*contextMenuRichText)
		profileButton.Alignment = widget.ButtonAlignLeading
		if pl.Name != "" {
			profileButton.SetText(pl.Name)
		} else {
			profileButton.SetText(pl.NamePrevious)
		}
		profileButton.SetIcon(sw.avatarCache.GetAvatar(pl.SteamId))
		profileButton.menu = generateUserMenu(sw.ctx, app, window, pl.SteamId, pl.UserId, cb, attrs, settings.Links)
		//profileButton.menu.Refresh()
		profileButton.Refresh()

		sw.objectMu.Unlock()
	})

	sw.queryEntry = widget.NewEntryWithData(sw.queryString)
	sw.queryEntry.PlaceHolder = "SteamID or Name"
	sw.queryEntry.OnSubmitted = func(s string) {
		results, errSearch := cb.searchPlayer(sw.ctx, model.SearchOpts{Query: s})
		if errSearch != nil {
			showUserError(errSearch, window)
			return
		}
		if errReload := sw.Reload(results); errReload != nil {
			showUserError(errReload, sw.Window)
		}
	}
	sw.SetContent(container.NewBorder(
		container.NewBorder(
			nil,
			nil,
			nil,
			widget.NewLabelWithData(binding.IntToStringWithFormat(
				sw.resultCount,
				fmt.Sprintf("%s%%d", translations.One(translations.LabelResultCount)))),
			container.NewMax(sw.queryEntry),
		),
		nil, nil, nil,
		container.NewMax(sw.list)))
	sw.Window.Resize(fyne.NewSize(500, 700))

	return &sw
}
