package store_test

import (
	"context"
	"testing"

	"github.com/leighmacdonald/bd/internal/store"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPlayer(t *testing.T) {
	const msgCount = 10

	logger := zap.NewNop()

	database := store.New(":memory:", logger)
	if errInit := database.Init(); errInit != nil {
		t.Fatalf("failed to setup database: %v", errInit)
	}

	t.Cleanup(func() {
		_ = database.Close()
	})

	player1 := store.NewPlayer(steamid.New(76561197961279983), util.RandomString(10))

	t.Run("Create New Player", func(t *testing.T) {
		require.NoError(t, database.GetPlayer(context.TODO(), player1.SteamID, true, player1), "Failed to create player")
	})

	t.Run("Fetch Existing Player", func(t *testing.T) {
		require.NoError(t, database.GetPlayer(context.TODO(), player1.SteamID, false, player1), "Failed to create player")
	})

	t.Run("Add User Messages", func(t *testing.T) {
		for i := 0; i < msgCount; i++ {
			msg, errMsg := store.NewUserMessage(player1.SteamID, util.RandomString(10), false, false)
			require.NoError(t, errMsg)
			require.NoError(t, database.SaveMessage(context.TODO(), msg))
		}
	})

	t.Run("Fetch User Messages", func(t *testing.T) {
		msgs, errMsgs := database.FetchMessages(context.TODO(), player1.SteamID)
		require.NoError(t, errMsgs)
		require.Equal(t, msgCount, len(msgs))
	})

	t.Run("Add User Names", func(t *testing.T) {
		for i := 0; i < msgCount; i++ {
			msg, errMsg := store.NewUserNameHistory(player1.SteamID, util.RandomString(10))
			require.NoError(t, errMsg)
			require.NoError(t, database.SaveUserNameHistory(context.TODO(), msg))
		}
	})

	t.Run("Fetch User Names", func(t *testing.T) {
		names, errMsgs := database.FetchNames(context.TODO(), player1.SteamID)
		require.NoError(t, errMsgs)
		require.Equal(t, msgCount+1, len(names))
	})

	t.Run("Search Players", func(t *testing.T) {
		knownIds := steamid.Collection{
			"76561197998365611", "76561197977133523", "76561198065825165", "76561198004429398", "76561198182505218",
		}
		knownNames := []string{"test name 1", "test name 2", "Blah Blah", "bob", "sally"}
		for idx, sid := range knownIds {
			player := store.NewPlayer(sid, knownNames[idx])
			require.NoError(t, database.GetPlayer(context.TODO(), player.SteamID, true, player), "Failed to create player")
		}
		t.Run("By SteamID", func(t *testing.T) {
			sid3Matches, errSid3Matches := database.SearchPlayers(context.TODO(), store.SearchOpts{Query: string(steamid.SID64ToSID3(knownIds[0]))})
			require.NoError(t, errSid3Matches)
			require.Equal(t, 1, len(sid3Matches))
			require.Equal(t, knownIds[0], sid3Matches[0].SteamID)
		})
		t.Run("By Name", func(t *testing.T) {
			nameMatches, errMatches := database.SearchPlayers(context.TODO(), store.SearchOpts{Query: "test name"})
			require.NoError(t, errMatches)
			require.Equal(t, 2, len(nameMatches))
			for _, found := range nameMatches {
				require.Contains(t, knownIds, found.SteamID)
			}
		})
	})
}
