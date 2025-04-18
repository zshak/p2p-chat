package setup

import (
	"context"
	"errors"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
)

type Observer struct {
	appState     *core.AppState
	bus          *bus.EventBus
	ctx          context.Context
	eventsChan   chan interface{}
	keyReadyChan chan struct{}
}

func NewObserver(appState *core.AppState, eventBus *bus.EventBus, keyReadyChan chan struct{}, ctx context.Context) (*Observer, error) {
	if appState == nil {
		return nil, errors.New("appState is nil")
	}
	return &Observer{
		appState:     appState,
		bus:          eventBus,
		keyReadyChan: keyReadyChan,
		ctx:          ctx,
		eventsChan:   make(chan interface{}),
	}, nil
}

func (ob *Observer) Start() {
	log.Println("setup observer started")
	ob.bus.Subscribe(ob.eventsChan, core.UserAuthenticatedEvent{})

	go ob.listen()
}

func (ob *Observer) listen() {
	for {
		select {
		case <-ob.ctx.Done():
			log.Println("app state observer job stopped")
			return

		case event := <-ob.eventsChan:
			ob.handleEvent(event)
		}
	}
}

func (ob *Observer) handleEvent(event interface{}) {
	switch event.(type) {

	case core.UserAuthenticatedEvent:
		log.Printf("User Authenticated, Unlocking Startup")
		select {
		// close (idempotent)
		case <-ob.keyReadyChan:
			// channel closed
		default:
			// else close
			close(ob.keyReadyChan)
		}
		return
	}
}
