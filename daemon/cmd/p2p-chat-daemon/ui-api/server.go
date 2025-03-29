package ui_api

import (
	"context" // Needed for background operations and cancellation
	"errors"  // For creating standard error values (like http.ErrServerClosed)
	"fmt"
	"log"
	"net"
	"net/http" // HTTP server elements
	"time"     // For shutdown timeout

	"github.com/libp2p/go-libp2p/core/host" // To pass the node to handlers
)

// StartAPIServer initializes and starts the HTTP API server in a separate goroutine.
// It takes the parent context, the address to listen on, and the libp2p node.
// It returns the configured http.Server instance (for graceful shutdown) and any setup error.
func StartAPIServer(ctx context.Context, listenAddr string, node host.Host) (net.Listener, *http.Server, error) {
	log.Printf("Configuring API server...")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/id", idHandler(node))
	/* Add other API handlers */

	/* Add UI file server logic here if combining, passing uiDir */
	/* fileServer := http.FileServer(http.Dir(uiDir)) ... */
	/* mux.HandleFunc("/", uiHandler(fileServer, uiDir)) ... */

	/* --- Start Listening --- */
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		/* Optional: Add specific EADDRINUSE error checking here */
		return nil, nil, fmt.Errorf("failed to listen on %s: %w", listenAddr, err)
	}
	/* Successfully listening now, listener.Addr() has the actual address */
	actualAddr := listener.Addr().String()
	log.Printf("API server setup: Listening on actual address: %s", actualAddr)

	/* Create the server instance */
	server := &http.Server{
		Handler: mux,
	}

	/* Start serving requests using the listener in a goroutine */
	go func() {
		log.Printf("API Server accepting connections via Serve() on %s", actualAddr)
		err := server.Serve(listener) /* Use Serve with the existing listener */
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("API server Serve() error: %v", err)
		}
		log.Println("API server Serve() stopped.")
	}()

	/* Optional: Context cancellation listener */
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		log.Println("Shutting down API server from server goroutine...")
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("API server shutdown error (from server goroutine): %v", err)
		}
	}()

	/* Return the created listener and server */
	return listener, server, nil
}
