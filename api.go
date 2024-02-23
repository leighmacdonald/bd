package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/bd-api/models"
	"github.com/leighmacdonald/bd/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type FriendMap map[steamid.SID64][]steamweb.Friend

type SourcebansMap map[steamid.SID64][]models.SbBanRecord

type DataSource interface {
	Summaries(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerSummary, error)
	Bans(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerBanState, error)
	friends(ctx context.Context, steamIDs steamid.Collection) (FriendMap, error)
	sourceBans(ctx context.Context, steamIDs steamid.Collection) (SourcebansMap, error)
}

// LocalDataSource implements a local only data source that can be used for people who do not want to use the bd-api
// service, or if it is otherwise down.
type LocalDataSource struct{}

func (n LocalDataSource) Summaries(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerSummary, error) {
	summaries, errSummaries := steamweb.PlayerSummaries(ctx, steamIDs)
	if errSummaries != nil {
		return nil, errors.Join(errSummaries, errFetchSummaries)
	}

	return summaries, nil
}

func (n LocalDataSource) Bans(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerBanState, error) {
	bans, errBans := steamweb.GetPlayerBans(ctx, steamIDs)
	if errBans != nil {
		return nil, errors.Join(errBans, errFetchBans)
	}

	return bans, nil
}

func (n LocalDataSource) friends(ctx context.Context, steamIDs steamid.Collection) (FriendMap, error) {
	var (
		resp      = FriendMap{}
		waitGroup = &sync.WaitGroup{}
		mutex     = &sync.RWMutex{}
	)

	for _, steamID := range steamIDs {
		waitGroup.Add(1)

		go func(sid steamid.SID64) {
			defer waitGroup.Done()

			var (
				friends      []steamweb.Friend
				errSummaries error
			)

			friends, errSummaries = steamweb.GetFriendList(ctx, sid)
			if errSummaries != nil {
				friends = []steamweb.Friend{}
			}

			mutex.Lock()
			defer mutex.Unlock()

			if friends == nil {
				resp[sid] = []steamweb.Friend{}
			} else {
				resp[sid] = friends
			}
		}(steamID)
	}

	waitGroup.Wait()

	return resp, nil
}

func (n LocalDataSource) sourceBans(_ context.Context, steamIDs steamid.Collection) (SourcebansMap, error) {
	dummy := SourcebansMap{}
	for _, sid := range steamIDs {
		dummy[sid] = []models.SbBanRecord{}
	}

	return dummy, nil
}

func createLocalDataSource(key string) (*LocalDataSource, error) {
	if errKey := steamweb.SetKey(key); errKey != nil {
		return nil, errors.Join(errKey, errAPIKey)
	}

	return &LocalDataSource{}, nil
}

func newDataSource(userSettings userSettings) (DataSource, error) { //nolint:ireturn
	if userSettings.BdAPIEnabled {
		return createAPIDataSource(userSettings.BdAPIAddress)
	}

	return createLocalDataSource(userSettings.APIKey)
}

// steamIDStringList transforms a steamid.Collection into a comma separated list of SID64 strings.
func steamIDStringList(collection steamid.Collection) string {
	ids := make([]string, len(collection))
	for index, steamID := range collection {
		ids[index] = steamID.String()
	}

	return strings.Join(ids, ",")
}

const APIDataSourceDefaultAddress = "https://bd-api.roto.lol"

// APIDataSource implements a client for the remote bd-api service.
type APIDataSource struct {
	baseURL string
	client  *http.Client
}

func createAPIDataSource(sourceURL string) (*APIDataSource, error) {
	if sourceURL == "" {
		sourceURL = APIDataSourceDefaultAddress
	}

	_, errParse := url.Parse(sourceURL)
	if errParse != nil {
		return nil, errors.Join(errParse, errDataSourceAPIAddr)
	}

	return &APIDataSource{baseURL: sourceURL, client: &http.Client{}}, nil
}

func (n APIDataSource) url(path string, collection steamid.Collection) string {
	return fmt.Sprintf("%s%s?steamids=%s", n.baseURL, path, steamIDStringList(collection))
}

func (n APIDataSource) get(ctx context.Context, path string, results any) error {
	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if errReq != nil {
		return errors.Join(errReq, errCreateRequest)
	}

	resp, errResp := n.client.Do(req)
	if errResp != nil {
		return errors.Join(errResp, errPerformRequest)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if errJSON := json.NewDecoder(resp.Body).Decode(&results); errJSON != nil {
		return errors.Join(errJSON, errDecodeResponse)
	}

	return nil
}

func (n APIDataSource) Summaries(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerSummary, error) {
	var out []steamweb.PlayerSummary
	if errGet := n.get(ctx, n.url("/summary", steamIDs), &out); errGet != nil {
		return nil, errGet
	}

	return out, nil
}

func (n APIDataSource) Bans(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerBanState, error) {
	var out []steamweb.PlayerBanState
	if errGet := n.get(ctx, n.url("/bans", steamIDs), &out); errGet != nil {
		return nil, errGet
	}

	return out, nil
}

func (n APIDataSource) friends(ctx context.Context, steamIDs steamid.Collection) (FriendMap, error) {
	var out FriendMap
	if errGet := n.get(ctx, n.url("/friends", steamIDs), &out); errGet != nil {
		return nil, errGet
	}

	return out, nil
}

func (n APIDataSource) sourceBans(ctx context.Context, steamIDs steamid.Collection) (SourcebansMap, error) {
	var out SourcebansMap
	if errGet := n.get(ctx, n.url("/sourcebans", steamIDs), &out); errGet != nil {
		return nil, errGet
	}

	return out, nil
}

type profileUpdater struct {
	profileUpdateQueue chan steamid.SID64
	datasource         DataSource
	state              *gameState
	db                 store.Querier
	settings           *settingsManager
}

func newProfileUpdater(db store.Querier, ds DataSource, state *gameState, settings *settingsManager) *profileUpdater {
	return &profileUpdater{
		db:                 db,
		datasource:         ds,
		state:              state,
		settings:           settings,
		profileUpdateQueue: make(chan steamid.SID64),
	}
}

// profileUpdater will update the 3rd party data from remote APIs.
// It will wait a short amount of time between updates to attempt to batch send the requests
// as much as possible.
func (p profileUpdater) start(ctx context.Context) {
	var (
		queue       steamid.Collection
		update      = make(chan any)
		updateTimer = time.NewTicker(DurationUpdateTimer)
	)

	for {
		select {
		case <-ctx.Done():
			return
		case <-updateTimer.C:
			go func() { update <- true }()
		case steamID := <-p.profileUpdateQueue:
			queue = append(queue, steamID)
			if len(queue) == 100 {
				go func() { update <- true }()
			}
		case <-update:
			// TODO wait for 1 second or 100 profiles and batch update
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
						if actualPlayer, errPlayer := p.state.players.bySteamID(steamID); errPlayer == nil {
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
func (p profileUpdater) applyRemoteData(data updatedRemoteData) {
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

type updatedRemoteData struct {
	summaries  []steamweb.PlayerSummary
	bans       []steamweb.PlayerBanState
	sourcebans SourcebansMap
	friends    FriendMap
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
func (p profileUpdater) fetchProfileUpdates(ctx context.Context, queued steamid.Collection) updatedRemoteData {
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
