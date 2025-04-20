package ui_api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/chat"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
)

// StartAPIServer initializes and starts the HTTP API server.
func StartAPIServer(ctx context.Context, addr string, appState *core.AppState, bus *bus.EventBus, chatService *chat.Service) (net.Listener, *http.Server, *ApiHandler, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	handler := newAPIHandler(appState, bus, chatService)

	mux := http.NewServeMux()

	setupRoutes(mux, handler)

	// Create HTTP server
	server := &http.Server{
		Addr:        listener.Addr().String(),
		Handler:     mux,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}

	go func() {
		log.Printf("API server starting on %s", server.Addr)
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("API server error: %v", err)
			appState.Mu.Lock()
			appState.State = core.StateError
			appState.LastError = fmt.Errorf("API server failed: %w", err)
			appState.Mu.Unlock()
		}
		log.Println("API server stopped serving.")
	}()

	return listener, server, handler, nil
}
