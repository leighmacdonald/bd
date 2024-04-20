package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/bd-api/models"
	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/bd/store"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type bulkUpdatedRemoteData struct {
	summaries  []steamweb.PlayerSummary
	bans       []steamweb.PlayerBanState
	sourcebans SourcebansMap
	friends    FriendMap
}

type playerDataUpdate struct {
	steamID    steamid.SteamID
	summary    steamweb.PlayerSummary
	bans       steamweb.PlayerBanState
	sourcebans []models.SbBanRecord
	friends    []steamweb.Friend
}

type playerDataLoader struct {
	profileUpdateQueue chan steamid.SteamID
	datasource         DataSource
	playerDataChan     chan playerDataUpdate
	db                 store.Querier
	settings           *settingsManager
	re                 *rules.Engine
}

func newPlayerDataLoader(db store.Querier, ds DataSource, settings *settingsManager, re *rules.Engine,
	profileUpdateQueue chan steamid.SteamID, playerDataChan chan playerDataUpdate,
) *playerDataLoader {
	return &playerDataLoader{
		db:                 db,
		datasource:         ds,
		settings:           settings,
		re:                 re,
		profileUpdateQueue: profileUpdateQueue,
		playerDataChan:     playerDataChan,
	}
}

// playerDataLoader will update the 3rd party data from remote APIs.
// It will wait a short amount of time between updates to attempt to batch send the requests
// as much as possible.
func (p *playerDataLoader) start(ctx context.Context) {
	var (
		queue       steamid.Collection
		updateTimer = time.NewTicker(DurationUpdateTimer)
	)

	for {
		select {
		case <-ctx.Done():
			return
		case steamID := <-p.profileUpdateQueue:
			queue = append(queue, steamID)
		case <-updateTimer.C:
			if len(queue) == 0 {
				continue
			}

			bulkData := p.fetchProfileUpdates(ctx, queue)

			// Flatten the results
			var updates []playerDataUpdate
			for _, steamID := range queue {
				u := playerDataUpdate{
					steamID:    steamID,
					friends:    make([]steamweb.Friend, 0),
					sourcebans: make([]models.SbBanRecord, 0),
					bans:       steamweb.PlayerBanState{},
				}
				for _, summary := range bulkData.summaries {
					if summary.SteamID == steamID {
						u.summary = summary
						break
					}
				}
				for _, ban := range bulkData.bans {
					if ban.SteamID == steamID {
						u.bans = ban
						break
					}
				}

				if friends, ok := bulkData.friends[steamID]; ok {
					u.friends = friends
				}

				if sourcebans, ok := bulkData.sourcebans[steamID]; ok {
					u.sourcebans = sourcebans
				}

				updates = append(updates, u)
			}

			for _, update := range updates {
				p.playerDataChan <- update
				p.saveFriends(ctx, update.steamID, update.friends)
				p.saveSourceBans(ctx, update.steamID, update.sourcebans)
			}

			slog.Info("Updated",
				slog.Int("sums", len(bulkData.summaries)), slog.Int("bans", len(bulkData.bans)),
				slog.Int("sourcebans", len(bulkData.sourcebans)), slog.Int("fiends", len(bulkData.friends)))

			queue = nil
		}
	}
}

func (p *playerDataLoader) saveSourceBans(ctx context.Context, steamID steamid.SteamID, records []models.SbBanRecord) {
	if err := p.db.SourcebansDelete(ctx, steamID.Int64()); err != nil {
		slog.Error("failed to delete sourcebans records", errAttr(err))
		return
	}

	for _, sb := range records {
		if _, err := p.db.SourcebansInsert(ctx, store.SourcebansInsertParams{
			SteamID:    steamID.Int64(),
			Site:       string(sb.SiteName),
			PlayerName: sb.PersonaName,
			Reason:     sb.Reason,
			Duration:   int64(sb.Duration),
			Permanent:  sb.Permanent,
			CreatedOn:  sb.CreatedOn,
		}); err != nil {
			slog.Error("failed to insert sourcebans record", errAttr(err))
		}
	}
}

func (p *playerDataLoader) saveFriends(ctx context.Context, steamID steamid.SteamID, friends []steamweb.Friend) {
	if err := p.db.FriendsDelete(ctx, steamID.Int64()); err != nil {
		slog.Error("failed to delete friends")
		return
	}

	for _, friend := range friends {
		if err := p.db.FriendsInsert(ctx, store.FriendsInsertParams{
			SteamID:       steamID.Int64(),
			SteamIDFriend: friend.SteamID.Int64(),
			FriendSince:   time.Unix(int64(friend.FriendSince), 0),
			CreatedOn:     time.Now(),
		}); err != nil {
			slog.Error("failed to insert friend", errAttr(err))
			continue
		}
	}
}

// fetchProfileUpdates handles fetching and updating new player data from the configured DataSource implementation,
// it handles fetching the following data points:
//
// - Valve Profile Summary
// - Valve Game/VAC Bans
// - Valve Friendslist
// - Scraped sourcebans data via bd-api at https://bd-api.roto.lol
//
// If the user does not configure their own steam api key using LocalDataSource, then the
// default bd-api backed APIDataSource will instead be used as a proxy for fetching the results.
func (p *playerDataLoader) fetchProfileUpdates(ctx context.Context, queued steamid.Collection) bulkUpdatedRemoteData {
	localCtx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	var (
		updated   bulkUpdatedRemoteData
		waitGroup = &sync.WaitGroup{}
	)

	waitGroup.Add(1)

	go func(c steamid.Collection) {
		defer waitGroup.Done()

		newSummaries, errSum := p.datasource.summaries(localCtx, c)
		if errSum == nil {
			updated.summaries = newSummaries
		}
	}(queued)

	waitGroup.Add(1)

	go func(c steamid.Collection) {
		defer waitGroup.Done()

		newBans, errSum := p.datasource.Bans(localCtx, c)
		if errSum == nil {
			updated.bans = newBans
		}
	}(queued)

	waitGroup.Add(1)

	go func(c steamid.Collection) {
		defer waitGroup.Done()

		newSourceBans, errSum := p.datasource.sourceBans(localCtx, c)
		if errSum == nil {
			updated.sourcebans = newSourceBans
		}
	}(queued)

	waitGroup.Add(1)

	go func(c steamid.Collection) {
		defer waitGroup.Done()

		newFriends, errSum := p.datasource.friends(localCtx, c)
		if errSum == nil {
			updated.friends = newFriends
		}
	}(queued)

	waitGroup.Wait()

	return updated
}
