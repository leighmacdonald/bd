package main

import (
	"context"
	"github.com/leighmacdonald/bd/rules"
	"github.com/leighmacdonald/bd/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"log/slog"
	"sync"
	"time"
)

type updatedRemoteData struct {
	summaries  []steamweb.PlayerSummary
	bans       []steamweb.PlayerBanState
	sourcebans SourcebansMap
	friends    FriendMap
}

type playerDataLoader struct {
	profileUpdateQueue chan steamid.SID64
	datasource         DataSource
	state              *gameState
	db                 store.Querier
	settings           *settingsManager
	re                 *rules.Engine
}

func newPlayerDataLoader(db store.Querier, ds DataSource, state *gameState, settings *settingsManager, re *rules.Engine) *playerDataLoader {
	return &playerDataLoader{
		db:                 db,
		datasource:         ds,
		state:              state,
		settings:           settings,
		re:                 re,
		profileUpdateQueue: make(chan steamid.SID64),
	}
}

func (p playerDataLoader) queue(sid64 steamid.SID64) {
	p.profileUpdateQueue <- sid64
}

// playerDataLoader will update the 3rd party data from remote APIs.
// It will wait a short amount of time between updates to attempt to batch send the requests
// as much as possible.
func (p playerDataLoader) start(ctx context.Context) {
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

			updateData := p.fetchProfileUpdates(ctx, queue)
			p.applyRemoteData(updateData)

			for _, player := range p.state.players.all() {
				localPlayer := player
				if errSave := p.db.PlayerUpdate(ctx, localPlayer.toUpdateParams()); errSave != nil {
					if errSave.Error() != "sql: database is closed" {
						slog.Error("Failed to save updated player state",
							slog.String("sid", localPlayer.SID64().String()), errAttr(errSave))
					}
				}

				p.state.players.update(localPlayer)
			}

			ourSteamID := p.settings.Settings().SteamID

			for steamID, friends := range updateData.friends {
				for _, friend := range friends {
					if friend.SteamID == ourSteamID {
						if actualPlayer, errPlayer := p.state.players.bySteamID(steamID, true); errPlayer == nil {
							actualPlayer.OurFriend = true

							p.state.players.update(actualPlayer)

							break
						}
					}
				}
			}

			slog.Info("Updated",
				slog.Int("sums", len(updateData.summaries)), slog.Int("bans", len(updateData.bans)),
				slog.Int("sourcebans", len(updateData.sourcebans)), slog.Int("fiends", len(updateData.friends)))

			queue = nil
		}
	}
}

// applyRemoteData updates the current player states with new incoming data.
func (p playerDataLoader) applyRemoteData(data updatedRemoteData) {
	players := p.state.players.all()

	for _, curPlayer := range players {
		player := curPlayer
		for _, sum := range data.summaries {
			if sum.SteamID == player.SteamID {
				player.AvatarHash = sum.AvatarHash
				player.AccountCreatedOn = time.Unix(int64(sum.TimeCreated), 0)
				player.Visibility = int64(sum.CommunityVisibilityState)

				break
			}
		}

		for _, ban := range data.bans {
			if ban.SteamID == player.SteamID {
				player.CommunityBanned = ban.CommunityBanned
				player.CommunityBanned = ban.VACBanned
				player.GameBans = int64(ban.NumberOfGameBans)
				player.VacBans = int64(ban.NumberOfVACBans)
				player.EconomyBan = ban.EconomyBan

				if ban.VACBanned && ban.DaysSinceLastBan > 0 {
					player.LastVacBanOn = time.Now().AddDate(0, 0, -ban.DaysSinceLastBan).Unix()
				}

				break
			}
		}

		if sb, ok := data.sourcebans[player.SteamID]; ok {
			player.Sourcebans = sb
		}

		player.UpdatedOn = time.Now()
		player.ProfileUpdatedOn = player.UpdatedOn

		p.state.players.update(player)
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
func (p playerDataLoader) fetchProfileUpdates(ctx context.Context, queued steamid.Collection) updatedRemoteData {
	localCtx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	var (
		updated   updatedRemoteData
		waitGroup = &sync.WaitGroup{}
	)

	waitGroup.Add(1)

	go func(c steamid.Collection) {
		defer waitGroup.Done()

		newSummaries, errSum := p.datasource.Summaries(localCtx, c)
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
