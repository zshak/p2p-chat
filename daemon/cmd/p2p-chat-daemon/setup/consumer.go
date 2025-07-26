package setup

import (
	"context"
	"errors"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core/events"
)

type Consumer struct {
	appState     *core.AppState
	bus          *bus.EventBus
	ctx          context.Context
	eventsChan   chan interface{}
	keyReadyChan chan struct{}
}

func NewConsumer(appState *core.AppState, eventBus *bus.EventBus, keyReadyChan chan struct{}, ctx context.Context) (*Consumer, error) {
	if appState == nil {
		return nil, errors.New("appState is nil")
	}
	return &Consumer{
		appState:     appState,
		bus:          eventBus,
		keyReadyChan: keyReadyChan,
		ctx:          ctx,
		eventsChan:   make(chan interface{}),
	}, nil
}

func (c *Consumer) Start() {
	log.Println("setup observer started")
	c.bus.Subscribe(c.eventsChan, events.UserAuthenticatedEvent{})
	c.bus.Subscribe(c.eventsChan, events.KeyGeneratedEvent{})

	go c.listen()
}

func (c *Consumer) listen() {
	for {
		select {
		case <-c.ctx.Done():
			log.Println("app state observer job stopped")
			return

		case event := <-c.eventsChan:
			c.handleEvent(event)
		}
	}
}

func (c *Consumer) handleEvent(event interface{}) {
	switch event.(type) {

	case events.UserAuthenticatedEvent, events.KeyGeneratedEvent:
		log.Printf("User Authenticated, Unlocking Startup")
		select {
		case <-c.keyReadyChan:
		default:
			// else close
			close(c.keyReadyChan)
		}
		return
	}
}
