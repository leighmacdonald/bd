// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0

package store

import (
	"context"
)

type Querier interface {
	MessageSave(ctx context.Context, arg MessageSaveParams) error
	Messages(ctx context.Context, steamID int64) ([]PlayerMessage, error)
	Player(ctx context.Context, steamID int64) (PlayerRow, error)
	PlayerInsert(ctx context.Context, arg PlayerInsertParams) (Player, error)
	PlayerSearch(ctx context.Context, arg PlayerSearchParams) ([]PlayerSearchRow, error)
	PlayerUpdate(ctx context.Context, arg PlayerUpdateParams) error
	UserNameSave(ctx context.Context, arg UserNameSaveParams) error
	UserNames(ctx context.Context, steamID int64) ([]PlayerName, error)
}

var _ Querier = (*Queries)(nil)
