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
	t.Parallel()

	const (
		testIDb4nny  steamid.SID64 = "76561197970669109"
		testIDCamper steamid.SID64 = "76561197992870439"
	)

	ctx := context.Background()
	testIds := steamid.Collection{testIDb4nny, testIDCamper}

	testableDS := map[string]detector.DataSource{}

	key, found := os.LookupEnv("BD_API_KEY")
	if found {
		lds, errLDS := detector.NewLocalDataSource(key)
		require.NoError(t, errLDS)

		testableDS["local"] = lds
	}

	baseURL := ""
	if _, isTest := os.LookupEnv("TEST"); isTest {
		baseURL = "http://localhost:8888"
	}

	apiDataSource, errAPI := detector.NewAPIDataSource(baseURL)

	require.NoError(t, errAPI)

	testableDS["api"] = apiDataSource

	for name, ds := range testableDS {
		dataSource := ds

		t.Run(fmt.Sprintf("%s_summary", name), func(t *testing.T) {
			t.Parallel()

			summaries, errSum := dataSource.Summaries(ctx, testIds)
			require.NoError(t, errSum)
			require.Equal(t, len(testIds), len(summaries))
		})

		t.Run(fmt.Sprintf("%s_friends", name), func(t *testing.T) {
			t.Parallel()

			friends, errSum := dataSource.Friends(ctx, testIds)
			require.NoError(t, errSum)
			require.Equal(t, len(testIds), len(friends))
		})

		t.Run(fmt.Sprintf("%s_bans", name), func(t *testing.T) {
			t.Parallel()

			vacBans, errSum := dataSource.Bans(ctx, testIds)
			require.NoError(t, errSum)
			require.Equal(t, len(testIds), len(vacBans))
		})

		t.Run(fmt.Sprintf("%s_sourcebans", name), func(t *testing.T) {
			t.Parallel()

			sourcebans, errSum := dataSource.Sourcebans(ctx, testIds)
			require.NoError(t, errSum)
			require.Equal(t, len(testIds), len(sourcebans))
		})
	}
}
