package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/leighmacdonald/bd/rules"
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
			return nil, errors.Join(reqErr, errCreateRequest)
		}

		resp, errResp := client.Do(req)
		if errResp != nil {
			return nil, errors.Join(errResp, errPerformRequest)
		}

		defer func() {
			_ = resp.Body.Close()
		}()

		body, errBody := io.ReadAll(resp.Body)
		if errBody != nil {
			return nil, errors.Join(errBody, errReadResponse)
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
			return fmt.Errorf("%w: %s", errFetchPlayerList, listConfig.URL)
		}

		body = FixSteamIDFormat(body)
		dur := time.Since(start)

		switch listConfig.ListType {
		case ListTypeTF2BDPlayerList:
			var result rules.PlayerListSchema
			if errParse := json.Unmarshal(body, &result); errParse != nil {
				return errors.Join(errParse, errDecodeResponse)
			}

			mutex.Lock()
			playerLists = append(playerLists, result)
			mutex.Unlock()

			slog.Info("Downloaded activePlayers successfully", slog.Duration("duration", dur), slog.String("name", result.FileInfo.Title))
		case ListTypeTF2BDRules:
			var result rules.RuleSchema
			if errParse := json.Unmarshal(body, &result); errParse != nil {
				return errors.Join(errParse, errDecodeResponse)
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

// refreshLists updates the 3rd party player lists using their update url.
func refreshLists(ctx context.Context) {
	playerLists, ruleLists := downloadLists(ctx, d.settings.Lists)
	for _, list := range playerLists {
		boundList := list

		count, errImport := d.rules.ImportPlayers(&boundList)
		if errImport != nil {
			slog.Error("Failed to import player list", slog.String("name", boundList.FileInfo.Title), errAttr(errImport))
		} else {
			slog.Info("Imported player list", slog.String("name", boundList.FileInfo.Title), slog.Int("count", count))
		}
	}

	for _, list := range ruleLists {
		boundList := list

		count, errImport := d.rules.ImportRules(&boundList)
		if errImport != nil {
			slog.Error("Failed to import rules list (%s): %v\n", slog.String("name", boundList.FileInfo.Title), errAttr(errImport))
		} else {
			slog.Info("Imported rules list", slog.String("name", boundList.FileInfo.Title), slog.Int("count", count))
		}
	}
}
