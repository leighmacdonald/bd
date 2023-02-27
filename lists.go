package main

import (
	"context"
	"encoding/json"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

func downloadLists(ctx context.Context, lists model.ListConfigCollection) ([]rules.PlayerListSchema, []rules.RuleSchema) {
	fetchURL := func(ctx context.Context, client http.Client, url string) ([]byte, error) {
		timeout, cancel := context.WithTimeout(ctx, model.DurationWebRequestTimeout)
		defer cancel()
		req, reqErr := http.NewRequestWithContext(timeout, "GET", url, nil)
		if reqErr != nil {
			return nil, errors.Wrap(reqErr, "Failed to create request\n")
		}
		resp, errResp := client.Do(req)
		if errResp != nil {
			return nil, errors.Wrapf(errResp, "Failed to download urlLocation: %s\n", url)
		}
		body, errBody := io.ReadAll(resp.Body)
		if errBody != nil {
			return nil, errors.Wrapf(errBody, "Failed to read body: %s\n", url)
		}
		defer logClose(resp.Body)
		return body, nil
	}
	var playerLists []rules.PlayerListSchema
	var rulesLists []rules.RuleSchema
	mu := &sync.RWMutex{}
	client := http.Client{}
	downloadFn := func(u *model.ListConfig) error {
		start := time.Now()
		body, errFetch := fetchURL(ctx, client, u.URL)
		if errFetch != nil {
			return errors.Wrapf(errFetch, "Failed to fetch player list: %s", u.URL)
		}
		dur := time.Since(start)
		switch u.ListType {
		case model.ListTypeTF2BDPlayerList:
			var result rules.PlayerListSchema
			if errParse := json.Unmarshal(body, &result); errParse != nil {
				return errors.Wrap(errParse, "Failed to parse request")
			}
			mu.Lock()
			playerLists = append(playerLists, result)
			mu.Unlock()
			log.Printf("Downloaded playerlist successfully (%s): %s\n", dur.String(), result.FileInfo.Title)
		case model.ListTypeTF2BDRules:
			var result rules.RuleSchema
			if errParse := json.Unmarshal(body, &result); errParse != nil {
				return errors.Wrap(errParse, "Failed to parse request")
			}
			mu.Lock()
			rulesLists = append(rulesLists, result)
			mu.Unlock()
			log.Printf("Downloaded rules successfully (%s): %s\n", dur.String(), result.FileInfo.Title)
		}
		return nil
	}
	wg := &sync.WaitGroup{}
	for _, listConfig := range lists {
		if !listConfig.Enabled {
			continue
		}
		wg.Add(1)
		go func(lc *model.ListConfig) {
			defer wg.Done()
			if errDL := downloadFn(lc); errDL != nil {
				log.Printf("Failed to download list: %v", errDL)
			}
		}(listConfig)
	}
	wg.Wait()
	return playerLists, rulesLists
}
