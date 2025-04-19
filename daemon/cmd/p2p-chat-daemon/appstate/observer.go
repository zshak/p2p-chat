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

type Observer struct {
	appState   *core.AppState
	bus        *bus.EventBus
	ctx        context.Context
	eventsChan chan interface{}
}

func NewObserver(appState *core.AppState, eventBus *bus.EventBus, ctx context.Context) (*Observer, error) {
	if appState == nil {
		return nil, errors.New("appState is nil")
	}
	return &Observer{appState: appState, bus: eventBus, ctx: ctx, eventsChan: make(chan interface{})}, nil
}

func (j *Observer) Start() {
	log.Println("app state observer observer started")
	j.bus.Subscribe(j.eventsChan, events.KeyGenerationFailedEvent{})
	j.bus.Subscribe(j.eventsChan, events.KeyLoadingFailedEvent{})
	j.bus.Subscribe(j.eventsChan, events.KeyGeneratedEvent{})
	j.bus.Subscribe(j.eventsChan, events.UserAuthenticatedEvent{})
	j.bus.Subscribe(j.eventsChan, events.ApiStartedEvent{})
	j.bus.Subscribe(j.eventsChan, events.HostInitializedEvent{})
	j.bus.Subscribe(j.eventsChan, events.DhtCreatedEvent{})
	j.bus.Subscribe(j.eventsChan, events.SetupCompletedEvent{})

	go j.listen()
}

func (j *Observer) listen() {
	for {
		select {
		case <-j.ctx.Done():
			log.Println("app state observer observer stopped")
			return

		case event := <-j.eventsChan:
			j.handleEvent(event)
		}
	}
}

func (j *Observer) handleEvent(event interface{}) {
	switch event := event.(type) {

	case events.KeyGenerationFailedEvent:
		log.Printf("Failed to generate/save key: %v", event.Err)
		j.updateState(core.StateError, event.Err)
		return

	case events.KeyLoadingFailedEvent:
		log.Printf("Failed to load/decrypt key: %v", event.Err)
		j.UpdateError(event.Err)
		return

	case events.KeyGeneratedEvent:
		log.Println("Key Generated Successfully")
		j.handleKeyProvided(event.Key)
		return

	case events.UserAuthenticatedEvent:
		log.Println("Key Loaded Successfully")
		j.handleKeyProvided(event.Key)
		return

	case events.ApiStartedEvent:
		j.handleApiStarted()
		return

	case events.HostInitializedEvent:
		j.handleHostInitializedEvent(event.Host)
		return

	case events.DhtCreatedEvent:
		j.handleDhtCreatedEvent(event.Dht)
		return

	case events.SetupCompletedEvent:
		j.updateState(core.StateRunning, nil)
		return
	}
}

func (j *Observer) updateState(state core.DaemonState, err error) {
	j.appState.Mu.Lock()
	defer j.appState.Mu.Unlock()

	j.appState.State = state

	if err != nil {
		j.appState.LastError = err
	}
}

func (j *Observer) handleKeyProvided(key crypto.PrivKey) {
	j.appState.Mu.Lock()
	defer j.appState.Mu.Unlock()

	j.appState.PrivKey = key
}

func (j *Observer) handleApiStarted() {
	j.appState.Mu.Lock()
	defer j.appState.Mu.Unlock()

	if identity.KeyExists(j.appState.KeyPath) {
		j.appState.State = core.StateWaitingForPassword
		log.Printf("Key file found at %s. Waiting for password via API.", j.appState.KeyPath)
	} else {
		j.appState.State = core.StateWaitingForKey
		log.Printf("Key file not found at %s. Waiting for key setup via API.", j.appState.KeyPath)
	}
}

func (j *Observer) handleDhtCreatedEvent(dht *dht.IpfsDHT) {
	log.Println("DHT set in application state")
	j.updateState(core.StateRunning, nil)

	j.appState.Mu.Lock()
	defer j.appState.Mu.Unlock()

	j.appState.Dht = dht
}

func (j *Observer) handleHostInitializedEvent(host *host.Host) {
	j.appState.Mu.Lock()
	j.appState.Node = host
	j.appState.Mu.Unlock()
}

func (j *Observer) UpdateError(err error) {
	j.appState.Mu.Lock()
	defer j.appState.Mu.Unlock()

	if err != nil {
		j.appState.LastError = err
	}
}
