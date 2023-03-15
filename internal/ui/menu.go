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
	"github.com/leighmacdonald/bd/internal/tr"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net/url"
	"sort"
	"strings"
)

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
			showUserError(markFunc(clsSteamId, []string{clsAttribute}), window)
		}
	}
	markAsMenuLabel := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{ID: "menu_markas_label", Other: "Mark As..."}})
	markAsMenu := fyne.NewMenu(markAsMenuLabel)
	knownAttributes, _ := attrList.Get()
	sort.Slice(knownAttributes, func(i, j int) bool {
		return strings.ToLower(knownAttributes[i]) < strings.ToLower(knownAttributes[j])
	})
	for _, mi := range knownAttributes {
		markAsMenu.Items = append(markAsMenu.Items, fyne.NewMenuItem(mi, mkAttr(mi)))
	}
	entry := widget.NewEntry()
	entry.Validator = func(s string) error {
		if s == "" {
			msg := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{ID: "error_attribute_empty", Other: "Attribute cannot be empty"}})
			return errors.New(msg)
		}
		for _, knownAttr := range knownAttributes {
			if strings.EqualFold(knownAttr, s) {
				msg := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{ID: "error_attribute_duplicate", Other: "Duplicate attribute: {{ .Attr }} "},
					TemplateData:   map[string]any{"Attr": knownAttr}})
				return errors.New(msg)
			}
		}
		return nil
	}
	attributeLabel := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "mark_label_attr", Other: "Attribute Name"}})
	fi := widget.NewFormItem(attributeLabel, entry)
	markAsMenu.Items = append(markAsMenu.Items, fyne.NewMenuItem(newItemLabel, func() {
		title := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "mark_title", Other: "Add custom mark attribute"}})
		save := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "mark_button_save", Other: "Save"}})
		cancel := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "mark_button_cancel", Other: "Cancel"}})
		w := dialog.NewForm(title, save, cancel,
			[]*widget.FormItem{fi}, func(success bool) {
				if !success {
					return
				}
				showUserError(markFunc(sid64, []string{entry.Text}), window)
			}, window)
		w.Show()
	}))
	markAsMenu.Refresh()
	return markAsMenu
}

func generateExternalLinksMenu(logger *zap.Logger, steamId steamid.SID64, links model.LinkConfigCollection, urlOpener func(url *url.URL) error) *fyne.Menu {
	lk := func(link *model.LinkConfig, sid64 steamid.SID64, urlOpener func(url *url.URL) error) func() {
		clsLinkValue := link
		clsSteamId := sid64
		return func() {
			u := clsLinkValue.URL
			switch model.SteamIdFormat(clsLinkValue.IdFormat) {
			case model.Steam:
				u = fmt.Sprintf(u, steamid.SID64ToSID(clsSteamId))
			case model.Steam3:
				u = fmt.Sprintf(u, steamid.SID64ToSID3(clsSteamId))
			case model.Steam32:
				u = fmt.Sprintf(u, steamid.SID64ToSID32(clsSteamId))
			case model.Steam64:
				u = fmt.Sprintf(u, clsSteamId.Int64())
			default:
				logger.Error("Got unhandled steamid format, trying steam64", zap.String("format", clsLinkValue.IdFormat))
			}
			ul, urlErr := url.Parse(u)
			if urlErr != nil {
				logger.Error("Failed to create external link", zap.Error(urlErr), zap.String("url", u))
				return
			}
			if errOpen := urlOpener(ul); errOpen != nil {
				logger.Error("Failed to open url", zap.Error(errOpen), zap.String("url", u))
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
	title := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "menu_title_steam_id", Other: "Copy SteamID..."}})
	m := fyne.NewMenu(title,
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

func generateWhitelistMenu(parent fyne.Window, ui *Ui, steamID steamid.SID64) *fyne.Menu {
	title := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "menu_title_whitelist", Other: "Whitelist"}})
	labelAdd := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "menu_whitelist_add", Other: "Enable"}})
	labelRemove := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "menu_whitelist_remove", Other: "Disable"}})
	m := fyne.NewMenu(title,
		&fyne.MenuItem{
			Icon:  theme.ContentAddIcon(),
			Label: labelAdd,
			Action: func() {
				showUserError(ui.bd.OnWhitelist(steamID, true), parent)
			},
		},
		&fyne.MenuItem{
			Icon:  theme.ContentRemoveIcon(),
			Label: labelRemove,
			Action: func() {
				showUserError(ui.bd.OnWhitelist(steamID, false), parent)
			},
		},
	)
	return m
}

func generateKickMenu(parent fyne.Window, userId int64, kickFunc model.KickFunc) *fyne.Menu {
	fn := func(reason model.KickReason) func() {
		return func() {
			showUserError(kickFunc(userId, reason), parent)
		}
	}
	title := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "menu_title_call_vote", Other: "Call Vote..."}})
	labelCheating := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "menu_call_vote_cheating", Other: "Cheating"}})
	labelIdle := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "menu_call_vote_idle", Other: "Idle"}})
	labelScamming := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "menu_call_vote_scamming", Other: "Scamming"}})
	labelOther := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "menu_call_vote_other", Other: "Other"}})
	return fyne.NewMenu(title,
		&fyne.MenuItem{Label: labelCheating, Action: fn(model.KickReasonCheating)},
		&fyne.MenuItem{Label: labelIdle, Action: fn(model.KickReasonIdle)},
		&fyne.MenuItem{Label: labelScamming, Action: fn(model.KickReasonScamming)},
		&fyne.MenuItem{Label: labelOther, Action: fn(model.KickReasonOther)},
	)
}

func generateUserMenu(ctx context.Context, window fyne.Window, ui *Ui, steamId steamid.SID64, userId int64, knownAttributes binding.StringList) *fyne.Menu {
	kickTitle := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "user_menu_call_vote", Other: "Call Vote..."}})
	markTitle := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "user_menu_mark", Other: "Mark As..."}})
	unMarkTitle := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "user_menu_unmark", Other: "Unmark"}})
	externalTitle := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "user_menu_external", Other: "Open External..."}})
	steamIdTitle := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "user_menu_steam_id", Other: "Copy SteamID..."}})
	chatHistoryTitle := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "user_menu_chat_hist", Other: "View Chat History"}})
	nameHistoryTitle := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "user_menu_name_hist", Other: "View Name History"}})
	whitelistTitle := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "user_menu_whitelist", Other: "Whitelist"}})
	notesTitle := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "user_menu_notes", Other: "Edit Notes"}})
	var items []*fyne.MenuItem
	if userId > 0 {
		items = append(items, &fyne.MenuItem{
			Icon:      theme.CheckButtonCheckedIcon(),
			ChildMenu: generateKickMenu(window, userId, ui.bd.CallVote),
			Label:     kickTitle})
	}
	unMarkFn := func(steamID steamid.SID64) func() {
		clsSteamId := steamID
		return func() {
			showUserError(ui.bd.OnUnMark(clsSteamId), window)
		}
	}
	items = append(items, []*fyne.MenuItem{
		{
			Icon:      theme.ZoomFitIcon(),
			ChildMenu: generateAttributeMenu(window, steamId, knownAttributes, ui.bd.OnMark),
			Label:     markTitle,
		},
		{
			Icon:   theme.DeleteIcon(),
			Label:  unMarkTitle,
			Action: unMarkFn(steamId),
		},
		{
			Icon:      theme.SearchIcon(),
			ChildMenu: generateExternalLinksMenu(ui.logger, steamId, ui.settings.GetLinks(), ui.application.OpenURL),
			Label:     externalTitle},
		{
			Icon:      theme.ContentCopyIcon(),
			ChildMenu: generateSteamIdMenu(window, steamId),
			Label:     steamIdTitle},
		{
			Icon: theme.ListIcon(),
			Action: func() {
				ui.createChatHistoryWindow(ctx, steamId)
			},
			Label: chatHistoryTitle},
		{
			Icon: theme.VisibilityIcon(),
			Action: func() {
				ui.createNameHistoryWindow(ctx, steamId)
			},
			Label: nameHistoryTitle},
		{
			Icon:      theme.VisibilityOffIcon(),
			ChildMenu: generateWhitelistMenu(window, ui, steamId),
			Label:     whitelistTitle},
		{
			Icon: theme.DocumentCreateIcon(),
			Action: func() {
				offline := false
				player := ui.bd.GetPlayer(steamId)
				if player == nil {
					player = model.NewPlayer(steamId, "")
					if errOffline := ui.bd.Store().GetPlayer(ctx, steamId, player); errOffline != nil {
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
				sz.Height = sizeDialogueHeight
				item.Widget.Resize(sz)

				editNoteTitle := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "edit_note_title", Other: "Edit Player Notes"}})
				editNoteSave := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "edit_note_button_save", Other: "Save"}})
				editNoteCancel := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "edit_note_button_cancel", Other: "Cancel"}})

				d := dialog.NewForm(editNoteTitle, editNoteSave, editNoteCancel, []*widget.FormItem{item}, func(b bool) {
					if !b {
						return
					}
					player.Lock()
					player.Notes = entry.Text
					player.Touch()
					player.Unlock()
					if offline {
						if errSave := ui.bd.Store().SavePlayer(ctx, player); errSave != nil {
							ui.logger.Error("Failed to save player note", zap.Error(errSave))
						}
					}

				}, window)
				d.Resize(window.Canvas().Size())
				d.Show()
			},
			Label: notesTitle},
	}...)
	menu := fyne.NewMenu("User Actions", items...)
	return menu
}
