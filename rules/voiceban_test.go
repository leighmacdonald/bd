package rules

import (
	"os"
	"testing"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestVoiceBans(t *testing.T) {
	vbTestFile, err := os.CreateTemp("", "")
	require.NoError(t, err)

	defer func() {
		_ = vbTestFile.Close()
	}()

	testIDs := steamid.Collection{
		steamid.New("76561198369477018"),
		steamid.New("76561197970669109"),
		steamid.New("76561197961279983"),
	}

	require.NoError(t, VoiceBanWrite(vbTestFile, testIDs))
	_ = vbTestFile.Sync()
	_, _ = vbTestFile.Seek(0, 0)
	bans, errRead := VoiceBanRead(vbTestFile)
	require.NoError(t, errRead)

	for idx, foundID := range bans {
		require.Equal(t, testIDs[idx], foundID)
	}
}
