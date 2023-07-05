package voiceban

import (
	"os"
	"testing"

	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/stretchr/testify/require"
)

func TestVoiceBans(t *testing.T) {
	vbTestFile, err := os.CreateTemp("", "")
	require.NoError(t, err)
	defer func() {
		_ = vbTestFile.Close()
	}()
	testIds := steamid.Collection{"76561198369477018", "76561197970669109", "76561197961279983"}
	require.NoError(t, Write(vbTestFile, testIds))
	_ = vbTestFile.Sync()
	_, _ = vbTestFile.Seek(0, 0)
	bans, errRead := Read(vbTestFile)
	require.NoError(t, errRead)
	for idx, foundId := range bans {
		require.Equal(t, testIds[idx], foundId)
	}
}
