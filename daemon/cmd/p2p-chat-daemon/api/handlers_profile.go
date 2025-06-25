package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// handleSendMessage handles POST requests to /profile/friends/request
func (h *ApiHandler) handleFriendRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "HEEEEEEEEEEEEEEEEEEEEEEEEREEEEEEEEEEEEEEE")
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
		http.Error(w, fmt.Sprintf("Error sending friends request: %v", err), http.StatusInternalServerError)
		return
	}

	// --- Send Success Response ---
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Friend request sent successfully")
}

// handleFriendRequestResponse handles PATCH requests to /profile/friends/response
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
		http.Error(w, fmt.Sprintf("Error responging to friends request: %v", err), http.StatusInternalServerError)
		return
	}

	// --- Send Success Response ---
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Responded To Friend Request successfully")
}

func (h *ApiHandler) handleGetFriends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	res, err := h.profileService.GetFriends()

	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting friends: %v", err), http.StatusInternalServerError)
		return
	}

	responseBytes, err := json.Marshal(res)
	if err != nil {
		log.Printf("API Handler: Error marshalling friends data to JSON: %v", err)
		http.Error(w, "Failed to prepare friends list response", http.StatusInternalServerError)
		return
	}

	// --- Send Success Response -
	w.Write(responseBytes)
	w.WriteHeader(http.StatusOK)
}
