package detector_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/leighmacdonald/bd/internal/detector"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/stretchr/testify/require"
)

func TestDataSource(t *testing.T) {
	const (
		testIDb4nny  steamid.SID64 = "76561197970669109"
		testIDCamper steamid.SID64 = "76561197992870439"
	)

	ctx := context.Background()
	testIds := steamid.Collection{testIDb4nny, testIDCamper}

	testableDS := map[string]detector.RemoteDataSource{}

	key, found := os.LookupEnv("BD_API_KEY")
	if found {
		lds, errLDS := detector.NewLocalDataSource(key)
		require.NoError(t, errLDS)

		testableDS["local"] = lds
	}

	apiDataSource, errAPI := detector.NewAPIDataSource("")
	require.NoError(t, errAPI)

	testableDS["api"] = apiDataSource

	for name, dataSource := range testableDS {
		t.Run(fmt.Sprintf("%s_summary", name), func(t *testing.T) {
			summaries, errSum := dataSource.Summaries(ctx, testIds)
			require.NoError(t, errSum)
			require.Equal(t, len(testIds), len(summaries))
		})

		t.Run(fmt.Sprintf("%s_friends", name), func(t *testing.T) {
			summaries, errSum := dataSource.Friends(ctx, testIds)
			require.NoError(t, errSum)
			require.Equal(t, len(testIds), len(summaries))
		})

		t.Run(fmt.Sprintf("%s_bans", name), func(t *testing.T) {
			summaries, errSum := dataSource.Bans(ctx, testIds)
			require.NoError(t, errSum)
			require.Equal(t, len(testIds), len(summaries))
		})

		t.Run(fmt.Sprintf("%s_sourcebans", name), func(t *testing.T) {
			summaries, errSum := dataSource.Sourcebans(ctx, testIds)
			require.NoError(t, errSum)
			require.Equal(t, len(testIds), len(summaries))
		})
	}
}
