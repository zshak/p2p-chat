package api

import (
	// Shared core types
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	// Service dependencies
	"p2p-chat-daemon/cmd/p2p-chat-daemon/identity"
	// Import host type directly
	"context"
	"github.com/libp2p/go-libp2p/core/host"
	"log"
	"time"
)

// apiHandler holds dependencies needed by the API handlers.
type apiHandler struct {
	appState         *core.AppState         // For overall status reporting
	idService        *identity.Service      // For key operations
	hostProvider     func() host.Host       // Function to get the host instance (when ready)
	p2pReadyNotifier func() <-chan struct{} // To wait for P2P service readiness (host availability)
}

// newAPIHandler creates a new handler instance with injected dependencies.
// The hostProvider function allows lazy retrieval of the host.
func newAPIHandler(
	appState *core.AppState,
	idSvc *identity.Service,
	hostProvider func() host.Host, // Pass a function to get the host
	p2pReadyNotifier func() <-chan struct{},
) *apiHandler {
	if appState == nil || idSvc == nil || hostProvider == nil || p2pReadyNotifier == nil {
		panic("nil dependencies provided to apiHandler")
	}
	return &apiHandler{
		appState:         appState,
		idService:        idSvc,
		hostProvider:     hostProvider,
		p2pReadyNotifier: p2pReadyNotifier,
	}
}

// getP2PHost safely retrieves the libp2p Host using the provider function, waiting if necessary.
// Returns nil if the host doesn't become available or context is cancelled.
func (h *apiHandler) getP2PHost(ctx context.Context) host.Host {
	node := h.hostProvider() // Try getting it immediately
	if node != nil {
		return node
	}

	// Wait for P2P ready signal if host not available yet
	log.Println("API Handler: P2P Host not ready, waiting for ready signal...")
	select {
	case <-h.p2pReadyNotifier():
		log.Println("API Handler: P2P Ready signal received.")
		node = h.hostProvider() // Try getting it again
		if node == nil {
			log.Println("API Handler: ERROR - P2P Ready signal received but host still nil.")
		}
		return node
	case <-ctx.Done():
		log.Printf("API Handler: Waiting for P2P Host cancelled: %v", ctx.Err())
		return nil
	// Add a reasonable timeout for waiting specifically for the host
	case <-time.After(15 * time.Second): // e.g., 15 seconds
		log.Println("API Handler: Timed out waiting for P2P Host.")
		node = h.hostProvider() // Check one last time
		return node
	}
}
