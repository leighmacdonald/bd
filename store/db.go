// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package store

import (
	"context"
	"database/sql"
	"fmt"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

func Prepare(ctx context.Context, db DBTX) (*Queries, error) {
	q := Queries{db: db}
	var err error
	if q.configStmt, err = db.PrepareContext(ctx, config); err != nil {
		return nil, fmt.Errorf("error preparing query Config: %w", err)
	}
	if q.configUpdateStmt, err = db.PrepareContext(ctx, configUpdate); err != nil {
		return nil, fmt.Errorf("error preparing query ConfigUpdate: %w", err)
	}
	if q.friendsStmt, err = db.PrepareContext(ctx, friends); err != nil {
		return nil, fmt.Errorf("error preparing query Friends: %w", err)
	}
	if q.friendsDeleteStmt, err = db.PrepareContext(ctx, friendsDelete); err != nil {
		return nil, fmt.Errorf("error preparing query FriendsDelete: %w", err)
	}
	if q.friendsInsertStmt, err = db.PrepareContext(ctx, friendsInsert); err != nil {
		return nil, fmt.Errorf("error preparing query FriendsInsert: %w", err)
	}
	if q.linksStmt, err = db.PrepareContext(ctx, links); err != nil {
		return nil, fmt.Errorf("error preparing query Links: %w", err)
	}
	if q.linksDeleteStmt, err = db.PrepareContext(ctx, linksDelete); err != nil {
		return nil, fmt.Errorf("error preparing query LinksDelete: %w", err)
	}
	if q.linksInsertStmt, err = db.PrepareContext(ctx, linksInsert); err != nil {
		return nil, fmt.Errorf("error preparing query LinksInsert: %w", err)
	}
	if q.linksUpdateStmt, err = db.PrepareContext(ctx, linksUpdate); err != nil {
		return nil, fmt.Errorf("error preparing query LinksUpdate: %w", err)
	}
	if q.listsStmt, err = db.PrepareContext(ctx, lists); err != nil {
		return nil, fmt.Errorf("error preparing query Lists: %w", err)
	}
	if q.listsDeleteStmt, err = db.PrepareContext(ctx, listsDelete); err != nil {
		return nil, fmt.Errorf("error preparing query ListsDelete: %w", err)
	}
	if q.listsInsertStmt, err = db.PrepareContext(ctx, listsInsert); err != nil {
		return nil, fmt.Errorf("error preparing query ListsInsert: %w", err)
	}
	if q.listsUpdateStmt, err = db.PrepareContext(ctx, listsUpdate); err != nil {
		return nil, fmt.Errorf("error preparing query ListsUpdate: %w", err)
	}
	if q.messageSaveStmt, err = db.PrepareContext(ctx, messageSave); err != nil {
		return nil, fmt.Errorf("error preparing query MessageSave: %w", err)
	}
	if q.messagesStmt, err = db.PrepareContext(ctx, messages); err != nil {
		return nil, fmt.Errorf("error preparing query Messages: %w", err)
	}
	if q.playerStmt, err = db.PrepareContext(ctx, player); err != nil {
		return nil, fmt.Errorf("error preparing query Player: %w", err)
	}
	if q.playerInsertStmt, err = db.PrepareContext(ctx, playerInsert); err != nil {
		return nil, fmt.Errorf("error preparing query PlayerInsert: %w", err)
	}
	if q.playerSearchStmt, err = db.PrepareContext(ctx, playerSearch); err != nil {
		return nil, fmt.Errorf("error preparing query PlayerSearch: %w", err)
	}
	if q.playerUpdateStmt, err = db.PrepareContext(ctx, playerUpdate); err != nil {
		return nil, fmt.Errorf("error preparing query PlayerUpdate: %w", err)
	}
	if q.sourcebansStmt, err = db.PrepareContext(ctx, sourcebans); err != nil {
		return nil, fmt.Errorf("error preparing query Sourcebans: %w", err)
	}
	if q.sourcebansDeleteStmt, err = db.PrepareContext(ctx, sourcebansDelete); err != nil {
		return nil, fmt.Errorf("error preparing query SourcebansDelete: %w", err)
	}
	if q.sourcebansInsertStmt, err = db.PrepareContext(ctx, sourcebansInsert); err != nil {
		return nil, fmt.Errorf("error preparing query SourcebansInsert: %w", err)
	}
	if q.userNameSaveStmt, err = db.PrepareContext(ctx, userNameSave); err != nil {
		return nil, fmt.Errorf("error preparing query UserNameSave: %w", err)
	}
	if q.userNamesStmt, err = db.PrepareContext(ctx, userNames); err != nil {
		return nil, fmt.Errorf("error preparing query UserNames: %w", err)
	}
	return &q, nil
}

func (q *Queries) Close() error {
	var err error
	if q.configStmt != nil {
		if cerr := q.configStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing configStmt: %w", cerr)
		}
	}
	if q.configUpdateStmt != nil {
		if cerr := q.configUpdateStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing configUpdateStmt: %w", cerr)
		}
	}
	if q.friendsStmt != nil {
		if cerr := q.friendsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing friendsStmt: %w", cerr)
		}
	}
	if q.friendsDeleteStmt != nil {
		if cerr := q.friendsDeleteStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing friendsDeleteStmt: %w", cerr)
		}
	}
	if q.friendsInsertStmt != nil {
		if cerr := q.friendsInsertStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing friendsInsertStmt: %w", cerr)
		}
	}
	if q.linksStmt != nil {
		if cerr := q.linksStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing linksStmt: %w", cerr)
		}
	}
	if q.linksDeleteStmt != nil {
		if cerr := q.linksDeleteStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing linksDeleteStmt: %w", cerr)
		}
	}
	if q.linksInsertStmt != nil {
		if cerr := q.linksInsertStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing linksInsertStmt: %w", cerr)
		}
	}
	if q.linksUpdateStmt != nil {
		if cerr := q.linksUpdateStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing linksUpdateStmt: %w", cerr)
		}
	}
	if q.listsStmt != nil {
		if cerr := q.listsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listsStmt: %w", cerr)
		}
	}
	if q.listsDeleteStmt != nil {
		if cerr := q.listsDeleteStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listsDeleteStmt: %w", cerr)
		}
	}
	if q.listsInsertStmt != nil {
		if cerr := q.listsInsertStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listsInsertStmt: %w", cerr)
		}
	}
	if q.listsUpdateStmt != nil {
		if cerr := q.listsUpdateStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listsUpdateStmt: %w", cerr)
		}
	}
	if q.messageSaveStmt != nil {
		if cerr := q.messageSaveStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing messageSaveStmt: %w", cerr)
		}
	}
	if q.messagesStmt != nil {
		if cerr := q.messagesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing messagesStmt: %w", cerr)
		}
	}
	if q.playerStmt != nil {
		if cerr := q.playerStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing playerStmt: %w", cerr)
		}
	}
	if q.playerInsertStmt != nil {
		if cerr := q.playerInsertStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing playerInsertStmt: %w", cerr)
		}
	}
	if q.playerSearchStmt != nil {
		if cerr := q.playerSearchStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing playerSearchStmt: %w", cerr)
		}
	}
	if q.playerUpdateStmt != nil {
		if cerr := q.playerUpdateStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing playerUpdateStmt: %w", cerr)
		}
	}
	if q.sourcebansStmt != nil {
		if cerr := q.sourcebansStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing sourcebansStmt: %w", cerr)
		}
	}
	if q.sourcebansDeleteStmt != nil {
		if cerr := q.sourcebansDeleteStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing sourcebansDeleteStmt: %w", cerr)
		}
	}
	if q.sourcebansInsertStmt != nil {
		if cerr := q.sourcebansInsertStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing sourcebansInsertStmt: %w", cerr)
		}
	}
	if q.userNameSaveStmt != nil {
		if cerr := q.userNameSaveStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing userNameSaveStmt: %w", cerr)
		}
	}
	if q.userNamesStmt != nil {
		if cerr := q.userNamesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing userNamesStmt: %w", cerr)
		}
	}
	return err
}

func (q *Queries) exec(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (sql.Result, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).ExecContext(ctx, args...)
	case stmt != nil:
		return stmt.ExecContext(ctx, args...)
	default:
		return q.db.ExecContext(ctx, query, args...)
	}
}

func (q *Queries) query(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (*sql.Rows, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryContext(ctx, args...)
	default:
		return q.db.QueryContext(ctx, query, args...)
	}
}

func (q *Queries) queryRow(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) *sql.Row {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryRowContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryRowContext(ctx, args...)
	default:
		return q.db.QueryRowContext(ctx, query, args...)
	}
}

type Queries struct {
	db                   DBTX
	tx                   *sql.Tx
	configStmt           *sql.Stmt
	configUpdateStmt     *sql.Stmt
	friendsStmt          *sql.Stmt
	friendsDeleteStmt    *sql.Stmt
	friendsInsertStmt    *sql.Stmt
	linksStmt            *sql.Stmt
	linksDeleteStmt      *sql.Stmt
	linksInsertStmt      *sql.Stmt
	linksUpdateStmt      *sql.Stmt
	listsStmt            *sql.Stmt
	listsDeleteStmt      *sql.Stmt
	listsInsertStmt      *sql.Stmt
	listsUpdateStmt      *sql.Stmt
	messageSaveStmt      *sql.Stmt
	messagesStmt         *sql.Stmt
	playerStmt           *sql.Stmt
	playerInsertStmt     *sql.Stmt
	playerSearchStmt     *sql.Stmt
	playerUpdateStmt     *sql.Stmt
	sourcebansStmt       *sql.Stmt
	sourcebansDeleteStmt *sql.Stmt
	sourcebansInsertStmt *sql.Stmt
	userNameSaveStmt     *sql.Stmt
	userNamesStmt        *sql.Stmt
}

func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db:                   tx,
		tx:                   tx,
		configStmt:           q.configStmt,
		configUpdateStmt:     q.configUpdateStmt,
		friendsStmt:          q.friendsStmt,
		friendsDeleteStmt:    q.friendsDeleteStmt,
		friendsInsertStmt:    q.friendsInsertStmt,
		linksStmt:            q.linksStmt,
		linksDeleteStmt:      q.linksDeleteStmt,
		linksInsertStmt:      q.linksInsertStmt,
		linksUpdateStmt:      q.linksUpdateStmt,
		listsStmt:            q.listsStmt,
		listsDeleteStmt:      q.listsDeleteStmt,
		listsInsertStmt:      q.listsInsertStmt,
		listsUpdateStmt:      q.listsUpdateStmt,
		messageSaveStmt:      q.messageSaveStmt,
		messagesStmt:         q.messagesStmt,
		playerStmt:           q.playerStmt,
		playerInsertStmt:     q.playerInsertStmt,
		playerSearchStmt:     q.playerSearchStmt,
		playerUpdateStmt:     q.playerUpdateStmt,
		sourcebansStmt:       q.sourcebansStmt,
		sourcebansDeleteStmt: q.sourcebansDeleteStmt,
		sourcebansInsertStmt: q.sourcebansInsertStmt,
		userNameSaveStmt:     q.userNameSaveStmt,
		userNamesStmt:        q.userNamesStmt,
	}
}
