package detector_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testApp() (*detector.Detector, func(), error) {
	tempDir, errTemp := os.MkdirTemp("", "bd-test")
	if errTemp != nil {
		return nil, func() {}, errors.Wrap(errTemp, "Failed to create temp dir")
	}

	logger := zap.NewNop()
	userSettings, _ := detector.NewSettings()
	userSettings.RunMode = detector.ModeTest
	userSettings.SteamID = steamid.RandSID64()
	userSettings.ConfigPath = path.Join(tempDir, "bd.yaml")

	var dataStore store.DataStore

	if os.Getenv("WRITE_TEST_DB") != "" {
		// Toggle if you want to inspect the database
		localDBPath := filepath.Join(tempDir, "db.sqlite?cache=shared")
		dataStore = store.New(localDBPath, logger)
	} else {
		dataStore = store.New(":memory:", logger)
	}

	cleanup := func() {
		_ = dataStore.Close()
		_ = os.RemoveAll(tempDir)
	}

	if errMigrate := dataStore.Init(); errMigrate != nil && !errors.Is(errMigrate, migrate.ErrNoChange) {
		return nil, cleanup, errors.Wrap(errMigrate, "Failed to create test app")
	}

	logChan := make(chan string)

	logReader, errLogReader := detector.NewLogReader(logger, filepath.Join(userSettings.TF2Dir, "console.log"), logChan)
	if errLogReader != nil {
		return nil, cleanup, errors.Wrap(errLogReader, "Failed to create test app")
	}

	versionInfo := detector.Version{Version: "", Commit: "", Date: "", BuiltBy: ""}
	ds, _ := detector.NewAPIDataSource("")
	application := detector.New(logger, userSettings, dataStore, versionInfo, &detector.NopCache{}, logReader, logChan, ds)

	return application, cleanup, nil
}

func fetchIntoWithStatus(t *testing.T, app *detector.Detector, method string, path string, status int, out any, body any) {
	t.Helper()

	var bodyReader io.Reader

	if body != nil {
		bodyJSON, errEncode := json.Marshal(body)
		require.NoError(t, errEncode)

		bodyReader = bytes.NewReader(bodyJSON)
	}

	req, _ := http.NewRequestWithContext(context.Background(), method, path, bodyReader)
	recorder := httptest.NewRecorder()

	app.Web.Engine.ServeHTTP(recorder, req)

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

	testPlayers := detector.CreateTestPlayers(app, 5)

	var state detector.CurrentState

	fetchIntoWithStatus(t, app, http.MethodGet, "/state", http.StatusOK, &state, nil)

	require.Equal(t, len(testPlayers), len(state.Players))
}

func TestGetSettingsHandler(t *testing.T) { //nolint:tparallel
	app, cleanup, errApp := testApp()
	require.NoError(t, errApp, "Failed to create test app")

	defer cleanup()

	t.Run("Get Settings", func(t *testing.T) { //nolint:tparallel
		var wus detector.WebUserSettings
		fetchIntoWithStatus(t, app, "GET", "/settings", http.StatusOK, &wus, nil)
		settings := detector.WebUserSettings{UserSettings: app.Settings(), UniqueTags: app.Rules().UniqueTags()}
		require.Equal(t, settings.SteamID, wus.SteamID)
		require.Equal(t, settings.SteamDir, wus.SteamDir)
		require.Equal(t, settings.AutoLaunchGame, wus.AutoLaunchGame)
		require.Equal(t, settings.AutoCloseOnGameExit, wus.AutoCloseOnGameExit)
		require.Equal(t, settings.APIKey, wus.APIKey)
		require.Equal(t, settings.DisconnectedTimeout, wus.DisconnectedTimeout)
		require.Equal(t, settings.DiscordPresenceEnabled, wus.DiscordPresenceEnabled)
		require.Equal(t, settings.KickerEnabled, wus.KickerEnabled)
		require.Equal(t, settings.ChatWarningsEnabled, wus.ChatWarningsEnabled)
		require.Equal(t, settings.PartyWarningsEnabled, wus.PartyWarningsEnabled)
		require.Equal(t, settings.KickTags, wus.KickTags)
		require.Equal(t, settings.VoiceBansEnabled, wus.VoiceBansEnabled)
		require.Equal(t, settings.DebugLogEnabled, wus.DebugLogEnabled)
		require.Equal(t, settings.Lists, wus.Lists)
		require.Equal(t, settings.Links, wus.Links)
		require.Equal(t, settings.RCONStatic, wus.RCONStatic)
		require.Equal(t, settings.HTTPEnabled, wus.HTTPEnabled)
		require.Equal(t, settings.HTTPListenAddr, wus.HTTPListenAddr)
		require.Equal(t, settings.PlayerExpiredTimeout, wus.PlayerExpiredTimeout)
		require.Equal(t, settings.PlayerDisconnectTimeout, wus.PlayerDisconnectTimeout)
	})

	t.Run("Save Settings", func(t *testing.T) { //nolint:tparallel
		newSettings := app.Settings()
		newSettings.TF2Dir = "new/dir"
		newSettings.SteamID = steamid.RandSID64()

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

	pls := detector.CreateTestPlayers(app, 1)
	req := detector.PostMarkPlayerOpts{
		Attrs: []string{"cheater", "test"},
	}

	t.Run("Mark Player", func(t *testing.T) { //nolint:tparallel
		fetchIntoWithStatus(t, app, "POST", fmt.Sprintf("/mark/%s", pls[0].SteamID), http.StatusNoContent, nil, req)
		matches := app.Rules().MatchSteam(pls[0].SteamID)
		require.True(t, len(matches) > 0)
	})

	t.Run("Mark Duplicate Player", func(t *testing.T) { //nolint:tparallel
		fetchIntoWithStatus(t, app, "POST", fmt.Sprintf("/mark/%s", pls[0].SteamID), http.StatusConflict, nil, req)
		matches := app.Rules().MatchSteam(pls[0].SteamID)
		require.True(t, len(matches) > 0)
	})

	t.Run("Mark Without Attrs", func(t *testing.T) { //nolint:tparallel
		fetchIntoWithStatus(t, app, "POST", fmt.Sprintf("/mark/%s", pls[0].SteamID), http.StatusBadRequest, nil, detector.PostMarkPlayerOpts{
			Attrs: []string{},
		})
		matches := app.Rules().MatchSteam(pls[0].SteamID)
		require.True(t, len(matches) > 0)
	})

	t.Run("Mark bad steamid", func(t *testing.T) { //nolint:tparallel
		fetchIntoWithStatus(t, app, "POST", "/mark/blah", http.StatusBadRequest, nil, detector.PostMarkPlayerOpts{
			Attrs: []string{"cheater", "test"},
		})
		matches := app.Rules().MatchSteam(pls[0].SteamID)
		require.True(t, len(matches) > 0)
	})
}

func TestUnmarkPlayerHandler(t *testing.T) {
	app, cleanup, errApp := testApp()
	require.NoError(t, errApp, "Failed to create test app")

	defer cleanup()

	markedPlayer := detector.CreateTestPlayers(app, 1)[0]

	testAttrs := []string{"cheater"}
	require.NoError(t, app.Mark(context.Background(), markedPlayer.SteamID, testAttrs))

	t.Run("Unmark Non-Marked Player", func(t *testing.T) {
		fetchIntoWithStatus(t, app, http.MethodDelete,
			fmt.Sprintf("/mark/%d", steamid.RandSID64().Int64()), http.StatusNotFound, nil, nil)
	})

	t.Run("Unmark Marked Player", func(t *testing.T) {
		var resp detector.UnmarkResponse
		fetchIntoWithStatus(t, app, http.MethodDelete,
			fmt.Sprintf("/mark/%d", markedPlayer.SteamID.Int64()), http.StatusOK, &resp, nil)
		require.Equal(t, 0, resp.Remaining)
	})
}

func TestWhitelistPlayerHandler(t *testing.T) { //nolint:tparallel
	app, cleanup, errApp := testApp()
	require.NoError(t, errApp, "Failed to create test app")

	defer cleanup()

	pls := detector.CreateTestPlayers(app, 1)

	require.NoError(t, app.Mark(context.TODO(), pls[0].SteamID, []string{"test_mark"}))

	t.Run("Whitelist Player", func(t *testing.T) { //nolint:tparallel
		fetchIntoWithStatus(t, app, "POST", fmt.Sprintf("/whitelist/%s", pls[0].SteamID), http.StatusNoContent, nil, nil)
		plr, e := app.GetPlayerOrCreate(context.Background(), pls[0].SteamID)
		require.NoError(t, e)
		require.True(t, plr.Whitelisted)
		require.Nil(t, app.Rules().MatchSteam(pls[0].SteamID))
		require.True(t, app.Rules().Whitelisted(pls[0].SteamID))
	})

	t.Run("Remove Player Whitelist", func(t *testing.T) { //nolint:tparallel
		fetchIntoWithStatus(t, app, "DELETE", fmt.Sprintf("/whitelist/%s", pls[0].SteamID), http.StatusNoContent, nil, nil)
		plr, e := app.GetPlayerOrCreate(context.Background(), pls[0].SteamID)
		require.NoError(t, e)
		require.False(t, plr.Whitelisted)
		require.NotNil(t, app.Rules().MatchSteam(pls[0].SteamID))
		require.False(t, app.Rules().Whitelisted(pls[0].SteamID))
	})
}

func TestPlayerNotes(t *testing.T) { //nolint:tparallel
	app, cleanup, errApp := testApp()
	require.NoError(t, errApp, "Failed to create test app")

	defer cleanup()

	pls := detector.CreateTestPlayers(app, 1)
	req := detector.PostNotesOpts{
		Note: "New Note",
	}

	t.Run("Set Player", func(t *testing.T) { //nolint:tparallel
		fetchIntoWithStatus(t, app, "POST", fmt.Sprintf("/notes/%s", pls[0].SteamID), http.StatusNoContent, nil, req)
		np, _ := app.GetPlayerOrCreate(context.TODO(), pls[0].SteamID)
		require.Equal(t, req.Note, np.Notes)
	})
}

func TestPlayerChatHistory(t *testing.T) { //nolint:tparallel
	app, cleanup, errApp := testApp()
	require.NoError(t, errApp, "Failed to create test app")

	defer cleanup()

	pls := detector.CreateTestPlayers(app, 1)

	for i := 0; i < 10; i++ {
		require.NoError(t, app.AddUserMessage(context.TODO(), pls[0], util.RandomString(i+1*2), false, true))
	}

	t.Run("Get Chat History", func(t *testing.T) { //nolint:tparallel
		var messages []*store.UserMessage
		fetchIntoWithStatus(t, app, "GET", fmt.Sprintf("/messages/%s", pls[0].SteamID), http.StatusOK, &messages, nil)
		require.Equal(t, 10, len(messages))
	})
}

func TestPlayerNameHistory(t *testing.T) { //nolint:tparallel
	app, cleanup, errApp := testApp()
	require.NoError(t, errApp, "Failed to create test app")

	defer cleanup()

	pls := detector.CreateTestPlayers(app, 2)

	for i := 0; i < 5; i++ {
		require.NoError(t, app.AddUserName(context.TODO(), pls[1], util.RandomString(i+1*2)))
	}

	t.Run("Get Name History", func(t *testing.T) {
		var names store.UserMessageCollection
		fetchIntoWithStatus(t, app, "GET", fmt.Sprintf("/names/%s", pls[1].SteamID), http.StatusOK, &names, nil)
		require.Equal(t, 5, len(names))
	})
}
