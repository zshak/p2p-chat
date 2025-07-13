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
func (h *ApiHandler) handleSendGroupMessage(w http.ResponseWriter, r *http.Request) {
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

func (h *ApiHandler) handleGetGroupMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode request body
	var req GetGroupChatMessagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	messages, err := h.chatService.GetGroupMessages(req.GroupId)

	if err != nil {
		log.Printf("API Handler: Error getting group chat messages: %v", err)
		http.Error(w, fmt.Sprintf("Error getting group chat messages: %v", err), http.StatusInternalServerError)
		return
	}

	responseBytes, err := json.Marshal(messages)
	if err != nil {
		log.Printf("API Handler: Error marshalling group chat messages to JSON: %v", err)
		http.Error(w, "Failed to prepare group chat messages response", http.StatusInternalServerError)
	}

	// --- Send Success Response -
	w.Write(responseBytes)
}

func (h *ApiHandler) handleGetGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groups, err := h.chatService.GetGroups()

	if err != nil {
		log.Printf("API Handler: Error getting group chats: %v", err)
		http.Error(w, fmt.Sprintf("Error getting group chats: %v", err), http.StatusInternalServerError)
		return
	}

	responseBytes, err := json.Marshal(groups)
	if err != nil {
		log.Printf("API Handler: Error marshalling group chats to JSON: %v", err)
		http.Error(w, "Failed to prepare group chats response", http.StatusInternalServerError)
	}

	// --- Send Success Response -
	w.Write(responseBytes)
}
