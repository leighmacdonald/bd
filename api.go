package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/leighmacdonald/bd-api/models"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
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
		return nil, errors.Wrap(errSummaries, "Failed to fetch summaries")
	}

	return summaries, nil
}

func (n LocalDataSource) Bans(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerBanState, error) {
	bans, errBans := steamweb.GetPlayerBans(ctx, steamIDs)
	if errBans != nil {
		return nil, errors.Wrap(errBans, "Failed to fetch bans")
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

func NewLocalDataSource(key string) (LocalDataSource, error) {
	if errKey := steamweb.SetKey(key); errKey != nil {
		return LocalDataSource{}, errors.Wrap(errKey, "Failed to set steam api key")
	}

	return LocalDataSource{}, nil
}

const APIDataSourceDefaultAddress = "https://bd-api.roto.lol"

// APIDataSource implements a client for the remote bd-api service.
type APIDataSource struct {
	baseURL string
	client  *http.Client
}

func NewAPIDataSource(url string) (APIDataSource, error) {
	if url == "" {
		url = APIDataSourceDefaultAddress
	}

	return APIDataSource{baseURL: url, client: &http.Client{}}, nil
}

func (n APIDataSource) url(path string, collection steamid.Collection) string {
	return fmt.Sprintf("%s%s?steamids=%s", n.baseURL, path, steamIDStringList(collection))
}

func (n APIDataSource) get(ctx context.Context, path string, results any) error {
	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if errReq != nil {
		return errors.Wrap(errReq, "Failed to create request")
	}

	resp, errResp := n.client.Do(req)
	if errResp != nil {
		return errors.Wrap(errResp, "Failed to perform request")
	}

	body, errBody := io.ReadAll(resp.Body)
	if errBody != nil {
		return errors.Wrap(errBody, "Failed to read response body")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if errJSON := json.Unmarshal(body, &results); errJSON != nil {
		return errors.Wrap(errJSON, "Failed to unmarshal json response")
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
