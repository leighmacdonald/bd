package main

import (
	"context"
	"github.com/leighmacdonald/steamid/v2/steamid"
)

type event struct {
	name  EventType
	value any
}

type evtUserMessageData struct {
	team      team
	player    string
	playerSID steamid.SID64
	userId    int64
	message   string
}

func eventSender(ctx context.Context, bd *BD, ui UserInterface) {
	for {
		select {
		case evt := <-bd.eventChan:
			switch evt.name {
			case EvtMsg:
				value := evt.value.(evtUserMessageData)
				ui.onNewMessage(value)
			}
		case <-ctx.Done():
			return
		}
	}
}
