package ui_api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// handleSendMessage handles POST requests to /chat/send
func (h *ApiHandler) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode request body
	var req ChatMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	if req.TargetPeerID == "" || req.Message == "" {
		http.Error(w, "Missing 'target_peer_id' or 'message' in request", http.StatusBadRequest)
		return
	}

	err := h.chatService.SendMessage(req.TargetPeerID, req.Message)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error sending message: %v", err), http.StatusInternalServerError)
		return
	}

	// --- Send Success Response ---
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Message sent successfully")
}
