package main

import (
	"context"
	"fmt"
	"os"
	"testing"

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
	testIDs := steamid.Collection{testIDb4nny, testIDCamper}

	testableDS := map[string]DataSource{}

	key, found := os.LookupEnv("BD_API_KEY")
	if found {
		lds, errLDS := createLocalDataSource(key)
		require.NoError(t, errLDS)

		testableDS["local"] = lds
	}

	baseURL := ""
	if apiURL, isTest := os.LookupEnv("API_URL"); isTest {
		baseURL = apiURL
	}

	apiDataSource, errAPI := createAPIDataSource(baseURL)

	require.NoError(t, errAPI)

	testableDS["api"] = apiDataSource

	for name, ds := range testableDS {
		dataSource := ds

		t.Run(fmt.Sprintf("%s_summary", name), func(t *testing.T) {
			t.Parallel()

			summaries, errSum := dataSource.Summaries(ctx, testIDs)
			require.NoError(t, errSum)
			require.Equal(t, len(testIDs), len(summaries))
		})

		t.Run(fmt.Sprintf("%s_friends", name), func(t *testing.T) {
			t.Parallel()

			friends, errSum := dataSource.friends(ctx, testIDs)
			require.NoError(t, errSum)
			require.Equal(t, len(testIDs), len(friends))
		})

		t.Run(fmt.Sprintf("%s_bans", name), func(t *testing.T) {
			t.Parallel()

			vacBans, errSum := dataSource.Bans(ctx, testIDs)
			require.NoError(t, errSum)
			require.Equal(t, len(testIDs), len(vacBans))
		})

		t.Run(fmt.Sprintf("%s_sourcebans", name), func(t *testing.T) {
			t.Parallel()

			sourcebans, errSum := dataSource.sourceBans(ctx, testIDs)
			require.NoError(t, errSum)
			require.Equal(t, len(testIDs), len(sourcebans))
		})
	}
}
