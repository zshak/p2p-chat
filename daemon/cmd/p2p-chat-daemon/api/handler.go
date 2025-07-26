package api

import (
	"github.com/gorilla/websocket"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/chat"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/connection"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/profile"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/storage"
	"sync"
)

// ApiHandler holds dependencies needed by the API handlers
type ApiHandler struct {
	appState          *core.AppState
	eventBus          *bus.EventBus
	chatService       *chat.Service
	profileService    *profile.Service
	connectionService *connection.Service
	displayNameRepo   storage.DisplayNameRepository
	wsConn            *websocket.Conn
	wsMu              sync.RWMutex
}

// newAPIHandler creates a new handler instance.
func newAPIHandler(
	appState *core.AppState,
	eventBus *bus.EventBus,
	chatService *chat.Service,
	profileService *profile.Service,
	connectionService *connection.Service,
	displayNameRepo storage.DisplayNameRepository,
) *ApiHandler {
	if appState == nil {
		panic("appState cannot be nil for apiHandler")
	}

	return &ApiHandler{
		appState:          appState,
		eventBus:          eventBus,
		chatService:       chatService,
		profileService:    profileService,
		connectionService: connectionService,
		displayNameRepo:   displayNameRepo,
	}
}
