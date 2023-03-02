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
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/internal/translations"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"log"
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

func defaultApp() fyne.App {
	application := app.NewWithID(AppId)
	application.Settings().SetTheme(&bdTheme{})
	application.SetIcon(resourceIconPng)
	return application
}

type windows struct {
	player      *playerWindow
	chat        *gameChatWindow
	chatHistory map[steamid.SID64]*userChatWindow
	nameHistory map[steamid.SID64]*userNameWindow
}

type callBacks struct {
	markFn                model.MarkFunc
	whitelistFn           model.SteamIDErrFunc
	kickFunc              model.KickFunc
	queryNamesFunc        model.QueryNamesFunc
	queryUserMessagesFunc model.QueryUserMessagesFunc
	gameLauncherFunc      model.LaunchFunc
	createUserChat        model.SteamIDFunc
	createNameHistory     model.SteamIDFunc
}

type MenuCreator func(window fyne.Window, steamId steamid.SID64, userId int64) *fyne.Menu

type Ui struct {
	ctx             context.Context
	bd              *detector.BD
	application     fyne.App
	boundSettings   boundSettings
	settings        *model.Settings
	windows         *windows
	callBacks       callBacks
	knownAttributes binding.StringList
	avatarCache     *avatarCache
}

func (ui *Ui) UpdateServerState(state model.Server) {
	ui.windows.player.UpdateServerState(state)
}

func (ui *Ui) UpdatePlayerState(collection model.PlayerCollection) {
	ui.windows.player.updatePlayerState(collection)
}

func New(ctx context.Context, bd *detector.BD, settings *model.Settings, store store.DataStore) model.UserInterface {
	ui := Ui{
		ctx:             ctx,
		bd:              bd,
		application:     defaultApp(),
		boundSettings:   boundSettings{binding.BindStruct(settings)},
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
		callBacks: callBacks{
			queryNamesFunc:        store.FetchNames,
			queryUserMessagesFunc: store.FetchMessages,
			kickFunc:              bd.CallVote,
			markFn:                bd.OnMark,
			gameLauncherFunc:      bd.LaunchGameAndWait,
			whitelistFn:           bd.OnWhitelist,
		},
	}
	ui.callBacks.createUserChat = func(sid64 steamid.SID64) {
		ui.createChatHistoryWindow(sid64)
	}
	ui.callBacks.createNameHistory = func(sid64 steamid.SID64) {
		ui.createNameHistoryWindow(sid64)
	}

	ui.windows.chat = newGameChatWindow(ui.ctx, ui.application, ui.callBacks, ui.knownAttributes, settings, ui.avatarCache)

	ui.windows.player = newPlayerWindow(
		ui.application,
		settings,
		ui.boundSettings,
		func() {
			ui.windows.chat.window.Show()
		},
		ui.callBacks,
		func(window fyne.Window, steamId steamid.SID64, userId int64) *fyne.Menu {
			return generateUserMenu(ui.ctx, ui.application, window, steamId, userId, ui.callBacks, ui.knownAttributes, ui.settings.Links)
		}, ui.avatarCache)

	return &ui
}

func (ui *Ui) SetAvatar(sid64 steamid.SID64, data []byte) {
	if !sid64.Valid() || data == nil {
		return
	}
	ui.avatarCache.SetAvatar(sid64, data)
}

func (ui *Ui) SetBuildInfo(version string, commit string, date string, builtBy string) {
	ui.windows.player.aboutDialog.SetBuildInfo(version, commit, date, builtBy)
}

func (ui *Ui) Refresh() {
	ui.windows.chat.window.Content().Refresh()
	if ui.windows.player != nil {
		ui.windows.player.content.Refresh()
	}
}

func (ui *Ui) UpdateAttributes(attrs []string) {
	if err := ui.knownAttributes.Set(attrs); err != nil {
		log.Printf("Failed to update known attribute: %v\n", err)
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
		log.Printf("Failed to append game message: %v", errAppend)
	}
	if userChat, found := ui.windows.chatHistory[msg.PlayerSID]; found {
		if errAppend := userChat.boundList.Append(msg); errAppend != nil {
			log.Printf("Failed to append user history message: %v", errAppend)
		}
		userChat.Content().Refresh()
	}
}

func (ui *Ui) createChatHistoryWindow(sid64 steamid.SID64) {
	_, found := ui.windows.chatHistory[sid64]
	if !found {
		ui.windows.chatHistory[sid64] = newUserChatWindow(ui.ctx, ui.application, ui.callBacks.queryUserMessagesFunc, sid64)
	}
	ui.windows.chatHistory[sid64].Show()
}

func (ui *Ui) createNameHistoryWindow(sid64 steamid.SID64) {
	_, found := ui.windows.nameHistory[sid64]
	if !found {
		ui.windows.nameHistory[sid64] = newUserNameWindow(ui.ctx, ui.application, ui.callBacks.queryNamesFunc, sid64)
	}
	ui.windows.nameHistory[sid64].Show()
}

func (ui *Ui) Start() {
	ui.windows.player.window.Show()
	ui.application.Run()
}

func showUserError(msg error, parent fyne.Window) {
	d := dialog.NewError(msg, parent)
	d.Show()
}

func validateUrl(urlString string) error {
	_, errParse := url.Parse(urlString)
	if errParse != nil {
		return errors.New(translations.One(translations.ErrorInvalidURL))
	}
	return nil
}

func validateName(name string) error {
	if len(name) == 0 {
		return errors.New(translations.One(translations.ErrorNameEmpty))
	}
	return nil
}

func validateSteamId(steamId string) error {
	if len(steamId) > 0 {
		_, err := steamid.StringToSID64(steamId)
		if err != nil {
			return errors.New(translations.One(translations.ErrorInvalidSteamId))
		}
	}
	return nil
}
func validateSteamRoot(newRoot string) error {
	if len(newRoot) > 0 {
		if !golib.Exists(newRoot) {
			return errors.New(translations.One(translations.ErrorInvalidPath))
		}
		fp := filepath.Join(newRoot, platform.TF2RootValidationFile)
		if !golib.Exists(fp) {
			return errors.New(translations.Tr(&i18n.Message{ID: string(translations.ErrorInvalidSteamRoot)},
				1, map[string]interface{}{"FileName": platform.TF2RootValidationFile}))
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
				return errors.Errorf("Duplicate tag found: %s", vt)
			}
		}
		validTags = append(validTags, normalized)
	}
	return nil
}
