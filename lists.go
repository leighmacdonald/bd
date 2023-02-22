package main

import (
	"bytes"
	"context"
	"github.com/leighmacdonald/bd/model"
	"github.com/leighmacdonald/bd/pkg/rules"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
)

func fetchURL(ctx context.Context, client http.Client, url string) ([]byte, error) {
	req, reqErr := http.NewRequestWithContext(ctx, "GET", url, nil)
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

func downloadLists(ctx context.Context, lists []model.ListConfig) ([]rules.PlayerListSchema, []rules.RuleSchema) {
	var playerLists []rules.PlayerListSchema
	var rulesLists []rules.RuleSchema
	client := http.Client{}
	for _, u := range lists {
		if !u.Enabled {
			continue
		}
		body, errFetch := fetchURL(ctx, client, u.URL)
		if errFetch != nil {
			log.Printf("Failed to fetch player list: %v", u.URL)
			continue
		}
		switch u.ListType {
		case model.ListTypeTF2BDPlayerList:
			var result rules.PlayerListSchema
			if errParse := rules.ParsePlayerSchema(bytes.NewReader(body), &result); errParse != nil {
				log.Printf("Failed to parse request: %v\n", errParse)
				continue
			}
			playerLists = append(playerLists, result)
			log.Printf("Downloaded playerlist successfully: %s\n", result.FileInfo.Title)
		case model.ListTypeTF2BDRules:
			var result rules.RuleSchema
			if errParse := rules.ParseRulesList(bytes.NewReader(body), &result); errParse != nil {
				log.Printf("Failed to parse request: %v\n", errParse)
				continue
			}
			rulesLists = append(rulesLists, result)
			log.Printf("Downloaded rules successfully: %s\n", result.FileInfo.Title)
		}

	}
	return playerLists, rulesLists
}
