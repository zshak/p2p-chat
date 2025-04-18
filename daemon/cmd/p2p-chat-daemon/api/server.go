package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"p2p-chat-daemon/cmd/config"
	"time"

	"p2p-chat-daemon/cmd/p2p-chat-daemon/identity"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	// Import only host type, not the whole p2p service
	"github.com/libp2p/go-libp2p/core/host"
)

// Service manages the HTTP API endpoint.
type Service struct {
	ctx        context.Context
	cfg        *config.APIConfig
	httpServer *http.Server
	listener   net.Listener
	handler    *apiHandler // The handler holds the dependencies now
	// Dependencies are passed to the handler during creation
	// appState      *core.AppState      // No longer needed directly here
	// idService     *identity.Service   // No longer needed directly here
	// p2pService    *p2p.Service        // Removed
	// hostProvider added to handler instead
}

// NewService creates a new API Service.
func NewService(
	ctx context.Context,
	cfg *config.APIConfig,
	appState *core.AppState, // Still needed for handler
	idSvc *identity.Service, // Still needed for handler
	hostProvider func() host.Host, // Pass host provider func
	p2pReadyNotifier func() <-chan struct{}, // Pass ready notifier func
) (*Service, error) {
	// Nil checks for dependencies needed by handler
	if cfg == nil || appState == nil || idSvc == nil || hostProvider == nil || p2pReadyNotifier == nil {
		return nil, errors.New("api service requires non-nil config, appState, idService, hostProvider, and p2pReadyNotifier")
	}

	// Create handler instance, injecting dependencies
	handler := newAPIHandler(appState, idSvc, hostProvider, p2pReadyNotifier)

	return &Service{
		ctx:     ctx,
		cfg:     cfg,
		handler: handler, // Store the configured handler
	}, nil
}

// SetP2PHost is no longer needed here, the provider function handles it.

// Start starts the HTTP server.
func (s *Service) Start() error {
	if s.httpServer != nil {
		return errors.New("API service already started")
	}
	log.Println("API Service: Starting...")

	mux := http.NewServeMux()
	setupRoutes(mux, s.handler) // Pass the handler instance

	// Listen
	listener, err := net.Listen("tcp", s.cfg.ListenAddr)
	// ... (rest of Listen and WriteAPIInfoFile logic) ...
	if err != nil {
		return fmt.Errorf("API listen failed on %s: %w", s.cfg.ListenAddr, err)
	}
	s.listener = listener
	actualAddr := listener.Addr().String()
	log.Printf("API Service: Listening on %s", actualAddr)

	s.httpServer = &http.Server{
		Addr:        actualAddr,
		Handler:     mux,
		BaseContext: func(_ net.Listener) context.Context { return s.ctx },
	}

	// Start server in background
	go func() {
		log.Printf("API Service: Starting server goroutine for %s", s.httpServer.Addr)
		if err := s.httpServer.Serve(s.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("API Service: Server error: %v", err)
			// Signal fatal error through AppState? Could lead to tight coupling.
			// Consider returning error channel or relying on context cancellation.
			// For now, just log. App coordination might need refinement.
		}
		log.Println("API Service: Server stopped.")
	}()

	log.Println("API Service: Started.")
	return nil
}

// Stop gracefully shuts down the HTTP server.
func (s *Service) Stop() error {
	// ... (Stop logic remains the same) ...
	log.Println("API Service: Stopping...")
	if s.httpServer == nil {
		log.Println("API Service: Already stopped.")
		return nil
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := s.httpServer.Shutdown(shutdownCtx)
	if err != nil {
		log.Printf("API Service: Shutdown error: %v", err)
	}
	s.httpServer = nil
	log.Println("API Service: Stopped.")
	return err
}
