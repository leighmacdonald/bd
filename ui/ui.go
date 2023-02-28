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
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"log"
	"sync"
)

const (
	AppId   = "com.github.leighmacdonald.bd"
	urlHome = "https://github.com/leighmacdonald/bd"
	urlHelp = "https://github.com/leighmacdonald/bd/wiki"
)

type UserInterface interface {
	Refresh()
	Start()
	SetBuildInfo(version string, commit string, date string, builtBy string)
	SetOnLaunchTF2(func())
	SetOnKick(kickFunc model.KickFunc)
	UpdateServerState(state model.Server)
	UpdatePlayerState(collection model.PlayerCollection)
	AddUserMessage(message model.UserMessage)
	UpdateAttributes([]string)
	SetAvatar(sid64 steamid.SID64, avatar []byte)
}

type windows struct {
	player      *playerWindow
	chat        *gameChatWindow
	chatHistory map[steamid.SID64]*userChatWindow
	nameHistory map[steamid.SID64]fyne.Window
}

type MenuCreator func(window fyne.Window, steamId steamid.SID64, userId int64) *fyne.Menu

type Ui struct {
	ctx           context.Context
	application   fyne.App
	boundSettings boundSettings
	settings      *model.Settings
	//players               model.PlayerCollection
	windows *windows

	knownAttributes       binding.StringList
	gameLauncherFunc      func()
	markFn                model.MarkFunc
	kickFunc              model.KickFunc
	queryNamesFunc        model.QueryNamesFunc
	queryUserMessagesFunc model.QueryUserMessagesFunc
	userAvatarMu          *sync.RWMutex

	userAvatar map[steamid.SID64]fyne.Resource
}

func (ui *Ui) UpdateServerState(state model.Server) {
	ui.windows.player.UpdateServerState(state)
}

func (ui *Ui) UpdatePlayerState(collection model.PlayerCollection) {
	ui.windows.player.updatePlayerState(collection)
}

func defaultApp() fyne.App {
	application := app.NewWithID(AppId)
	application.Settings().SetTheme(&bdTheme{})
	application.SetIcon(resourceIconPng)
	return application
}

func New(ctx context.Context, settings *model.Settings, markFunc model.MarkFunc, namesFunc model.QueryNamesFunc,
	messagesFunc model.QueryUserMessagesFunc, gameLaunchFunc func(), kickFunc model.KickFunc) UserInterface {
	ui := Ui{
		ctx:             ctx,
		application:     defaultApp(),
		boundSettings:   boundSettings{binding.BindStruct(settings)},
		settings:        settings,
		knownAttributes: binding.NewStringList(),
		windows: &windows{
			chatHistory: map[steamid.SID64]*userChatWindow{},
			nameHistory: map[steamid.SID64]fyne.Window{},
		},
		userAvatarMu:          &sync.RWMutex{},
		userAvatar:            make(map[steamid.SID64]fyne.Resource),
		queryNamesFunc:        namesFunc,
		queryUserMessagesFunc: messagesFunc,
		kickFunc:              kickFunc,
		gameLauncherFunc:      gameLaunchFunc,
	}

	ui.windows.chat = newGameChatWindow(ui.ctx, ui.application, ui.kickFunc, ui.knownAttributes, markFunc, settings, func(sid64 steamid.SID64) {
		ui.createChatHistoryWindow(sid64)
	}, func(sid64 steamid.SID64) {
		ui.createNameHistoryWindow(sid64)
	})

	ui.windows.player = newPlayerWindow(
		ui.application,
		settings,
		ui.boundSettings, func() {
			ui.windows.chat.window.Show()
		},
		ui.gameLauncherFunc,
		func(window fyne.Window, steamId steamid.SID64, userId int64) *fyne.Menu {
			return generateUserMenu(ui.ctx, ui.application, window, steamId, userId, ui.kickFunc, ui.knownAttributes, ui.markFn,
				ui.settings.Links, ui.createChatHistoryWindow, ui.createNameHistoryWindow)
		})

	return &ui
}

func (ui *Ui) SetAvatar(sid64 steamid.SID64, data []byte) {
	if !sid64.Valid() || data == nil {
		return
	}
	ui.userAvatarMu.Lock()
	ui.userAvatar[sid64] = fyne.NewStaticResource(sid64.String(), data)
	ui.userAvatarMu.Unlock()
}

func (ui *Ui) GetAvatar(sid64 steamid.SID64) fyne.Resource {
	ui.userAvatarMu.RLock()
	defer ui.userAvatarMu.RUnlock()
	av, found := ui.userAvatar[sid64]
	if found {
		return av
	}
	return resourceDefaultavatarJpg
}

func (ui *Ui) SetBuildInfo(version string, commit string, date string, builtBy string) {
	ui.windows.player.aboutDialog.SetBuildInfo(version, commit, date, builtBy)
}

func (ui *Ui) SetFetchMessageHistory(messagesFunc model.QueryUserMessagesFunc) {
	ui.queryUserMessagesFunc = messagesFunc
}

func (ui *Ui) SetFetchNameHistory(namesFunc model.QueryNamesFunc) {
	ui.queryNamesFunc = namesFunc
}

func (ui *Ui) SetOnMark(fn model.MarkFunc) {
	ui.markFn = fn
}

func (ui *Ui) SetOnKick(fn model.KickFunc) {
	ui.kickFunc = fn
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
	//ui.chatWindow.window.Content().Refresh()

	if userChat, found := ui.windows.chatHistory[msg.PlayerSID]; found {
		if errAppend := userChat.boundList.Append(msg); errAppend != nil {
			log.Printf("Failed to append user history message: %v", errAppend)
		}
		userChat.Content().Refresh()
	}
}

func (ui *Ui) createChatHistoryWindow(sid64 steamid.SID64) {
	_, found := ui.windows.chatHistory[sid64]
	if found {
		ui.windows.chatHistory[sid64].Show()
	} else {
		ui.windows.chatHistory[sid64] = newUserChatWindow(ui.ctx, ui.application, ui.queryUserMessagesFunc, sid64)
	}
}

func (ui *Ui) createNameHistoryWindow(sid64 steamid.SID64) {
	_, found := ui.windows.nameHistory[sid64]
	if found {
		ui.windows.nameHistory[sid64].Show()
	} else {
		ui.windows.nameHistory[sid64] = newUserNameWindow(ui.ctx, ui.application, ui.queryNamesFunc, sid64)
	}
}

func (ui *Ui) SetOnLaunchTF2(fn func()) {
	ui.gameLauncherFunc = fn
}

func (ui *Ui) Start() {
	ui.windows.player.window.Show()
	ui.application.Run()
}

func (ui *Ui) OnDisconnect(sid64 steamid.SID64) {
	log.Printf("Player disconnected: %d", sid64.Int64())
}

func (ui *Ui) Run() {
	ui.windows.player.window.Show()
	ui.application.Run()
}

func showUserError(msg string, parent fyne.Window) {
	d := dialog.NewError(errors.New(msg), parent)
	d.Show()
}
