package store

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStore(t *testing.T) {
	tempDbPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-db-%d.sqlite", time.Now().Unix()))
	logger, _ := zap.NewDevelopment()
	impl := New(tempDbPath, logger)
	defer func(store DataStore) {
		if util.Exists(tempDbPath) {
			_ = store.Close()
			if errRemove := os.Remove(tempDbPath); errRemove != nil {
				fmt.Printf("Failed to remove test database: %v\n", errRemove)
			}
		}
	}(impl)
	testStoreImpl(t, impl)
}

func testStoreImpl(t *testing.T, ds DataStore) {
	require.NoError(t, ds.Init(), "Failed to migrate default schema")
	player1 := NewPlayer(steamid.SID64(76561197961279983), util.RandomString(10))

	ctx := context.Background()
	require.NoError(t, ds.LoadOrCreatePlayer(ctx, player1.SteamId, player1), "Failed to create player")
	randName := util.RandomString(10)
	randNameLast := util.RandomString(10)
	require.NoError(t, ds.SaveUserNameHistory(ctx, player1.SteamId, randName))
	require.NoError(t, ds.SaveUserNameHistory(ctx, player1.SteamId, randName))
	require.NoError(t, ds.SaveUserNameHistory(ctx, player1.SteamId, randNameLast))
	names, errNames := ds.FetchNames(ctx, player1.SteamId)
	require.NoError(t, errNames)
	require.Equal(t, 3, len(names))

	var player2 Player
	require.NoError(t, ds.LoadOrCreatePlayer(ctx, player1.SteamId, &player2), "Failed to create player2")
	require.Equal(t, player1.Visibility, player2.Visibility)
	require.Equal(t, randNameLast, player2.NamePrevious)
	require.NoError(t, ds.SaveMessage(ctx, &UserMessage{PlayerSID: player1.SteamId, Message: util.RandomString(40)}))
	require.NoError(t, ds.SaveMessage(ctx, &UserMessage{PlayerSID: player1.SteamId, Message: util.RandomString(40)}))
	messages, errMessages := ds.FetchMessages(ctx, player1.SteamId)
	require.NoError(t, errMessages)
	require.Equal(t, 2, len(messages))
}
