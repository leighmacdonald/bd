package main

import (
	"context"
	"log/slog"

	"github.com/leighmacdonald/bd/store"
)

type chatRecorder struct {
	incoming chan LogEvent
	db       store.Querier
}

func newChatRecorder(db store.Querier, ingest *logIngest) chatRecorder {
	cr := chatRecorder{
		incoming: make(chan LogEvent),
		db:       db,
	}

	ingest.registerConsumer(cr.incoming, EvtMsg)

	return cr
}

func (s *chatRecorder) start(ctx context.Context) {
	for {
		select {
		case evt := <-s.incoming:
			if errUm := s.db.MessageSave(ctx, store.MessageSaveParams{
				SteamID:   evt.PlayerSID.Int64(),
				Message:   evt.Message,
				CreatedOn: evt.Timestamp,
				Team:      evt.TeamOnly,
				Dead:      evt.Dead,
			}); errUm != nil {
				slog.Error("Failed to save user message", errAttr(errUm))
				continue
			}

			slog.Debug("Chat message saved", slog.String("msg", evt.Message))
		case <-ctx.Done():
			return
		}
	}
}
