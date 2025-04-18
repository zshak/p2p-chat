package main // Or move to its own 'app' package

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"p2p-chat-daemon/cmd/config"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/api"
	"sync"
	"syscall"
	"time"

	// Internal packages
	"p2p-chat-daemon/cmd/p2p-chat-daemon/identity"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/p2p" // Import the refactored p2p package

	"github.com/libp2p/go-libp2p/core/event" // For event bus
	"github.com/libp2p/go-libp2p/core/host"
)

// App coordinates the lifecycle and dependencies of the application services.
type App struct {
	ctx       context.Context
	cancel    context.CancelFunc
	cfg       *config.Config
	appState  *core.AppState // Shared observable state, primarily for API
	idService *identity.Service
	// P2P Components (initialized later)
	p2pNode      *p2p.Node
	p2pDHT       *p2p.DHT
	p2pDiscovery *p2p.Discovery
	p2pChat      *p2p.ChatService
	// API Service (initialized early)
	apiSvc *api.Service
	// Lifecycle
	wg           sync.WaitGroup // Waits for key tasks like P2P starter & logger
	p2pReadyChan chan struct{}  // Signals when P2P host is ready
}

// NewApp creates the application coordinator and its essential services.
// P2P components are created later when the key is available.
func NewApp(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())
	appState := core.NewAppState(cfg.P2P.PrivateKeyPath)
	p2pReadyChan := make(chan struct{})

	idSvc, err := identity.NewService(&cfg.P2P)
	if err != nil { /* ... */
	}

	// Create the application struct FIRST
	app := &App{
		ctx:          ctx,
		cancel:       cancel,
		cfg:          cfg,
		appState:     appState,
		idService:    idSvc,
		p2pReadyChan: p2pReadyChan,
		// P2P and API services initialized later or passed nil
	}

	// --- Create Host Provider Function ---
	// This closure captures the 'app' pointer.
	hostProvider := func() host.Host {
		// Add locking if access needs to be thread-safe,
		// though usually called after initialization is complete.
		if app.p2pNode == nil {
			return nil
		}
		return app.p2pNode.Host()
	}
	// --- ----------------------------- ---

	// Create API Service, injecting the host provider
	apiSvc, err := api.NewService(ctx, &cfg.API, appState, idSvc, hostProvider, func() <-chan struct{} { return p2pReadyChan })
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create api service: %w", err)
	}
	app.apiSvc = apiSvc // Assign created service back to app

	return app, nil
}

// Start initiates the application services that can start immediately.
func (a *App) Start() error {
	log.Println("App: Starting services...")

	// 1. Start API Service (must be first for setup)
	if err := a.apiSvc.Start(); err != nil {
		a.cancel() // Ensure context cancelled on fatal startup error
		return fmt.Errorf("failed to start API service: %w", err)
	}

	// 2. Start background task to wait for key and then initialize P2P stack
	a.wg.Add(1)
	go a.waitForKeyAndStartP2PServices()

	// 3. Set initial state based on key file presence
	a.checkInitialKeyState()

	log.Println("App: Services initiated.")
	return nil
}

// Stop gracefully shuts down the application services in the correct order.
func (a *App) Stop() error {
	log.Println("App: Stopping services...")

	// 1. Signal all components using the context to stop
	a.cancel()

	// 2. Stop services in reverse dependency order (API depends on P2P indirectly)
	var firstErr error
	if a.apiSvc != nil {
		if err := a.apiSvc.Stop(); err != nil {
			log.Printf("App: API service stop error: %v", err)
			firstErr = err // Record first error
		}
	}
	if a.p2pDiscovery != nil {
		a.p2pDiscovery.Stop() // Stops discovery loops
	}
	// Chat service doesn't have explicit stop, relies on host closing streams
	if a.p2pDHT != nil {
		if err := a.p2pDHT.Close(); err != nil {
			log.Printf("App: DHT close error: %v", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	if a.p2pNode != nil {
		if err := a.p2pNode.Close(); err != nil {
			log.Printf("App: Node close error: %v", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	// 3. Wait for essential background goroutines managed by App
	log.Println("App: Waiting for background tasks...")
	a.wg.Wait()

	log.Println("App: Services stopped.")
	return firstErr
}

// WaitForShutdown sets up OS signal handling and blocks until shutdown is initiated.
func (a *App) WaitForShutdown() {
	log.Println("App: Waiting for shutdown signal (Ctrl+C)...")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Block waiting for either a signal or context cancellation
	select {
	case sig := <-c:
		log.Printf("\r- Signal %s received. Initiating shutdown...", sig)
	case <-a.ctx.Done():
		log.Println("App: Shutdown initiated by internal context cancellation.")
	}
	signal.Stop(c) // Stop listening once signal is received or context cancelled
	a.cancel()     // Ensure context is cancelled regardless of trigger
	log.Println("App: Shutdown signal processed.")
}

// --- Internal Coordination Methods ---

// checkInitialKeyState sets the initial state based on key file presence.
func (a *App) checkInitialKeyState() {
	a.appState.Mu.Lock()
	defer a.appState.Mu.Unlock()
	if a.appState.State == core.StateInitializing {
		if a.idService.KeyExists() { // Use identity service method
			a.appState.State = core.StateWaitingForPassword
			log.Printf("App State: Key file found at %s. Waiting for password via API.", a.appState.KeyPath)
		} else {
			a.appState.State = core.StateWaitingForKey
			log.Printf("App State: Key file not found at %s. Waiting for key setup via API.", a.appState.KeyPath)
		}
	}
}

// waitForKeyAndStartP2PServices is the goroutine that initializes the P2P stack.
func (a *App) waitForKeyAndStartP2PServices() {
	defer a.wg.Done() // Signal this goroutine is done when Stop() waits
	log.Println("P2P Starter: Waiting for key ready signal...")

	select {
	case <-a.appState.KeyReadyChan:
		log.Println("P2P Starter: Key ready signal received.")
	case <-a.ctx.Done():
		log.Println("P2P Starter: Shutdown signal received before key was ready.")
		return
	}

	// Get the key from the identity service
	privKey := a.idService.GetPrivateKey()
	if privKey == nil {
		err := errors.New("key signal received but private key is not available in identity service")
		a.handleP2PStartupError(err)
		return
	}

	// Update state *before* long-running P2P init
	a.appState.Mu.Lock()
	if a.appState.State == core.StateShuttingDown || a.appState.State == core.StateError {
		log.Printf("P2P Starter: Aborting P2P start due to current state (%s).", a.appState.State)
		a.appState.Mu.Unlock()
		return
	}
	a.appState.State = core.StateInitializingP2P
	a.appState.Mu.Unlock()

	// --- Create P2P Components ---
	var err error
	log.Println("P2P Starter: Creating P2P Node...")
	a.p2pNode, err = p2p.NewNode(&a.cfg.P2P, privKey) // Pass only needed config
	if err != nil {
		a.handleP2PStartupError(fmt.Errorf("node creation failed: %w", err))
		return
	}

	log.Println("P2P Starter: Creating DHT...")
	a.p2pDHT, err = p2p.NewDHT(a.ctx, &a.cfg.P2P, a.p2pNode.Host())
	if err != nil {
		log.Printf("P2P Starter: WARN - DHT setup failed: %v. Discovery may be limited.", err)
		// Non-fatal for now, clear the DHT pointer
		a.p2pDHT = nil
	} else {
		a.appState.Mu.Lock()
		a.appState.Dht = a.p2pDHT.Instance() // Store the underlying DHT instance for peerSource
		a.appState.Mu.Unlock()
		log.Println("P2P Starter: DHT Created.")
	}

	log.Println("P2P Starter: Creating Discovery Service...")
	a.p2pDiscovery, err = p2p.NewDiscovery(a.ctx, &a.cfg.P2P, a.p2pNode.Host(), a.p2pDHT) // Pass DHT component
	if err != nil {
		// This might be more critical if DHT failed?
		log.Printf("P2P Starter: WARN - Failed to create discovery component: %v", err)
	} else {
		a.p2pDiscovery.Start() // Start discovery loops (runs background tasks)
		log.Println("P2P Starter: Discovery Service Started.")
	}

	log.Println("P2P Starter: Creating Chat Service...")
	a.p2pChat = p2p.NewChatService(a.p2pNode.Host())
	a.p2pChat.RegisterHandler()
	log.Println("P2P Starter: Chat Handler Registered.")

	// --- Start reachability logger ---
	a.wg.Add(1)
	go a.listenForReachabilityEvents()

	// --- Signal P2P Ready ---
	close(a.p2pReadyChan) // Signal API (and potentially others)

	// --- Final State Update ---
	a.appState.Mu.Lock()
	if a.appState.State != core.StateShuttingDown && a.appState.State != core.StateError {
		a.appState.State = core.StateRunning
		log.Println("P2P Starter: P2P Stack is now Running.")
	}
	a.appState.Mu.Unlock()
}

// handleP2PStartupError logs fatal errors during P2P init and triggers shutdown.
func (a *App) handleP2PStartupError(err error) {
	log.Printf("App P2P Starter: FATAL - %v", err)
	a.appState.Mu.Lock()
	a.appState.LastError = err
	a.appState.State = core.StateError
	a.appState.Mu.Unlock()
	a.cancel() // Trigger global shutdown
}

// listenForReachabilityEvents subscribes to and logs AutoNAT status changes.
func (a *App) listenForReachabilityEvents() {
	defer a.wg.Done() // Managed by the App's WaitGroup

	// Wait for node to be ready (with timeout and context check)
	var node host.Host
	for i := 0; i < 15; i++ { // Wait up to ~3 seconds
		if a.p2pNode != nil {
			node = a.p2pNode.Host()
			if node != nil {
				break
			}
		}
		select {
		case <-a.ctx.Done():
			return
		case <-time.After(200 * time.Millisecond):
		}
	}
	if node == nil {
		log.Println("Reachability Logger: Host never became available. Exiting.")
		return
	}

	subReachability, err := node.EventBus().Subscribe(new(event.EvtLocalReachabilityChanged))
	if err != nil {
		log.Printf("Reachability Logger: ERROR - Failed to subscribe: %v", err)
		return
	}
	defer subReachability.Close()
	log.Println("Reachability Logger: Subscribed and listening for AutoNAT events.")

	for {
		select {
		case <-a.ctx.Done():
			log.Println("Reachability Logger: Stopping.")
			return
		case ev, ok := <-subReachability.Out():
			if !ok {
				log.Println("Reachability Logger: Subscription channel closed.")
				return
			}
			reachabilityEvent, ok := ev.(event.EvtLocalReachabilityChanged)
			if !ok {
				continue
			}
			log.Printf("===== AutoNAT Status Update: Reachability = %s =====", reachabilityEvent.Reachability)
		}
	}
}
