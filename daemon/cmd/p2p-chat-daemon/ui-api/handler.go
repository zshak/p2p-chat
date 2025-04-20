package ui_api

import (
	"github.com/gorilla/websocket"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/chat"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"sync"
)

// ApiHandler holds dependencies needed by the API handlers
type ApiHandler struct {
	appState    *core.AppState
	eventBus    *bus.EventBus
	chatService *chat.Service
	wsConn      *websocket.Conn
	wsMu        sync.RWMutex
}

// newAPIHandler creates a new handler instance.
func newAPIHandler(appState *core.AppState, eventBus *bus.EventBus, chatService *chat.Service) *ApiHandler {
	if appState == nil {
		panic("appState cannot be nil for apiHandler")
	}

	return &ApiHandler{
		appState:    appState,
		eventBus:    eventBus,
		chatService: chatService,
	}
}
