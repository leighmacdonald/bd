package main

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStore(t *testing.T) {
	tempDb := filepath.Join(os.TempDir(), fmt.Sprintf("test-db-%d.sqlite", time.Now().Unix()))
	defer func() {
		if exists(tempDb) {
			os.Remove(tempDb)
		}
	}()
	testStoreImpl(t, newSqliteStore(tempDb))
}

func testStoreImpl(t *testing.T, ds dataStore) {
	require.NoError(t, ds.Init(), "Failed to migrate default schema")
	p1 := steamid.SID64(76561197961279983)
	var player1 model.PlayerState
	require.NoError(t, ds.LoadOrCreatePlayer(context.Background(), p1, &player1), "Failed to create new player")

}
