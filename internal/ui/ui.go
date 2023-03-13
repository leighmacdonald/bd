// Package ui provides a simple, cross-platform interface to the bot detector tool
//
// TODO
// - Use external data map/struct(?) for table data updates
// - Remove old players from state on configurable delay
package ui

import (
	"context"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/internal/platform"
	"github.com/leighmacdonald/bd/internal/tr"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
)

const (
	AppId   = "com.github.leighmacdonald.bd"
	urlHome = "https://github.com/leighmacdonald/bd"
	urlHelp = "https://github.com/leighmacdonald/bd/wiki"
)

const (
	sizeWindowMainWidth  = 800
	sizeWindowMainHeight = 1000

	sizeDialogueWidth  = 700
	sizeDialogueHeight = 500

	sizeWindowChatWidth  = 1000
	sizeWindowChatHeight = 500
)

func defaultApp() fyne.App {
	application := app.NewWithID(AppId)
	application.Settings().SetTheme(&bdTheme{})
	application.SetIcon(resourceIconPng)
	return application
}

type windows struct {
	player      *playerWindow
	chat        *gameChatWindow
	search      *searchWindow
	chatHistory map[steamid.SID64]*userChatWindow
	nameHistory map[steamid.SID64]*userNameWindow
}

type MenuCreator func(window fyne.Window, steamId steamid.SID64, userId int64) *fyne.Menu

type Ui struct {
	bd              *detector.BD
	application     fyne.App
	settings        *model.Settings
	windows         *windows
	knownAttributes binding.StringList
	avatarCache     *avatarCache
	version         model.Version
	logger          *zap.Logger
}

func (ui *Ui) UpdateServerState(state model.Server) {
	ui.windows.player.UpdateServerState(state)
}

func (ui *Ui) UpdatePlayerState(collection model.PlayerCollection) {
	ui.windows.player.updatePlayerState(collection)
}

func New(ctx context.Context, logger *zap.Logger, bd *detector.BD, settings *model.Settings, version model.Version) model.UserInterface {
	ui := Ui{
		bd:              bd,
		logger:          logger,
		version:         version,
		application:     defaultApp(),
		settings:        settings,
		knownAttributes: binding.NewStringList(),
		windows: &windows{
			chatHistory: map[steamid.SID64]*userChatWindow{},
			nameHistory: map[steamid.SID64]*userNameWindow{},
		},
		avatarCache: &avatarCache{
			RWMutex:    &sync.RWMutex{},
			userAvatar: make(map[steamid.SID64]fyne.Resource),
		},
	}

	ui.windows.chat = newGameChatWindow(ctx, &ui)

	ui.windows.search = newSearchWindow(ctx, &ui)

	ui.windows.player = ui.newPlayerWindow(
		ui.logger,
		func(window fyne.Window, steamId steamid.SID64, userId int64) *fyne.Menu {
			return generateUserMenu(ctx, window, &ui, steamId, userId, ui.knownAttributes)
		}, version)

	return &ui
}

func (ui *Ui) SetAvatar(sid64 steamid.SID64, data []byte) {
	if !sid64.Valid() || data == nil {
		return
	}
	ui.avatarCache.SetAvatar(sid64, data)
}

func (ui *Ui) Refresh() {
	ui.windows.chat.Content().Refresh()
	if ui.windows.player != nil {
		ui.windows.player.content.Refresh()
	}
}

func (ui *Ui) UpdateAttributes(attrs []string) {
	if err := ui.knownAttributes.Set(attrs); err != nil {
		ui.logger.Error("Failed to update known attribute", zap.Error(err))
	}
}

type playerSortType string

const (
	playerSortName   playerSortType = "Name"
	playerSortKills  playerSortType = "Kills"
	playerSortKD     playerSortType = "K:D"
	playerSortStatus playerSortType = "Status"
	playerSortTeam   playerSortType = "Team"
	playerSortTime   playerSortType = "Time"
)

var sortDirections = []playerSortType{playerSortName, playerSortKills, playerSortKD, playerSortStatus, playerSortTeam, playerSortTime}

func (ui *Ui) AddUserMessage(msg model.UserMessage) {
	if errAppend := ui.windows.chat.append(msg); errAppend != nil {
		ui.logger.Error("Failed to append game message", zap.Error(errAppend))
	}
	if userChat, found := ui.windows.chatHistory[msg.PlayerSID]; found {
		if errAppend := userChat.boundList.Append(msg); errAppend != nil {
			ui.logger.Error("Failed to append user history message", zap.Error(errAppend))
		}
		userChat.Content().Refresh()
	}
}

func (ui *Ui) createChatHistoryWindow(ctx context.Context, sid64 steamid.SID64) {
	_, found := ui.windows.chatHistory[sid64]
	if !found {
		ui.windows.chatHistory[sid64] = newUserChatWindow(ctx, ui.logger, ui.application, ui.bd.Store().FetchMessages, sid64)
	}
	ui.windows.chatHistory[sid64].Show()
}

func (ui *Ui) createNameHistoryWindow(ctx context.Context, sid64 steamid.SID64) {
	_, found := ui.windows.nameHistory[sid64]
	if !found {
		ui.windows.nameHistory[sid64] = newUserNameWindow(ctx, ui.logger, ui.application, ui.bd.Store().FetchNames, sid64)
	}
	ui.windows.nameHistory[sid64].Show()
}

func (ui *Ui) Start(ctx context.Context) {
	defer ui.bd.Shutdown()
	ui.bd.AttachGui(ui)
	go ui.bd.Start(ctx)
	ui.windows.player.window.Show()
	ui.application.Run()
	ctx.Done()
}

func (ui *Ui) Quit() {
	ui.application.Quit()
}

func showUserError(err error, parent fyne.Window) {
	if err == nil {
		return
	}
	d := dialog.NewError(err, parent)
	d.Show()
}

func validateUrl(urlString string) error {
	if urlString == "" {
		return nil
	}
	_, errParse := url.Parse(urlString)
	if errParse != nil {
		msg := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "error_invalid_url", Other: "Invalid URL"}})
		return errors.New(msg)
	}
	return nil
}

//func validateName(name string) error {
//	if len(name) == 0 {
//		return errors.New(tr.One(tr.ErrorNameEmpty))
//	}
//	return nil
//}

func validateSteamId(steamId string) error {
	if len(steamId) > 0 {
		_, err := steamid.StringToSID64(steamId)
		if err != nil {
			msg := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "error_invalid_steam_id", Other: "Invalid Steam ID"}})
			return errors.New(msg)
		}
	}
	return nil
}
func validateSteamRoot(newRoot string) error {
	if len(newRoot) > 0 {
		if !util.Exists(newRoot) {
			msg := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "error_invalid_path", Other: "Invalid Path"}})
			return errors.New(msg)
		}
		fp := filepath.Join(newRoot, platform.TF2RootValidationFile)
		if !util.Exists(fp) {
			msg := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "error_invalid_path",
					Other: "Invalid Path: {{ .FileName }}"},
				TemplateData: map[string]interface{}{
					"FileName": platform.TF2RootValidationFile,
				}})
			return errors.New(msg)
		}
	}
	return nil
}

func validateTags(tagStr string) error {
	var validTags []string
	for _, tag := range strings.Split(tagStr, ",") {
		normalized := strings.Trim(tag, " ")
		for _, vt := range validTags {
			if strings.EqualFold(vt, normalized) {
				msg := tr.Localizer.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "error_invalid_tag",
						Other: "Duplicate tag found: {{ .TagName }}"},
					TemplateData: map[string]interface{}{
						"TagName": vt,
					}})
				return errors.New(msg)
			}
		}
		validTags = append(validTags, normalized)
	}
	return nil
}
