package api

import (
	"net/http"
)

// setupRoutes configures the routes for the API server.
func setupRoutes(mux *http.ServeMux, handler *ApiHandler) {
	mux.HandleFunc("/status", handler.handleStatus)

	mux.HandleFunc("/setup/create-key", handler.handleCreateKey)
	mux.HandleFunc("/setup/unlock-key", handler.handleUnlockKey)

	mux.HandleFunc("/chat/send", handler.handleSendMessage)
	mux.HandleFunc("/chat/messages", handler.handleGetMessages)

	mux.HandleFunc("/profile/friend/request", handler.handleFriendRequest)
	mux.HandleFunc("/profile/friend/response", handler.handleFriendRequestResponse)
	mux.HandleFunc("/profile/friends", handler.handleGetFriends)
	mux.HandleFunc("/profile/friendRequests", handler.handleGetFriendRequests)

	mux.HandleFunc("/group-chat", handler.handleCreateGroupChat)
	mux.HandleFunc("/group-chats", handler.handleGetGroups)
	mux.HandleFunc("/group-chat/send", handler.handleSendGroupMessage)
	mux.HandleFunc("/group-chat/messages", handler.handleGetGroupMessages)

	mux.HandleFunc("/ws", handler.handleWebSocket)
}
