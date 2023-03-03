package ui

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/internal/translations"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"log"
	"net/url"
	"sort"
	"strings"
)

func newMenuItem(key translations.Key, fn func()) *fyne.MenuItem {
	return &fyne.MenuItem{
		Label:  translations.One(key),
		Action: fn,
	}
}

type menuButton struct {
	widget.Button
	menu *fyne.Menu
}

func (m *menuButton) Tapped(event *fyne.PointEvent) {
	widget.ShowPopUpMenuAtPosition(m.menu, fyne.CurrentApp().Driver().CanvasForObject(m), event.AbsolutePosition)
}

func newMenuButton(menu *fyne.Menu) *menuButton {
	c := &menuButton{menu: menu}
	c.ExtendBaseWidget(c)
	c.SetIcon(theme.SettingsIcon())

	return c
}

const newItemLabel = "New..."

func generateAttributeMenu(window fyne.Window, sid64 steamid.SID64, attrList binding.StringList, markFunc model.MarkFunc) *fyne.Menu {
	mkAttr := func(attrName string) func() {
		clsAttribute := attrName
		clsSteamId := sid64
		return func() {
			log.Printf("marking %d as %s", clsSteamId, clsAttribute)
			if errMark := markFunc(sid64, []string{clsAttribute}); errMark != nil {
				log.Printf("Failed to mark player: %v\n", errMark)
			}
		}
	}
	attrMenu := fyne.NewMenu(translations.One(translations.LabelMarkAs))
	knownAttributes, errGet := attrList.Get()
	if errGet != nil {
		log.Panicf("Failed to get list: %v\n", errGet)
	}
	sort.Slice(knownAttributes, func(i, j int) bool {
		return strings.ToLower(knownAttributes[i]) < strings.ToLower(knownAttributes[j])
	})
	for _, mi := range knownAttributes {
		attrMenu.Items = append(attrMenu.Items, fyne.NewMenuItem(mi, mkAttr(mi)))
	}
	entry := widget.NewEntry()
	entry.Validator = func(s string) error {
		if s == "" {
			return errors.New(translations.One(translations.ErrorAttributeEmpty))
		}
		for _, knownAttr := range knownAttributes {
			if strings.EqualFold(knownAttr, s) {
				return errors.New(translations.One(translations.ErrorAttributeDuplicate))
			}
		}
		return nil
	}
	fi := widget.NewFormItem(translations.One(translations.LabelAttributeName), entry)

	attrMenu.Items = append(attrMenu.Items, fyne.NewMenuItem(newItemLabel, func() {
		w := dialog.NewForm(
			translations.One(translations.WindowMarkCustom),
			translations.One(translations.LabelApply),
			translations.One(translations.LabelClose),
			[]*widget.FormItem{fi}, func(success bool) {
				if !success {
					return
				}
				if errMark := markFunc(sid64, []string{entry.Text}); errMark != nil {
					log.Printf("Failed to mark player: %v\n", errMark)
				}
			}, window)
		w.Show()
	}))
	attrMenu.Refresh()
	return attrMenu
}

func generateExternalLinksMenu(steamId steamid.SID64, links []model.LinkConfig, urlOpener func(url *url.URL) error) *fyne.Menu {
	lk := func(link model.LinkConfig, sid64 steamid.SID64, urlOpener func(url *url.URL) error) func() {
		clsLinkValue := link
		clsSteamId := sid64
		return func() {
			u := clsLinkValue.URL
			switch clsLinkValue.IdFormat {
			case model.Steam:
				u = fmt.Sprintf(u, steamid.SID64ToSID(clsSteamId))
			case model.Steam3:
				u = fmt.Sprintf(u, steamid.SID64ToSID3(clsSteamId))
			case model.Steam32:
				u = fmt.Sprintf(u, steamid.SID64ToSID32(clsSteamId))
			case model.Steam64:
				u = fmt.Sprintf(u, clsSteamId.Int64())
			default:
				log.Printf("Got unhandled steamid format, trying steam64: %v", clsLinkValue.IdFormat)
			}
			ul, urlErr := url.Parse(u)
			if urlErr != nil {
				log.Printf("Failed to create link: %v", urlErr)
				return
			}
			if errOpen := urlOpener(ul); errOpen != nil {
				log.Printf("Failed to open url: %v", errOpen)
			}
		}
	}

	var items []*fyne.MenuItem
	sort.Slice(links, func(i, j int) bool {
		return strings.ToLower(links[i].Name) < strings.ToLower(links[j].Name)
	})
	for _, link := range links {
		if !link.Enabled {
			continue
		}
		items = append(items, fyne.NewMenuItem(link.Name, lk(link, steamId, urlOpener)))
	}
	return fyne.NewMenu("Sub Menu", items...)
}

func generateSteamIdMenu(window fyne.Window, steamId steamid.SID64) *fyne.Menu {
	m := fyne.NewMenu(translations.One(translations.MenuCopySteamId),
		fyne.NewMenuItem(fmt.Sprintf("%d", steamId), func() {
			window.Clipboard().SetContent(fmt.Sprintf("%d", steamId))
		}),
		fyne.NewMenuItem(string(steamid.SID64ToSID(steamId)), func() {
			window.Clipboard().SetContent(string(steamid.SID64ToSID(steamId)))
		}),
		fyne.NewMenuItem(string(steamid.SID64ToSID3(steamId)), func() {
			window.Clipboard().SetContent(string(steamid.SID64ToSID3(steamId)))
		}),
		fyne.NewMenuItem(fmt.Sprintf("%d", steamid.SID64ToSID32(steamId)), func() {
			window.Clipboard().SetContent(fmt.Sprintf("%d", steamid.SID64ToSID32(steamId)))
		}),
	)
	return m
}

func generateKickMenu(ctx context.Context, userId int64, kickFunc model.KickFunc) *fyne.Menu {
	fn := func(reason model.KickReason) func() {
		return func() {
			log.Printf("Calling vote: %d %v", userId, reason)
			if errKick := kickFunc(ctx, userId, reason); errKick != nil {
				log.Printf("Error trying to call kick: %v\n", errKick)
			}
		}
	}
	return fyne.NewMenu(translations.One(translations.MenuCallVote),
		newMenuItem(translations.MenuVoteCheating, fn(model.KickReasonCheating)),
		newMenuItem(translations.MenuVoteIdle, fn(model.KickReasonIdle)),
		newMenuItem(translations.MenuVoteScamming, fn(model.KickReasonScamming)),
		newMenuItem(translations.MenuVoteOther, fn(model.KickReasonOther)),
	)
}

func generateUserMenu(ctx context.Context, app fyne.App, window fyne.Window, steamId steamid.SID64, userId int64, cb callBacks,
	knownAttributes binding.StringList, links []model.LinkConfig) *fyne.Menu {

	var items []*fyne.MenuItem
	if userId > 0 {
		items = append(items, &fyne.MenuItem{
			Icon:      theme.CheckButtonCheckedIcon(),
			ChildMenu: generateKickMenu(ctx, userId, cb.kickFunc),
			Label:     translations.One(translations.MenuCallVote)})
	}
	items = append(items, []*fyne.MenuItem{
		{
			Icon:      theme.ZoomFitIcon(),
			ChildMenu: generateAttributeMenu(window, steamId, knownAttributes, cb.markFn),
			Label:     translations.One(translations.MenuMarkAs)},
		{
			Icon:      theme.SearchIcon(),
			ChildMenu: generateExternalLinksMenu(steamId, links, app.OpenURL),
			Label:     translations.One(translations.MenuOpenExternal)},
		{
			Icon:      theme.ContentCopyIcon(),
			ChildMenu: generateSteamIdMenu(window, steamId),
			Label:     translations.One(translations.MenuCopySteamId)},
		{
			Icon: theme.ListIcon(),
			Action: func() {
				cb.createUserChat(steamId)
			},
			Label: translations.One(translations.MenuChatHistory)},
		{
			Icon: theme.VisibilityIcon(),
			Action: func() {
				cb.createNameHistory(steamId)
			},
			Label: translations.One(translations.MenuNameHistory)},
		{
			Icon: theme.VisibilityOffIcon(),
			Action: func() {
				if err := cb.whitelistFn(steamId); err != nil {
					showUserError(err, window)
				}
			},
			Label: translations.One(translations.MenuWhitelist)},
		{
			Icon: theme.DocumentCreateIcon(),
			Action: func() {
				offline := false
				player := cb.getPlayer(steamId)
				if player == nil {
					player = model.NewPlayer(steamId, "")
					if errOffline := cb.getPlayerOffline(ctx, steamId, player); errOffline != nil {
						showUserError(errors.Errorf("Unknown player: %v", errOffline), window)
						return
					}
					offline = true
				}
				entry := widget.NewMultiLineEntry()
				entry.SetMinRowsVisible(30)
				player.RLock()
				entry.SetText(player.Notes)
				player.RUnlock()
				item := widget.NewFormItem("", entry)
				sz := item.Widget.Size()
				sz.Height = 500
				item.Widget.Resize(sz)
				d := dialog.NewForm("Edit Player Notes", "Save", "Cancel", []*widget.FormItem{item}, func(b bool) {
					if !b {
						return
					}
					player.Lock()
					player.Notes = entry.Text
					player.Touch()
					player.Unlock()
					if offline {
						if errSave := cb.savePlayer(ctx, player); errSave != nil {
							log.Printf("Failed to save: %v\n", errSave)
						}
					}

				}, window)
				d.Resize(fyne.NewSize(700, 600))
				d.Show()
			},
			Label: "Edit Notes"},
	}...)
	menu := fyne.NewMenu("User Actions", items...)
	return menu
}
