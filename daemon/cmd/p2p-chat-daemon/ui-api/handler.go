package ui_api

import (
	"p2p-chat-daemon/cmd/p2p-chat-daemon/chat"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
)

// apiHandler holds dependencies needed by the API handlers
type apiHandler struct {
	appState    *core.AppState
	eventBus    *bus.EventBus
	chatService *chat.Service
}

// newAPIHandler creates a new handler instance.
func newAPIHandler(appState *core.AppState, eventBus *bus.EventBus, chatService *chat.Service) *apiHandler {
	if appState == nil {
		panic("appState cannot be nil for apiHandler")
	}

	return &apiHandler{
		appState:    appState,
		eventBus:    eventBus,
		chatService: chatService,
	}
}
