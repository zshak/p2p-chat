package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// handleCreateGroupChat handles POST requests to /group-chat
func (h *ApiHandler) handleCreateGroupChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode request body
	var req CreateGroupChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	err := h.chatService.CreateGroup(req.MemberPeerIds, req.ChatName)
	if err != nil {
		log.Printf("API Handler: Error creating group chat: %v", err)
		http.Error(w, fmt.Sprintf("Error creating group chat: %v", err), http.StatusInternalServerError)
		return
	}

	// --- Send Success Response ---
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Group chat created successfully")
}

// handleCreateGroupChat handles POST requests to /group-chat
func (h *ApiHandler) handelSendGroupMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode request body
	var req SendGroupChatMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	err := h.chatService.SendGroupMessage(req.GroupId, req.Message)
	if err != nil {
		log.Printf("API Handler: Error sending group chat message: %v", err)
		http.Error(w, fmt.Sprintf("Error sending group chat message: %v", err), http.StatusInternalServerError)
		return
	}

	// --- Send Success Response ---
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Group chat message sent successfully")
}
