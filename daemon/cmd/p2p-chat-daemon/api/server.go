package api

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/chat"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/connection"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/profile"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/storage"
	"strings"
)

//go:embed dist/*
var uiFiles embed.FS

// StartAPIServer initializes and starts the HTTP API server with embedded UI.
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
	setupUIRoutes(mux)

	server := &http.Server{
		Addr:        listener.Addr().String(),
		Handler:     corsMiddleware(mux),
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}

	go func() {
		log.Printf("Unified server (UI + API) starting on %s", server.Addr)
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Unified server error: %v", err)
			appState.Mu.Lock()
			appState.State = core.StateError
			appState.LastError = fmt.Errorf("unified server failed: %w", err)
			appState.Mu.Unlock()
		}
		log.Println("Unified server stopped serving.")
	}()

	return listener, server, handler, nil
}

// setupUIRoutes configures the UI routes for serving the embedded React app
func setupUIRoutes(mux *http.ServeMux) {
	uiFS, err := fs.Sub(uiFiles, "dist")
	if err != nil {
		log.Printf("Warning: Could not access embedded UI files: %v", err)
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				http.Error(w, "UI not available in this build", http.StatusNotFound)
				return
			}
			http.NotFound(w, r)
		})
		return
	}

	fileServer := http.FileServer(http.FS(uiFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/")

		if path != "/" && !strings.Contains(path, ".") {
			if _, err := fs.Stat(uiFS, strings.TrimPrefix(path, "/")); err != nil {
				r.URL.Path = "/"
			}
		}

		if strings.HasSuffix(path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		} else if strings.HasSuffix(path, ".html") {
			w.Header().Set("Content-Type", "text/html")
		}

		fileServer.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin == "" ||
			origin == "http://localhost:5173" ||
			origin == "http://localhost:5174" ||
			strings.HasPrefix(origin, "http://127.0.0.1:") ||
			strings.HasPrefix(origin, "http://localhost:") {
			w.Header().Set("Access-Control-Allow-Origin", "*")
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
