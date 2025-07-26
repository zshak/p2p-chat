package api

import (
	"net/http"
)

// setupRoutes configures the routes for the API server.
func setupRoutes(mux *http.ServeMux, handler *ApiHandler) {
	mux.HandleFunc("/api/status", handler.handleStatus)

	mux.HandleFunc("/api/setup/create-key", handler.handleCreateKey)
	mux.HandleFunc("/api/setup/unlock-key", handler.handleUnlockKey)

	mux.HandleFunc("/api/chat/send", handler.handleSendMessage)
	mux.HandleFunc("/api/chat/messages", handler.handleGetMessages)

	mux.HandleFunc("/api/profile/friend/request", handler.handleFriendRequest)
	mux.HandleFunc("/api/profile/friend/response", handler.handleFriendRequestResponse)
	mux.HandleFunc("/api/profile/friends", handler.handleGetFriends)
	mux.HandleFunc("/api/profile/friendRequests", handler.handleGetFriendRequests)

	mux.HandleFunc("/api/group-chat", handler.handleCreateGroupChat)
	mux.HandleFunc("/api/group-chats", handler.handleGetGroups)
	mux.HandleFunc("/api/group-chat/send", handler.handleSendGroupMessage)
	mux.HandleFunc("/api/group-chat/messages", handler.handleGetGroupMessages)

	mux.HandleFunc("/api/ws", handler.handleWebSocket)

	mux.HandleFunc("/api/profile/display-name", handler.handleSetDisplayName)
	mux.HandleFunc("/api/profile/display-name/get", handler.handleGetDisplayName)
	mux.HandleFunc("/api/profile/display-name/delete", handler.handleDeleteDisplayName)
}
