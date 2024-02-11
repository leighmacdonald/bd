package main

import (
	"context"
	"encoding/json"
	"github.com/leighmacdonald/bd/rules"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// FixSteamIDFormat converts raw unquoted steamids to quoted ones
// e.g. "steamid":76561199063807260 -> "steamid": "76561199063807260".
func FixSteamIDFormat(body []byte) []byte {
	r := regexp.MustCompile(`("steamid":\+?(\d+))`)

	return r.ReplaceAll(body, []byte("\"steamid\": \"$2\""))
}

func downloadLists(ctx context.Context, lists ListConfigCollection) ([]rules.PlayerListSchema, []rules.RuleSchema) {
	fetchURL := func(ctx context.Context, client http.Client, url string) ([]byte, error) {
		timeout, cancel := context.WithTimeout(ctx, DurationWebRequestTimeout)
		defer cancel()

		req, reqErr := http.NewRequestWithContext(timeout, http.MethodGet, url, nil)
		if reqErr != nil {
			return nil, errors.Wrap(reqErr, "failed to create request")
		}

		resp, errResp := client.Do(req)
		if errResp != nil {
			return nil, errors.Wrapf(errResp, "failed to download urlLocation: %s", url)
		}

		defer func() {
			_ = resp.Body.Close()
		}()

		body, errBody := io.ReadAll(resp.Body)
		if errBody != nil {
			return nil, errors.Wrapf(errBody, "failed to read body: %s", url)
		}

		return body, nil
	}

	var (
		playerLists []rules.PlayerListSchema
		rulesLists  []rules.RuleSchema
		mutex       = &sync.RWMutex{}
		client      = http.Client{}
	)

	downloadFn := func(listConfig *ListConfig) error {
		start := time.Now()

		body, errFetch := fetchURL(ctx, client, listConfig.URL)
		if errFetch != nil {
			return errors.Wrapf(errFetch, "failed to fetch player list: %s", listConfig.URL)
		}

		body = FixSteamIDFormat(body)
		dur := time.Since(start)

		switch listConfig.ListType {
		case ListTypeTF2BDPlayerList:
			var result rules.PlayerListSchema
			if errParse := json.Unmarshal(body, &result); errParse != nil {
				return errors.Wrap(errParse, "failed to parse request")
			}

			mutex.Lock()
			playerLists = append(playerLists, result)
			mutex.Unlock()

			slog.Info("Downloaded activePlayers successfully", slog.Duration("duration", dur), slog.String("name", result.FileInfo.Title))
		case ListTypeTF2BDRules:
			var result rules.RuleSchema
			if errParse := json.Unmarshal(body, &result); errParse != nil {
				return errors.Wrap(errParse, "failed to parse request")
			}

			mutex.Lock()
			rulesLists = append(rulesLists, result)
			mutex.Unlock()

			slog.Info("Downloaded rules successfully", slog.Duration("duration", dur), slog.String("name", result.FileInfo.Title))
		}

		return nil
	}

	waitGroup := &sync.WaitGroup{}

	for _, listConfig := range lists {
		if !listConfig.Enabled {
			continue
		}

		waitGroup.Add(1)

		go func(lc *ListConfig) {
			defer waitGroup.Done()

			if errDL := downloadFn(lc); errDL != nil {
				slog.Error("Failed to download list", errAttr(errDL))
			}
		}(listConfig)
	}

	waitGroup.Wait()

	return playerLists, rulesLists
}
