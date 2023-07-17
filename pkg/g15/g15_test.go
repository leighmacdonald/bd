package g15_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leighmacdonald/bd/pkg/g15"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/stretchr/testify/require"
)

func TestParser_Parse(t *testing.T) {
	testFile, errOpen := os.Open(filepath.Join("testdata", "g15_dumpplayer.log"))

	require.NoError(t, errOpen)

	parser := g15.New()

	var dumpData g15.DumpData

	require.NoError(t, parser.Parse(testFile, &dumpData))
	require.Equal(t, dumpData.Names[3], "The Legendary 215 Gray Cakes")
	require.Equal(t, dumpData.Ping[4], 89)
	require.Equal(t, dumpData.Score[5], 4)
	require.Equal(t, dumpData.Deaths[6], 3)
	require.Equal(t, dumpData.Connected[7], true)
	require.Equal(t, dumpData.Team[8], 2)
	require.Equal(t, dumpData.Alive[9], true)
	require.Equal(t, dumpData.Health[10], 1)
	require.Equal(t, dumpData.SteamID[11], steamid.New(76561199201900779))
	require.Equal(t, dumpData.Valid[12], true)
	require.Equal(t, dumpData.UserID[13], 2265)
}
