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
	"github.com/leighmacdonald/bd/internal/platform"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/internal/tr"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"github.com/puzpuzpuz/xsync/v2"
	"go.uber.org/zap"
	"net/url"
	"path/filepath"
	"strings"
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

var (
	application     fyne.App
	windows         *windowMap
	knownAttributes binding.StringList
	avatarCache     *xsync.MapOf[string, fyne.Resource]
	version         detector.Version
	logger          *zap.Logger
)

func defaultApp() fyne.App {
	newApplication := app.NewWithID(AppId)
	newApplication.Settings().SetTheme(&bdTheme{})
	newApplication.SetIcon(resourceIconPng)
	return newApplication
}

type windowMap struct {
	player      *playerWindow
	chat        *gameChatWindow
	search      *searchWindow
	chatHistory map[steamid.SID64]*userChatWindow
	nameHistory map[steamid.SID64]*userNameWindow
}

type MenuCreator func(window fyne.Window, steamId steamid.SID64, userId int64) *fyne.Menu

func UpdateServerState(state detector.Server) {
	windows.player.UpdateServerState(state)
}

func UpdatePlayerState(collection store.PlayerCollection) {
	windows.player.updatePlayerState(collection)
}

func Init(ctx context.Context, rootLogger *zap.Logger, versionInfo detector.Version) {
	logger = rootLogger.Named("bd.gui")
	version = versionInfo
	knownAttributes = binding.NewStringList()
	windows = &windowMap{
		chatHistory: map[steamid.SID64]*userChatWindow{},
		nameHistory: map[steamid.SID64]*userNameWindow{},
	}
	avatarCache = xsync.NewMapOf[fyne.Resource]()

	application = defaultApp()
	windows.chat = newGameChatWindow(ctx)
	windows.search = newSearchWindow(ctx)
	windows.player = newPlayerWindow(
		logger,
		func(window fyne.Window, steamId steamid.SID64, userId int64) *fyne.Menu {
			return generateUserMenu(ctx, window, steamId, userId)
		}, version)

}

func Refresh() {
	windows.chat.Content().Refresh()
	if windows.player != nil {
		windows.player.content.Refresh()
	}
}

func UpdateAttributes(attrs []string) {
	if err := knownAttributes.Set(attrs); err != nil {
		logger.Error("Failed to update known attribute", zap.Error(err))
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

func AddUserMessage(msg store.UserMessage) {
	if errAppend := windows.chat.append(msg); errAppend != nil {
		logger.Error("Failed to append game message", zap.Error(errAppend))
	}
	if userChat, found := windows.chatHistory[msg.SteamId]; found {
		if errAppend := userChat.boundList.Append(msg); errAppend != nil {
			logger.Error("Failed to append user history message", zap.Error(errAppend))
		}
		userChat.Content().Refresh()
	}
}

func createChatHistoryWindow(ctx context.Context, sid64 steamid.SID64) {
	_, found := windows.chatHistory[sid64]
	if !found {
		windows.chatHistory[sid64] = newUserChatWindow(ctx, detector.Store().FetchMessages, sid64)
	}
	windows.chatHistory[sid64].Show()
}

func createNameHistoryWindow(ctx context.Context, sid64 steamid.SID64) {
	_, found := windows.nameHistory[sid64]
	if !found {
		windows.nameHistory[sid64] = newUserNameWindow(ctx, detector.Store().FetchNames, sid64)
	}
	windows.nameHistory[sid64].Show()
}

func Start(ctx context.Context) {
	windows.player.window.Show()
	application.Run()
	ctx.Done()
}

func Quit() {
	application.Quit()
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
