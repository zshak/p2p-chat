package setup

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/appstate"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/chat"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/config"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/discovery"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core/events"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/peer"
	uiapi "p2p-chat-daemon/cmd/p2p-chat-daemon/ui-api"
	"syscall"
	"time"
)

type Application struct {
	ctx         context.Context
	eventBus    *bus.EventBus
	config      *config.Config
	appstate    *core.AppState
	chatService *chat.Service
	cancel      context.CancelFunc
	server      *http.Server
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

	chatHandler := chat.NewProtocolHandler(appState)

	_, server, err := uiapi.StartAPIServer(ctx, cfg.API.ListenAddr, appState, eventbus, chatHandler)
	eventbus.PublishAsync(events.ApiStartedEvent{})

	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create api service: %w", err)
	}

	app := &Application{
		ctx:         ctx,
		config:      cfg,
		appstate:    appState,
		eventBus:    eventbus,
		chatService: chatHandler,
		cancel:      cancel,
		server:      server,
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
	host, err := nodeManager.Initialize()
	if err != nil {
		return err
	}

	app.eventBus.PublishAsync(events.HostInitializedEvent{Host: host})
	nodeManager.LogNodeDetails()

	if err != nil {
		return err
	}
	discoveryManager, err := discovery.NewDiscoveryManager(app.ctx, host, app.config, app.eventBus)
	err = discoveryManager.Initialize()
	app.eventBus.PublishAsync(events.SetupCompletedEvent{})

	app.chatService.Register()

	if err != nil {
		return err
	}

	return nil
}

func (app *Application) WaitForShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Printf("\r- Received signal %s. Triggering shutdown...", sig)
		app.cancel()
	}()

	/* Block until termination signal received */
	<-app.ctx.Done()
}

func (app *Application) Stop() {
	/* --- Graceful Shutdown Sequence --- */
	log.Println("Shutting down daemon...")

	/* Shutdown API server */
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	log.Println("Shutting down API server...")
	if err := app.server.Shutdown(shutdownCtx); err != nil {
		log.Printf("API server shutdown error: %v", err)
	} else {
		log.Println("API server stopped.")
	}

	/* Shutdown node */
	if app.appstate.Node != nil {
		log.Println("Closing libp2p node...")
		if err := (*app.appstate.Node).Close(); err != nil {
			log.Printf("Error closing libp2p node: %v", err)
		} else {
			log.Println("Libp2p node closed.")
		}
	}

	log.Println("Daemon shut down gracefully.")
}
