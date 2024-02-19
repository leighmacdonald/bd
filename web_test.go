package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/leighmacdonald/bd/store"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/stretchr/testify/require"
)

func testApp() (*Detector, func(), error) {
	tempDir, errTemp := os.MkdirTemp("", "bd-test")
	if errTemp != nil {
		return nil, func() {}, errors.Join(errTemp, errTempDir)
	}

	userSettings, _ := newSettings()
	userSettings.RunMode = ModeTest
	userSettings.SteamID = steamid.RandSID64()
	userSettings.ConfigPath = path.Join(tempDir, "bd.yaml")

	var dataStore store.DataStore

	if os.Getenv("WRITE_TEST_DB") != "" {
		// Toggle if you want to inspect the database
		localDBPath := filepath.Join(tempDir, "db.sqlite?cache=shared")
		dataStore = store.NewStore(localDBPath)
	} else {
		dataStore = store.NewStore(":memory:")
	}

	cleanup := func() {
		_ = dataStore.Close()
		_ = os.RemoveAll(tempDir)
	}

	if errMigrate := dataStore.Init(); errMigrate != nil && !errors.Is(errMigrate, migrate.ErrNoChange) {
		return nil, cleanup, errors.Join(errMigrate, errPerformMigration)
	}

	logChan := make(chan string)

	logReader, errLogReader := newLogReader(filepath.Join(userSettings.TF2Dir, "console.log"), logChan, false)
	if errLogReader != nil {
		return nil, cleanup, errLogReader
	}

	versionInfo := Version{Version: "", Commit: "", Date: "", BuiltBy: ""}
	ds, _ := createAPIDataSource("")
	application := NewDetector(userSettings, dataStore, versionInfo, &NopCache{}, logReader, logChan, ds)

	return application, cleanup, nil
}

func fetchIntoWithStatus(t *testing.T, app *Detector, method string, path string, status int, out any, body any) {
	t.Helper()

	var bodyReader io.Reader

	if body != nil {
		bodyJSON, errEncode := json.Marshal(body)
		require.NoError(t, errEncode)

		bodyReader = bytes.NewReader(bodyJSON)
	}

	req, _ := http.NewRequestWithContext(context.Background(), method, path, bodyReader)
	recorder := httptest.NewRecorder()

	app.Web.Handler.ServeHTTP(recorder, req)

	if out != nil {
		responseData, errBody := io.ReadAll(recorder.Body)
		require.NoError(t, errBody)
		require.NoError(t, json.Unmarshal(responseData, out))
	}

	require.Equal(t, status, recorder.Code)
}

func TestGetPlayers(t *testing.T) {
	app, cleanup, errApp := testApp()
	require.NoError(t, errApp, "Failed to create test app")

	defer cleanup()

	CreateTestPlayers(app, 5)

	var state CurrentState

	fetchIntoWithStatus(t, app, http.MethodGet, "/state", http.StatusOK, &state, nil)

	require.Equal(t, len(app.players.all()), len(state.Players))
}

func TestGetSettingsHandler(t *testing.T) { //nolint:tparallel
	app, cleanup, errApp := testApp()
	require.NoError(t, errApp, "Failed to create test app")

	defer cleanup()

	t.Run("Get Settings", func(t *testing.T) { //nolint:tparallel
		var wus WebUserSettings

		fetchIntoWithStatus(t, app, "GET", "/settings", http.StatusOK, &wus, nil)
		userSettings := WebUserSettings{UserSettings: app.Settings(), UniqueTags: app.Rules().UniqueTags()}
		require.Equal(t, userSettings.SteamID, wus.SteamID)
		require.Equal(t, userSettings.SteamDir, wus.SteamDir)
		require.Equal(t, userSettings.AutoLaunchGame, wus.AutoLaunchGame)
		require.Equal(t, userSettings.AutoCloseOnGameExit, wus.AutoCloseOnGameExit)
		require.Equal(t, userSettings.APIKey, wus.APIKey)
		require.Equal(t, userSettings.DisconnectedTimeout, wus.DisconnectedTimeout)
		require.Equal(t, userSettings.DiscordPresenceEnabled, wus.DiscordPresenceEnabled)
		require.Equal(t, userSettings.KickerEnabled, wus.KickerEnabled)
		require.Equal(t, userSettings.ChatWarningsEnabled, wus.ChatWarningsEnabled)
		require.Equal(t, userSettings.PartyWarningsEnabled, wus.PartyWarningsEnabled)
		require.Equal(t, userSettings.KickTags, wus.KickTags)
		require.Equal(t, userSettings.VoiceBansEnabled, wus.VoiceBansEnabled)
		require.Equal(t, userSettings.DebugLogEnabled, wus.DebugLogEnabled)
		require.Equal(t, userSettings.Lists, wus.Lists)
		require.Equal(t, userSettings.Links, wus.Links)
		require.Equal(t, userSettings.RCONStatic, wus.RCONStatic)
		require.Equal(t, userSettings.HTTPEnabled, wus.HTTPEnabled)
		require.Equal(t, userSettings.HTTPListenAddr, wus.HTTPListenAddr)
		require.Equal(t, userSettings.PlayerExpiredTimeout, wus.PlayerExpiredTimeout)
		require.Equal(t, userSettings.PlayerDisconnectTimeout, wus.PlayerDisconnectTimeout)
	})

	t.Run("Save Settings", func(t *testing.T) { //nolint:tparallel
		newSettings := app.Settings()
		newSettings.TF2Dir = "new/dir"
		newSettings.SteamID = steamid.RandSID64()
		newSettings.BdAPIEnabled = true
		newSettings.BdAPIAddress = APIDataSourceDefaultAddress

		require.NoError(t, app.SaveSettings(newSettings))
		fetchIntoWithStatus(t, app, http.MethodPut, "/settings", http.StatusNoContent, nil, newSettings)

		updated := app.Settings()

		require.EqualValues(t, newSettings, updated)
	})
}

func TestPostMarkPlayerHandler(t *testing.T) { //nolint:tparallel
	app, cleanup, errApp := testApp()
	require.NoError(t, errApp, "Failed to create test app")

	defer cleanup()

	CreateTestPlayers(app, 1)

	req := PostMarkPlayerOpts{
		Attrs: []string{"cheater", "test"},
	}

	players := app.players.all()

	t.Run("Mark Player", func(t *testing.T) { //nolint:tparallel
		fetchIntoWithStatus(t, app, "POST", fmt.Sprintf("/mark/%s", players[0].SteamID), http.StatusNoContent, nil, req)

		matches := app.Rules().MatchSteam(players[0].SteamID)

		require.True(t, len(matches) > 0)
	})

	t.Run("Mark Duplicate Player", func(t *testing.T) { //nolint:tparallel
		fetchIntoWithStatus(t, app, "POST", fmt.Sprintf("/mark/%s", players[0].SteamID), http.StatusConflict, nil, req)

		matches := app.Rules().MatchSteam(players[0].SteamID)

		require.True(t, len(matches) > 0)
	})

	t.Run("Mark Without Attrs", func(t *testing.T) { //nolint:tparallel
		fetchIntoWithStatus(t, app, "POST", fmt.Sprintf("/mark/%s", players[0].SteamID), http.StatusBadRequest, nil, PostMarkPlayerOpts{
			Attrs: []string{},
		})

		matches := app.Rules().MatchSteam(players[0].SteamID)

		require.True(t, len(matches) > 0)
	})

	t.Run("Mark bad steamid", func(t *testing.T) { //nolint:tparallel
		fetchIntoWithStatus(t, app, "POST", "/mark/blah", http.StatusBadRequest, nil, PostMarkPlayerOpts{
			Attrs: []string{"cheater", "test"},
		})

		matches := app.Rules().MatchSteam(players[0].SteamID)

		require.True(t, len(matches) > 0)
	})
}

func TestUnmarkPlayerHandler(t *testing.T) {
	app, cleanup, errApp := testApp()
	require.NoError(t, errApp, "Failed to create test app")

	defer cleanup()

	CreateTestPlayers(app, 1)

	markedPlayer := app.players.all()[0]

	testAttrs := []string{"cheater"}
	require.NoError(t, app.mark(context.Background(), markedPlayer.SteamID, testAttrs))

	t.Run("Unmark Non-Marked Player", func(t *testing.T) {
		fetchIntoWithStatus(t, app, http.MethodDelete,
			fmt.Sprintf("/mark/%d", steamid.RandSID64().Int64()), http.StatusNotFound, nil, nil)
	})

	t.Run("Unmark Marked Player", func(t *testing.T) {
		var resp UnmarkResponse

		fetchIntoWithStatus(t, app, http.MethodDelete,
			fmt.Sprintf("/mark/%d", markedPlayer.SteamID.Int64()), http.StatusOK, &resp, nil)
		require.Equal(t, 0, resp.Remaining)
	})
}

func TestWhitelistPlayerHandler(t *testing.T) { //nolint:tparallel
	app, cleanup, errApp := testApp()
	require.NoError(t, errApp, "Failed to create test app")

	defer cleanup()

	CreateTestPlayers(app, 1)

	players := app.players.all()

	require.NoError(t, app.mark(context.TODO(), players[0].SteamID, []string{"test_mark"}))

	t.Run("Whitelist Player", func(t *testing.T) { //nolint:tparallel
		fetchIntoWithStatus(t, app, "POST", fmt.Sprintf("/whitelist/%s", players[0].SteamID), http.StatusNoContent, nil, nil)
		player, errPlayer := app.GetPlayerOrCreate(context.Background(), players[0].SteamID)
		require.NoError(t, errPlayer)
		require.True(t, player.Whitelisted)
		require.Nil(t, app.Rules().MatchSteam(players[0].SteamID))
		require.True(t, app.Rules().Whitelisted(players[0].SteamID))
	})

	t.Run("Remove Player Whitelist", func(t *testing.T) { //nolint:tparallel
		fetchIntoWithStatus(t, app, "DELETE", fmt.Sprintf("/whitelist/%s", players[0].SteamID), http.StatusNoContent, nil, nil)
		player, errPlayer := app.GetPlayerOrCreate(context.Background(), players[0].SteamID)
		require.NoError(t, errPlayer)
		require.False(t, player.Whitelisted)
		require.NotNil(t, app.Rules().MatchSteam(players[0].SteamID))
		require.False(t, app.Rules().Whitelisted(players[0].SteamID))
	})
}

func TestPlayerNotes(t *testing.T) { //nolint:tparallel
	app, cleanup, errApp := testApp()
	require.NoError(t, errApp, "Failed to create test app")

	defer cleanup()

	CreateTestPlayers(app, 1)

	players := app.players.all()

	req := PostNotesOpts{
		Note: "New Note",
	}

	t.Run("Set Player", func(t *testing.T) {
		fetchIntoWithStatus(t, app, "POST", fmt.Sprintf("/notes/%s", players[0].SteamID), http.StatusNoContent, nil, req)
		player, errPlayer := app.GetPlayerOrCreate(context.TODO(), players[0].SteamID)
		require.NoError(t, errPlayer)
		require.Equal(t, req.Note, player.Notes)
	})
}

func TestPlayerChatHistory(t *testing.T) {
	app, cleanup, errApp := testApp()
	require.NoError(t, errApp, "Failed to create test app")

	defer cleanup()

	CreateTestPlayers(app, 1)

	players := app.players.all()

	for i := 0; i < 10; i++ {
		require.NoError(t, app.addUserMessage(context.TODO(), &players[0], RandomString(i+1*2), false, true))
	}

	t.Run("Get Chat History", func(t *testing.T) { //nolint:tparallel
		var messages []*UserMessage

		fetchIntoWithStatus(t, app, "GET", fmt.Sprintf("/messages/%s", players[0].SteamID), http.StatusOK, &messages, nil)
		require.Equal(t, 10, len(messages))
	})
}

func TestPlayerNameHistory(t *testing.T) { //nolint:tparallel
	app, cleanup, errApp := testApp()
	require.NoError(t, errApp, "Failed to create test app")

	defer cleanup()

	CreateTestPlayers(app, 2)

	players := app.players.all()

	for i := 0; i < 5; i++ {
		players[1].Name = RandomString(i + 1*2)
		require.NoError(t, app.addUserName(context.TODO(), &players[1]))
	}

	t.Run("Get Name History", func(t *testing.T) {
		var names UserMessageCollection

		fetchIntoWithStatus(t, app, "GET", fmt.Sprintf("/names/%s", players[1].SteamID), http.StatusOK, &names, nil)

		require.Equal(t, 5, len(names))
	})
}
