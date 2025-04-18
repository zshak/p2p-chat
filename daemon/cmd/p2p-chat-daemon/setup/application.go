package setup

import (
	"context"
	"fmt"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/appstate"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/config"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/discovery"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/peer"
	uiapi "p2p-chat-daemon/cmd/p2p-chat-daemon/ui-api"
)

type Application struct {
	ctx      context.Context
	eventBus *bus.EventBus
	config   *config.Config
	appstate *core.AppState
}

func NewApplication(cfg *config.Config) (*Application, error) {
	ctx, cancel := context.WithCancel(context.Background())
	appState := core.NewAppState(cfg.P2P.PrivateKeyPath)

	eventbus := bus.NewEventBus()
	appStateObs, err := appstate.NewObserver(appState, eventbus, ctx)

	if err != nil {
		log.Fatal("app state observer startup failed", err)
	}
	appStateObs.Start()

	_, _, err = uiapi.StartAPIServer(ctx, cfg.API.ListenAddr, appState, eventbus)
	eventbus.Publish(core.ApiStartedEvent{})

	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create api service: %w", err)
	}

	app := &Application{
		ctx:      ctx,
		config:   cfg,
		appstate: appState,
		eventBus: eventbus,
	}

	return app, nil
}

func (app *Application) Start() error {
	keyReadyChan := make(chan struct{})
	obs, err := NewObserver(app.appstate, app.eventBus, keyReadyChan, app.ctx)

	if err != nil {
		return err
	}

	obs.Start()

	log.Println("waiting for key signal")
	select {
	case <-keyReadyChan:
		log.Println("P2P Initializer: Key signal received.")
	case <-app.ctx.Done():
		log.Println("P2P Initializer: Shutdown signal received before key was ready.")
	}

	nodeManager := peer.NewNodeManager(app.ctx, app.appstate, &app.config.P2P)
	err = nodeManager.Initialize()

	if err != nil {
		return err
	}

	discoveryManager, err := discovery.NewDiscoveryManager(app.ctx, nodeManager.GetHost(), app.config, app.eventBus)
	err = discoveryManager.Initialize()

	if err != nil {
		return err
	}

	return nil
}
