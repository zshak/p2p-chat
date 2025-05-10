package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// handleSendMessage handles POST requests to /profile/friend/request
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

	err := h.profileService.SendFriendRequest(req.ReceiverPeerId)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error sending friend request: %v", err), http.StatusInternalServerError)
		return
	}

	// --- Send Success Response ---
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Friend request sent successfully")
}

// handleFriendRequestResponse handles PATCH requests to /profile/friend/response
func (h *ApiHandler) handleFriendRequestResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode request body
	var req FriendRequestResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	err := h.profileService.RespondToFriendRequest(req.PeerId, req.IsAccepted)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error responging to friend request: %v", err), http.StatusInternalServerError)
		return
	}

	// --- Send Success Response ---
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Responded To Friend Request successfully")
}
