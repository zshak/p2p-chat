package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/chat"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/connection"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/profile"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/storage"
)

// StartAPIServer initializes and starts the HTTP API server.
func StartAPIServer(
	ctx context.Context,
	addr string,
	appState *core.AppState,
	bus *bus.EventBus,
	chatService *chat.Service,
	profileService *profile.Service,
	connectionService *connection.Service,
	displayNameRepo storage.DisplayNameRepository,
) (net.Listener, *http.Server, *ApiHandler, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	handler := newAPIHandler(appState, bus, chatService, profileService, connectionService, displayNameRepo)

	mux := http.NewServeMux()

	setupRoutes(mux, handler)

	// Create HTTP server
	server := &http.Server{
		Addr:        listener.Addr().String(),
		Handler:     corsMiddleware(mux),
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

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		origin := r.Header.Get("Origin")

		// Check if the origin is allowed
		if origin == "http://localhost:5173" || origin == "http://localhost:5174" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
