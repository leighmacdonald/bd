package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"
)

func downloadPlayerLists(ctx context.Context, listUrl ...string) playerListCollection {
	urls := []string{
		"https://trusted.roto.lol/v1/steamids",
		"https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/staging/cfg/playerlist.official.json",
		"https://uncletopia.com/export/bans/tf2bd",
	}
	urls = append(urls, listUrl...)
	var results playerListCollection
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
		var result TF2BDPlayerList
		if errParse := parseTF2BD(body, &result); errParse != nil {
			log.Printf("Failed to parse request: %v\n", reqErr)
			continue
		}
		results = append(results, result)
		log.Printf("Downloaded playerlist successfully: %s\n", result.FileInfo.Title)
	}
	return results
}
