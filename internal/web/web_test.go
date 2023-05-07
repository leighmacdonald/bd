package web

import (
	"bytes"
	"encoding/json"
	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	detector.Setup(detector.Version{}, true)
	Setup(detector.Logger(), true)
	retCode := m.Run()
	os.Exit(retCode)
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
}

func TestPostSettingsHandler(t *testing.T) {
	s := detector.Settings()
	newSettings := *s
	newSettings.TF2Dir = "new/dir"
	fetchIntoWithStatus(t, "POST", "/settings", http.StatusNoContent, nil, newSettings)
	s2 := detector.Settings()
	require.Equal(t, newSettings.TF2Dir, s2.TF2Dir)
}

func TestPostMarkPlayerHandler(t *testing.T) {
	pls := createTestPlayers(1)
	p := pls[0]
	req := postMarkPlayerOpts{
		SteamID: p.SteamIdString,
		Attrs:   []string{"cheater", "test"},
	}
	fetchIntoWithStatus(t, "POST", "/mark", http.StatusNoContent, nil, req)
	matches := rules.MatchSteam(p.SteamId)
	require.True(t, len(matches) > 0)
}
