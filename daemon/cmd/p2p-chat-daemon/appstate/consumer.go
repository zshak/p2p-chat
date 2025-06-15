package appstate

import (
	"context"
	"errors"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/identity"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core/events"
)

type Consumer struct {
	appState   *core.AppState
	bus        *bus.EventBus
	ctx        context.Context
	eventsChan chan interface{}
}

func NewConsumer(appState *core.AppState, eventBus *bus.EventBus, ctx context.Context) (*Consumer, error) {
	if appState == nil {
		return nil, errors.New("appState is nil")
	}
	return &Consumer{appState: appState, bus: eventBus, ctx: ctx, eventsChan: make(chan interface{})}, nil
}

func (c *Consumer) Start() {
	log.Println("app state observer observer started")
	c.bus.Subscribe(c.eventsChan, events.KeyGenerationFailedEvent{})
	c.bus.Subscribe(c.eventsChan, events.KeyLoadingFailedEvent{})
	c.bus.Subscribe(c.eventsChan, events.KeyGeneratedEvent{})
	c.bus.Subscribe(c.eventsChan, events.UserAuthenticatedEvent{})
	c.bus.Subscribe(c.eventsChan, events.ApiStartedEvent{})
	c.bus.Subscribe(c.eventsChan, events.HostInitializedEvent{})
	c.bus.Subscribe(c.eventsChan, events.DhtCreatedEvent{})
	c.bus.Subscribe(c.eventsChan, events.SetupCompletedEvent{})

	go c.listen()
}

func (c *Consumer) listen() {
	for {
		select {
		case <-c.ctx.Done():
			log.Println("app state observer observer stopped")
			return

		case event := <-c.eventsChan:
			c.handleEvent(event)
		}
	}
}

func (c *Consumer) handleEvent(event interface{}) {
	switch event := event.(type) {

	case events.KeyGenerationFailedEvent:
		log.Printf("Failed to generate/save key: %v", event.Err)
		c.updateState(core.StateError, event.Err)
		return

	case events.KeyLoadingFailedEvent:
		log.Printf("Failed to load/decrypt key: %v", event.Err)
		c.UpdateError(event.Err)
		return

	case events.KeyGeneratedEvent:
		log.Println("Key Generated Successfully")
		c.handleKeyProvided(event.Key, nil)
		return

	case events.UserAuthenticatedEvent:
		log.Println("Key Loaded Successfully")
		c.handleKeyProvided(event.Key, event.DbKey)
		return

	case events.ApiStartedEvent:
		c.handleApiStarted()
		return

	case events.HostInitializedEvent:
		c.handleHostInitializedEvent(event.Host)
		return

	case events.DhtCreatedEvent:
		c.handleDhtCreatedEvent(event.Dht)
		return

	case events.SetupCompletedEvent:
		c.updateState(core.StateRunning, nil)
		return
	}
}

func (c *Consumer) updateState(state core.DaemonState, err error) {
	c.appState.Mu.Lock()
	defer c.appState.Mu.Unlock()

	c.appState.State = state

	if err != nil {
		c.appState.LastError = err
	}
}

func (c *Consumer) handleKeyProvided(key crypto.PrivKey, dbKey []byte) {
	c.appState.Mu.Lock()
	defer c.appState.Mu.Unlock()

	c.appState.PrivKey = key

	if dbKey != nil {
		c.appState.DbKey = dbKey
	}
}

func (c *Consumer) handleApiStarted() {
	c.appState.Mu.Lock()
	defer c.appState.Mu.Unlock()

	if identity.KeyExists(c.appState.KeyPath) {
		c.appState.State = core.StateWaitingForPassword
		log.Printf("Key file found at %s. Waiting for password via API.", c.appState.KeyPath)
	} else {
		c.appState.State = core.StateWaitingForKey
		log.Printf("Key file not found at %s. Waiting for key setup via API.", c.appState.KeyPath)
	}
}

func (c *Consumer) handleDhtCreatedEvent(dht *dht.IpfsDHT) {
	log.Println("DHT set in application state")
	c.updateState(core.StateRunning, nil)

	c.appState.Mu.Lock()
	defer c.appState.Mu.Unlock()

	c.appState.Dht = dht
}

func (c *Consumer) handleHostInitializedEvent(host *host.Host) {
	log.Println("Host set in application state")
	c.appState.Mu.Lock()
	c.appState.Node = host
	c.appState.Mu.Unlock()
}

func (c *Consumer) UpdateError(err error) {
	c.appState.Mu.Lock()
	defer c.appState.Mu.Unlock()

	if err != nil {
		c.appState.LastError = err
	}
}
