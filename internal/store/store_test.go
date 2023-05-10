package store

import (
	"context"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"os"
	"testing"
)

var testDb DataStore

func TestMain(m *testing.M) {
	logger, _ := zap.NewDevelopment()
	testDb = New(":memory:", logger)
	if errInit := testDb.Init(); errInit != nil {
		logger.Fatal("failed to setup database", zap.Error(errInit))
	}
	os.Exit(m.Run())
}

func TestPlayer(t *testing.T) {
	player1 := NewPlayer(steamid.SID64(76561197961279983), util.RandomString(10))
	const msgCount = 10
	t.Run("Create New Player", func(t *testing.T) {
		require.NoError(t, testDb.GetPlayer(context.TODO(), player1.SteamId, true, player1), "Failed to create player")
	})
	t.Run("Fetch Existing Player", func(t *testing.T) {
		require.NoError(t, testDb.GetPlayer(context.TODO(), player1.SteamId, false, player1), "Failed to create player")
	})
	t.Run("Add User Messages", func(t *testing.T) {
		for i := 0; i < msgCount; i++ {
			msg, errMsg := NewUserMessage(player1.SteamId, util.RandomString(10), false, false)
			require.NoError(t, errMsg)
			require.NoError(t, testDb.SaveMessage(context.TODO(), msg))
		}
	})
	t.Run("Fetch User Messages", func(t *testing.T) {
		msgs, errMsgs := testDb.FetchMessages(context.TODO(), player1.SteamId)
		require.NoError(t, errMsgs)
		require.Equal(t, msgCount, len(msgs))
	})
	t.Run("Add User Names", func(t *testing.T) {
		for i := 0; i < msgCount; i++ {
			msg, errMsg := NewUserNameHistory(player1.SteamId, util.RandomString(10))
			require.NoError(t, errMsg)
			require.NoError(t, testDb.SaveUserNameHistory(context.TODO(), msg))
		}
	})
	t.Run("Fetch User Names", func(t *testing.T) {
		names, errMsgs := testDb.FetchNames(context.TODO(), player1.SteamId)
		require.NoError(t, errMsgs)
		require.Equal(t, msgCount+1, len(names))
	})
	t.Run("Search Players", func(t *testing.T) {
		knownIds := steamid.Collection{
			76561197998365611, 76561197977133523, 76561198065825165, 76561198004429398, 76561198182505218,
		}
		knownNames := []string{"test name 1", "test name 2", "Blah Blah", "bob", "sally"}
		for idx, sid := range knownIds {
			player := NewPlayer(sid, knownNames[idx])
			require.NoError(t, testDb.GetPlayer(context.TODO(), player.SteamId, true, player), "Failed to create player")
		}
		t.Run("By SteamID", func(t *testing.T) {
			sid3Matches, errSid3Matches := testDb.SearchPlayers(context.TODO(), SearchOpts{Query: string(steamid.SID64ToSID3(knownIds[0]))})
			require.NoError(t, errSid3Matches)
			require.Equal(t, 1, len(sid3Matches))
			require.Equal(t, knownIds[0], sid3Matches[0].SteamId)
		})
		t.Run("By Name", func(t *testing.T) {
			nameMatches, errMatches := testDb.SearchPlayers(context.TODO(), SearchOpts{Query: "test name"})
			require.NoError(t, errMatches)
			require.Equal(t, 2, len(nameMatches))
			for _, found := range nameMatches {
				require.Contains(t, knownIds, found.SteamId)
			}
		})
	})
	t.Run("Test Close", func(t *testing.T) {
		require.NoError(t, testDb.Close())
	})
}
