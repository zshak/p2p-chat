package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	. "p2p-chat-daemon/cmd/p2p-chat-daemon/storage"
)

// SetDisplayNameRequest represents the request to set a display name
type SetDisplayNameRequest struct {
	EntityID    string `json:"entity_id"`
	EntityType  string `json:"entity_type"`
	DisplayName string `json:"display_name"`
}

// GetDisplayNameRequest represents the request to get a display name
type GetDisplayNameRequest struct {
	EntityID   string `json:"entity_id"`
	EntityType string `json:"entity_type"`
}

// GetDisplayNameResponse represents the response with fallback logic
type GetDisplayNameResponse struct {
	EntityID     string `json:"entity_id"`
	EntityType   string `json:"entity_type"`
	DisplayName  string `json:"display_name"`
	IsCustomName bool   `json:"is_custom_name"`
}

// DeleteDisplayNameRequest represents the request to delete a display name
type DeleteDisplayNameRequest struct {
	EntityID   string `json:"entity_id"`
	EntityType string `json:"entity_type"`
}

// formatEntityIdFallback creates a user-friendly fallback name for entities
func formatEntityIdFallback(entityID, entityType string) string {
	if entityID == "" {
		return "Unknown"
	}

	if entityType == "group" {
		return "Group Chat"
	}

	if len(entityID) >= 8 {
		first2 := entityID[:2]
		last6 := entityID[len(entityID)-6:]
		return fmt.Sprintf("%s*%s", first2, last6)
	}

	return entityID
}

// handleSetDisplayName handles POST requests to /profile/display-name
func (h *ApiHandler) handleSetDisplayName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SetDisplayNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.EntityID == "" || req.EntityType == "" || req.DisplayName == "" {
		http.Error(w, "entity_id, entity_type, and display_name are required", http.StatusBadRequest)
		return
	}

	if req.EntityType != "friend" && req.EntityType != "group" {
		http.Error(w, "entity_type must be 'friend' or 'group'", http.StatusBadRequest)
		return
	}

	displayName := DisplayName{
		EntityID:    req.EntityID,
		EntityType:  req.EntityType,
		DisplayName: req.DisplayName,
	}

	err := h.displayNameRepo.Store(r.Context(), displayName)
	if err != nil {
		log.Printf("API Handler: Error storing display name: %v", err)
		http.Error(w, fmt.Sprintf("Failed to store display name: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Display name set successfully")
}

// handleGetDisplayName handles POST requests to /profile/display-name/get
// This function NEVER returns 404 - it always provides a fallback display name
func (h *ApiHandler) handleGetDisplayName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GetDisplayNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.EntityID == "" || req.EntityType == "" {
		http.Error(w, "entity_id and entity_type are required", http.StatusBadRequest)
		return
	}

	displayName, err := h.displayNameRepo.GetByEntity(r.Context(), req.EntityID, req.EntityType)

	var response GetDisplayNameResponse
	response.EntityID = req.EntityID
	response.EntityType = req.EntityType

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			response.DisplayName = formatEntityIdFallback(req.EntityID, req.EntityType)
			response.IsCustomName = false
			log.Printf("API Handler: No custom display name found for %s %s, using fallback: %s", req.EntityType, req.EntityID, response.DisplayName)
		} else {
			log.Printf("API Handler: Database error getting display name for %s %s: %v, using fallback", req.EntityType, req.EntityID, err)
			response.DisplayName = formatEntityIdFallback(req.EntityID, req.EntityType)
			response.IsCustomName = false
		}
	} else {
		response.DisplayName = displayName.DisplayName
		response.IsCustomName = true
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("API Handler: Error marshalling display name response: %v", err)
		http.Error(w, "Failed to prepare response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

// handleDeleteDisplayName handles DELETE requests to /profile/display-name/delete
func (h *ApiHandler) handleDeleteDisplayName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeleteDisplayNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.EntityID == "" || req.EntityType == "" {
		http.Error(w, "entity_id and entity_type are required", http.StatusBadRequest)
		return
	}

	err := h.displayNameRepo.Delete(r.Context(), req.EntityID, req.EntityType)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			// No display name to delete - this is fine, return success
			log.Printf("API Handler: No display name to delete for %s %s", req.EntityType, req.EntityID)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "No display name to delete (already using default)")
			return
		}
		log.Printf("API Handler: Error deleting display name: %v", err)
		http.Error(w, fmt.Sprintf("Failed to delete display name: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Display name deleted successfully")
}
