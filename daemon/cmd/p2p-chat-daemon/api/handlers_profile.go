package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// handleSendMessage handles POST requests to /chat/send
func (h *ApiHandler) handleFriendRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode request body
	var req FriendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	if req.ReceiverPeerId == "" {
		http.Error(w, "Missing 'ReceiverPeerId' in request", http.StatusBadRequest)
		return
	}

	err := h.profileService.SendFriendRequest(req.ReceiverPeerId)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error sending friend request: %v", err), http.StatusInternalServerError)
		return
	}

	// --- Send Success Response ---
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Friend request sent successfully")
}
