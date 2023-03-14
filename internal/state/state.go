package state

import (
	"github.com/leighmacdonald/bd/internal/model"
	"sync"
)

type Handler func(event UpdateEvent)

var (
	players        []model.Player
	server         model.Server
	incomingEvents chan UpdateEvent
	handlers       map[updateType][]Handler
	handlersMu     sync.RWMutex
)

func Update(events ...UpdateEvent) {
	for _, event := range events {
		incomingEvents <- event
	}
}

func Register(typ updateType, handler Handler) {
	handlersMu.Lock()
	defer handlersMu.Unlock()
	if _, found := handlers[typ]; !found {
		handlers[typ] = []Handler{}
	}
	handlers[typ] = append(handlers[typ], handler)
}

func incomingEventHandler() {
	for {
		select {
		case event := <-incomingEvents:
			eventHandler, found := handlers[event.Kind]
			if !found {
				continue
			}
			for _, handler := range eventHandler {
				handler(event)
			}
		}
	}
}

func init() {
	handlers = make(map[updateType][]Handler)
	incomingEvents = make(chan UpdateEvent)
	go incomingEventHandler()
}
