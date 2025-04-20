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
	"p2p-chat-daemon/cmd/p2p-chat-daemon/storage"
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
	messageRepo storage.MessageRepository
}

func NewApplication(cfg *config.Config) (*Application, error) {
	ctx, cancel := context.WithCancel(context.Background())
	appState := core.NewAppState(cfg.P2P.PrivateKeyPath)

	eventbus := bus.NewEventBus()
	appStateObs, err := appstate.NewConsumer(appState, eventbus, ctx)

	if err != nil {
		cancel()
		log.Fatal("app state observer startup failed", err)
	}
	appStateObs.Start()

	db, err := storage.NewDB(cfg)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	msgRepo, err := storage.NewSQLiteMessageRepository(db)
	if err != nil {
		db.Close()
		cancel()
		return nil, fmt.Errorf("failed to create message repository: %w", err)
	}

	chatHandler := chat.NewProtocolHandler(appState, eventbus)

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
		messageRepo: msgRepo,
	}

	return app, nil
}

func (app *Application) Start() error {
	keyReadyChan := make(chan struct{})
	consumer, err := NewConsumer(app.appstate, app.eventBus, keyReadyChan, app.ctx)

	if err != nil {
		return err
	}

	consumer.Start()

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

	chatCons, err := chat.NewConsumer(app.appstate, app.eventBus, app.messageRepo, app.ctx)
	if err != nil {
		log.Println("Failed to create chat consumer")
		return err
	}

	go chatCons.Start()
	go app.chatService.Register()

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
