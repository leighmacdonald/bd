package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	testLogger, _ := zap.NewDevelopment()
	settings, _ := detector.NewSettings()
	var testDb store.DataStore
	if false {
		// Note that we are not using :memory: due to the fact that our migration will close the connection
		dir, errDir := os.MkdirTemp("", "bd-test")
		if errDir != nil {
			panic(errDir)
		}
		localDbPath := filepath.Join(dir, "db.sqlite?cache=shared")
		testDb = store.New(localDbPath, testLogger)
		testLogger.Info("USing database", zap.String("path", localDbPath))
		defer func() {
			_ = testDb.Close()
			if errRemove := os.RemoveAll(dir); errRemove != nil {
				fmt.Print("Failed to remove temp db")
			}
		}()
	} else {
		testDb = store.New(":memory:", testLogger)
	}
	if errDb := testDb.Init(); errDb != nil {
		panic(errDb)
	}
	detector.Init(detector.Version{}, settings, testLogger, testDb, true)
	Init(detector.Logger(), true)
	os.Exit(m.Run())
}

func fetchIntoWithStatus(t *testing.T, method string, path string, status int, out any, body any) {
	var bodyReader io.Reader
	if body != nil {
		bodyJson, errEncode := json.Marshal(body)
		require.NoError(t, errEncode)
		bodyReader = bytes.NewReader(bodyJson)
	}
	req, _ := http.NewRequest(method, path, bodyReader)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if out != nil {
		responseData, errBody := io.ReadAll(w.Body)
		require.NoError(t, errBody)
		require.NoError(t, json.Unmarshal(responseData, out))
	}
	require.Equal(t, status, w.Code)
}

func TestGetPlayers(t *testing.T) {
	tp := createTestPlayers(5)
	for _, p := range tp {
		detector.AddPlayer(p)
	}
	var ps []store.Player
	fetchIntoWithStatus(t, "GET", "/players", http.StatusOK, &ps, nil)
	known := detector.Players()
	require.Equal(t, len(known), len(ps))
}

func TestGetSettingsHandler(t *testing.T) {
	t.Run("Get Settings", func(t *testing.T) {
		var wus webUserSettings
		fetchIntoWithStatus(t, "GET", "/settings", http.StatusOK, &wus, nil)
		s := webUserSettings{UserSettings: detector.Settings(), UniqueTags: rules.UniqueTags()}
		require.Equal(t, s.SteamID, wus.SteamID)
		require.Equal(t, s.SteamDir, wus.SteamDir)
		require.Equal(t, s.AutoLaunchGame, wus.AutoLaunchGame)
		require.Equal(t, s.AutoCloseOnGameExit, wus.AutoCloseOnGameExit)
		require.Equal(t, s.APIKey, wus.APIKey)
		require.Equal(t, s.DisconnectedTimeout, wus.DisconnectedTimeout)
		require.Equal(t, s.DiscordPresenceEnabled, wus.DiscordPresenceEnabled)
		require.Equal(t, s.KickerEnabled, wus.KickerEnabled)
		require.Equal(t, s.ChatWarningsEnabled, wus.ChatWarningsEnabled)
		require.Equal(t, s.PartyWarningsEnabled, wus.PartyWarningsEnabled)
		require.Equal(t, s.KickTags, wus.KickTags)
		require.Equal(t, s.VoiceBansEnabled, wus.VoiceBansEnabled)
		require.Equal(t, s.DebugLogEnabled, wus.DebugLogEnabled)
		require.Equal(t, s.Lists, wus.Lists)
		require.Equal(t, s.Links, wus.Links)
		require.Equal(t, s.RCONStatic, wus.RCONStatic)
		require.Equal(t, s.GUIEnabled, wus.GUIEnabled)
		require.Equal(t, s.HTTPEnabled, wus.HTTPEnabled)
		require.Equal(t, s.HTTPListenAddr, wus.HTTPListenAddr)
		require.Equal(t, s.PlayerExpiredTimeout, wus.PlayerExpiredTimeout)
		require.Equal(t, s.PlayerDisconnectTimeout, wus.PlayerDisconnectTimeout)
	})
	t.Run("Save Settings", func(t *testing.T) {
		s := detector.Settings()
		newSettings := *s
		newSettings.TF2Dir = "new/dir"
		fetchIntoWithStatus(t, "POST", "/settings", http.StatusNoContent, nil, newSettings)
		s2 := detector.Settings()
		require.Equal(t, newSettings.TF2Dir, s2.TF2Dir)
	})
}

func TestPostMarkPlayerHandler(t *testing.T) {
	pls := createTestPlayers(1)
	req := postMarkPlayerOpts{
		SteamIdOpt: SteamIdOpt{SteamID: pls[0].SteamIdString},
		Attrs:      []string{"cheater", "test"},
	}
	t.Run("Mark Player", func(t *testing.T) {
		fetchIntoWithStatus(t, "POST", "/mark", http.StatusNoContent, nil, req)
		matches := rules.MatchSteam(pls[0].SteamId)
		require.True(t, len(matches) > 0)
	})
	t.Run("Mark Duplicate Player", func(t *testing.T) {
		fetchIntoWithStatus(t, "POST", "/mark", http.StatusConflict, nil, req)
		matches := rules.MatchSteam(pls[0].SteamId)
		require.True(t, len(matches) > 0)
	})
	t.Run("Mark Without Attrs", func(t *testing.T) {
		fetchIntoWithStatus(t, "POST", "/mark", http.StatusBadRequest, nil, postMarkPlayerOpts{
			SteamIdOpt: SteamIdOpt{SteamID: pls[0].SteamIdString},
			Attrs:      []string{},
		})
		matches := rules.MatchSteam(pls[0].SteamId)
		require.True(t, len(matches) > 0)
	})
	t.Run("Mark bad steamid", func(t *testing.T) {
		fetchIntoWithStatus(t, "POST", "/mark", http.StatusBadRequest, nil, postMarkPlayerOpts{
			SteamIdOpt: SteamIdOpt{SteamID: "blah"},
			Attrs:      []string{"cheater", "test"},
		})
		matches := rules.MatchSteam(pls[0].SteamId)
		require.True(t, len(matches) > 0)
	})
}

func TestWhitelistPlayerHandler(t *testing.T) {
	pls := createTestPlayers(1)
	req := SteamIdOpt{
		SteamID: pls[0].SteamIdString,
	}
	require.NoError(t, detector.Mark(context.TODO(), pls[0].SteamId, []string{"test_mark"}))
	t.Run("Whitelist Player", func(t *testing.T) {
		fetchIntoWithStatus(t, "POST", "/whitelist", http.StatusNoContent, nil, req)
		plr, e := detector.GetPlayerOrCreate(context.Background(), pls[0].SteamId, false)
		require.NoError(t, e)
		require.True(t, plr.Whitelisted)
		require.Nil(t, rules.MatchSteam(pls[0].SteamId))
		require.True(t, rules.Whitelisted(pls[0].SteamId))
	})
	t.Run("Remove Player Whitelist", func(t *testing.T) {
		fetchIntoWithStatus(t, "DELETE", "/whitelist", http.StatusNoContent, nil, req)
		plr, e := detector.GetPlayerOrCreate(context.Background(), pls[0].SteamId, false)
		require.NoError(t, e)
		require.False(t, plr.Whitelisted)
		require.NotNil(t, rules.MatchSteam(pls[0].SteamId))
		require.False(t, rules.Whitelisted(pls[0].SteamId))
	})
}

func TestPlayerNotes(t *testing.T) {
	pls := createTestPlayers(1)
	req := postNotesOpts{
		SteamIdOpt: SteamIdOpt{SteamID: pls[0].SteamIdString},
		Note:       "New Note",
	}
	t.Run("Set Player", func(t *testing.T) {
		fetchIntoWithStatus(t, "POST", "/notes", http.StatusNoContent, nil, req)
		np, _ := detector.GetPlayerOrCreate(context.TODO(), pls[0].SteamId, false)
		require.Equal(t, req.Note, np.Notes)
	})
}

func TestPlayerChatHistory(t *testing.T) {
	pls := createTestPlayers(1)
	for i := 0; i < 10; i++ {
		require.NoError(t, detector.AddUserMessage(context.TODO(), pls[0], util.RandomString(i+1*2), false, true))
	}
	req := SteamIdOpt{
		SteamID: pls[0].SteamIdString,
	}
	t.Run("Get Chat History", func(t *testing.T) {
		var messages []*store.UserMessage
		fetchIntoWithStatus(t, "GET", fmt.Sprintf("/messages/%s", pls[0].SteamIdString), http.StatusOK, &messages, req)
		require.Equal(t, 10, len(messages))
	})
}

func TestPlayerNameHistory(t *testing.T) {
	pls := createTestPlayers(1)
	for i := 0; i < 5; i++ {
		require.NoError(t, detector.AddUserName(context.TODO(), pls[0], util.RandomString(i+1*2)))
	}
	req := SteamIdOpt{
		SteamID: pls[0].SteamIdString,
	}
	t.Run("Get Name History", func(t *testing.T) {
		var names store.UserMessageCollection
		fetchIntoWithStatus(t, "GET", fmt.Sprintf("/names/%s", pls[0].SteamIdString), http.StatusOK, &names, req)
		require.Equal(t, 5, len(names))
	})
}
