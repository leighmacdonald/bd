package detector

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"

	"golang.org/x/exp/slog"

	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/pkg/errors"
)

// fixSteamIDFormat converts raw unquoted steamids to quoted ones
// e.g. "steamid":76561199063807260 -> "steamid": "76561199063807260".
func fixSteamIDFormat(body []byte) []byte {
	r := regexp.MustCompile(`("steamid":\+?(\d+))`)
	return r.ReplaceAll(body, []byte("\"steamid\": \"$2\""))
}

func downloadLists(ctx context.Context, logger *slog.Logger, lists ListConfigCollection) ([]rules.PlayerListSchema, []rules.RuleSchema) {
	fetchURL := func(ctx context.Context, client http.Client, url string) ([]byte, error) {
		timeout, cancel := context.WithTimeout(ctx, DurationWebRequestTimeout)
		defer cancel()
		req, reqErr := http.NewRequestWithContext(timeout, http.MethodGet, url, nil)
		if reqErr != nil {
			return nil, errors.Wrap(reqErr, "Failed to create request\n")
		}
		resp, errResp := client.Do(req)
		if errResp != nil {
			return nil, errors.Wrapf(errResp, "Failed to download urlLocation: %s\n", url)
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		body, errBody := io.ReadAll(resp.Body)
		if errBody != nil {
			return nil, errors.Wrapf(errBody, "Failed to read body: %s\n", url)
		}
		return body, nil
	}
	var playerLists []rules.PlayerListSchema
	var rulesLists []rules.RuleSchema
	mu := &sync.RWMutex{}
	client := http.Client{}
	downloadFn := func(u *ListConfig) error {
		start := time.Now()
		body, errFetch := fetchURL(ctx, client, u.URL)
		if errFetch != nil {
			return errors.Wrapf(errFetch, "Failed to fetch player list: %s", u.URL)
		}
		body = fixSteamIDFormat(body)
		dur := time.Since(start)
		switch u.ListType {
		case ListTypeTF2BDPlayerList:
			var result rules.PlayerListSchema
			if errParse := json.Unmarshal(body, &result); errParse != nil {
				return errors.Wrap(errParse, "Failed to parse request")
			}
			mu.Lock()
			playerLists = append(playerLists, result)
			mu.Unlock()
			logger.Info("Downloaded players successfully", "duration", dur, "name", result.FileInfo.Title)
		case ListTypeTF2BDRules:
			var result rules.RuleSchema
			if errParse := json.Unmarshal(body, &result); errParse != nil {
				return errors.Wrap(errParse, "Failed to parse request")
			}
			mu.Lock()
			rulesLists = append(rulesLists, result)
			mu.Unlock()
			logger.Info("Downloaded rules successfully", "duration", dur, "name", result.FileInfo.Title)
		}
		return nil
	}
	wg := &sync.WaitGroup{}
	for _, listConfig := range lists {
		if !listConfig.Enabled {
			continue
		}
		wg.Add(1)
		go func(lc *ListConfig) {
			defer wg.Done()
			if errDL := downloadFn(lc); errDL != nil {
				logger.Error("Failed to download list", "err", errDL)
			}
		}(listConfig)
	}
	wg.Wait()
	return playerLists, rulesLists
}
