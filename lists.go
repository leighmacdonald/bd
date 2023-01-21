package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type FileInfo struct {
	Authors     []string `json:"authors"`
	Description string   `json:"description"`
	Title       string   `json:"title"`
	UpdateURL   string   `json:"update_url"`
}
type LastSeen struct {
	PlayerName string `json:"player_name,omitempty"`
	Time       int    `json:"time,omitempty"`
}
type Players struct {
	Attributes []string `json:"attributes"`
	LastSeen   LastSeen `json:"last_seen,omitempty"`
	SteamId    any      `json:"steamid"`
	Proof      []string `json:"proof,omitempty"`
}

type TF2BDSchema struct {
	Schema   string    `json:"$schema"`
	FileInfo FileInfo  `json:"file_info"`
	Players  []Players `json:"players"`
}

func parseTF2BD(data []byte, schema *TF2BDSchema) error {
	if errUnmarshal := json.Unmarshal(data, schema); errUnmarshal != nil {
		return errUnmarshal
	}
	return nil
}

func downloadPlayerLists(ctx context.Context, listUrl ...string) []TF2BDSchema {
	urls := []string{
		"https://trusted.roto.lol/v1/steamids",
		"https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/staging/cfg/playerlist.official.json",
		"https://uncletopia.com/export/bans/tf2bd",
	}
	for _, lu := range listUrl {
		urls = append(urls, lu)
	}
	var results []TF2BDSchema
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	for _, u := range urls {
		req, reqErr := http.NewRequestWithContext(ctx, "GET", u, nil)
		if reqErr != nil {
			log.Printf("Failed to create request: %v\n", reqErr)
			continue
		}
		resp, errResp := client.Do(req)
		if errResp != nil {
			log.Printf("Failed to download url: %s %v\n", u, errResp)
			continue
		}
		body, errBody := io.ReadAll(resp.Body)
		if errBody != nil {
			log.Printf("Failed to read body: %s %v\n", u, errResp)
			continue
		}
		var result TF2BDSchema
		if errParse := parseTF2BD(body, &result); errParse != nil {
			log.Printf("Failed to parse request: %v\n", reqErr)
			continue
		}
		results = append(results, result)
		log.Printf("Downloaded playerlist successfully: %s\n", result.FileInfo.Title)
	}
	return results
}
