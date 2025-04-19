package ui_api

import (
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
)

// apiHandler holds dependencies needed by the API handlers
type apiHandler struct {
	appState *core.AppState
}

// newAPIHandler creates a new handler instance.
func newAPIHandler(appState *core.AppState) *apiHandler {
	if appState == nil {
		panic("appState cannot be nil for apiHandler")
	}
	return &apiHandler{
		appState: appState,
	}
}
