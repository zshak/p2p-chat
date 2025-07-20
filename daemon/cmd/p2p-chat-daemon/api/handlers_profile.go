package api

import (
	"encoding/json"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	"log"
	"net/http"
)

// handleSendMessage handles POST requests to /profile/friends/request
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

	// Get friends from profile service
	friends, err := h.profileService.GetFriends()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting friends: %v", err), http.StatusInternalServerError)
		return
	}

	// Enhance friends data with real-time online status and display names
	for i := range friends {
		// Decode peer ID to check online status
		peerID, err := peer.Decode(friends[i].PeerID)
		if err != nil {
			log.Printf("API Handler: Error decoding peer ID %s: %v", friends[i].PeerID, err)
			friends[i].IsOnline = false
			continue
		}

		// Get real-time online status from connection service
		friends[i].IsOnline = h.connectionService.IsOnline(peerID)

		// Get display name for this friend
		displayName, err := h.displayNameRepo.GetByEntity(r.Context(), friends[i].PeerID, "friend")
		if err != nil {
			if err.Error() != "sql: no rows in result set" {
				log.Printf("API Handler: Error getting display name for friend %s: %v", friends[i].PeerID, err)
			}
			// No display name found or error - leave DisplayName empty/nil
			// The frontend will fall back to using the PeerID
		} else {
			// Set the display name if found
			friends[i].DisplayName = displayName.DisplayName
		}
	}

	// Marshal response
	responseBytes, err := json.Marshal(friends)
	if err != nil {
		log.Printf("API Handler: Error marshalling friends data to JSON: %v", err)
		http.Error(w, "Failed to prepare friends list response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

func (h *ApiHandler) handleGetFriendRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	res, err := h.profileService.GetFriendRequests()

	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting friend requests: %v", err), http.StatusInternalServerError)
		return
	}

	responseBytes, err := json.Marshal(res)
	if err != nil {
		log.Printf("API Handler: Error marshalling friend requests data to JSON: %v", err)
		http.Error(w, "Failed to prepare friend requests response", http.StatusInternalServerError)
		return
	}

	// Set content type header
	w.Header().Set("Content-Type", "application/json")

	// Write response
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}
