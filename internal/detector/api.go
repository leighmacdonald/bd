package detector

import (
	"context"

	"github.com/leighmacdonald/bd-api/models"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type RemoteDataSource interface {
	Summaries(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerSummary, error)
	Bans(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerBanState, error)
	Friends(ctx context.Context, steamIDs steamid.Collection) (map[steamid.SID64][]steamweb.Friend, error)
	Sourcebans(ctx context.Context, steamIDs steamid.Collection) ([]models.SbBanRecord, error)
}

// LocalDataSource implements a local only data source that can be used for people who do not want to use the bd-api
// service, or if it is otherwise down.
type LocalDataSource struct{}

func (n *LocalDataSource) Summaries(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerSummary, error) {
	return []steamweb.PlayerSummary{}, nil
}

func (n *LocalDataSource) Bans(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerBanState, error) {
	return []steamweb.PlayerBanState{}, nil
}

func (n *LocalDataSource) Friends(ctx context.Context, steamIDs steamid.Collection) (map[steamid.SID64][]steamweb.Friend, error) {
	return map[steamid.SID64][]steamweb.Friend{}, nil
}

func (n *LocalDataSource) Sourcebans(ctx context.Context, steamIDs steamid.Collection) ([]models.SbBanRecord, error) {
	return []models.SbBanRecord{}, nil
}

// APIDataSource implements a client for the remote bd-api service.
type APIDataSource struct{}

func (n *APIDataSource) Summaries(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerSummary, error) {
	return []steamweb.PlayerSummary{}, nil
}

func (n *APIDataSource) Bans(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerBanState, error) {
	return []steamweb.PlayerBanState{}, nil
}

func (n *APIDataSource) Friends(ctx context.Context, steamIDs steamid.Collection) (map[steamid.SID64][]steamweb.Friend, error) {
	return map[steamid.SID64][]steamweb.Friend{}, nil
}

func (n *APIDataSource) Sourcebans(ctx context.Context, steamIDs steamid.Collection) ([]models.SbBanRecord, error) {
	return []models.SbBanRecord{}, nil
}
