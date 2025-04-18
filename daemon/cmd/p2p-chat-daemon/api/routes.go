package api

import (
	"net/http"
)

// setupRoutes configures the routes for the API server.
func setupRoutes(mux *http.ServeMux, handler *apiHandler) {
	// Status endpoint
	mux.HandleFunc("/status", handler.handleStatus)

	// Setup endpoints
	mux.HandleFunc("/setup/create-key", handler.handleCreateKey)
	mux.HandleFunc("/setup/unlock-key", handler.handleUnlockKey)

	mux.HandleFunc("/chat/send", handler.handleSendMessage)

}
