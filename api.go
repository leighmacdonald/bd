package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/leighmacdonald/bd-api/models"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type FriendMap map[steamid.SID64][]steamweb.Friend

type SourcebansMap map[steamid.SID64][]models.SbBanRecord

type DataSource interface {
	Summaries(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerSummary, error)
	Bans(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerBanState, error)
	Friends(ctx context.Context, steamIDs steamid.Collection) (FriendMap, error)
	Sourcebans(ctx context.Context, steamIDs steamid.Collection) (SourcebansMap, error)
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

func (n LocalDataSource) Friends(ctx context.Context, steamIDs steamid.Collection) (FriendMap, error) {
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

func (n LocalDataSource) Sourcebans(_ context.Context, steamIDs steamid.Collection) (SourcebansMap, error) {
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

func NewDataSource(userSettings userSettings) (DataSource, error) {
	if userSettings.BdAPIEnabled {
		return createAPIDataSource(userSettings.BdAPIAddress)
	} else {
		return createLocalDataSource(userSettings.APIKey)
	}
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

func (n APIDataSource) Friends(ctx context.Context, steamIDs steamid.Collection) (FriendMap, error) {
	var out FriendMap
	if errGet := n.get(ctx, n.url("/friends", steamIDs), &out); errGet != nil {
		return nil, errGet
	}

	return out, nil
}

func (n APIDataSource) Sourcebans(ctx context.Context, steamIDs steamid.Collection) (SourcebansMap, error) {
	var out SourcebansMap
	if errGet := n.get(ctx, n.url("/sourcebans", steamIDs), &out); errGet != nil {
		return nil, errGet
	}

	return out, nil
}
