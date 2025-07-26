package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// handleSendMessage handles POST requests to /chat/send
func (h *ApiHandler) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	var request WsDirectMessageRequestPayload
	json.Unmarshal(req.Payload, &request)

	if request.TargetPeerID == "" || request.Message == "" {
		http.Error(w, "Missing 'target_peer_id' or 'message' in request", http.StatusBadRequest)
		return
	}

	err := h.chatService.SendMessage(request.TargetPeerID, request.Message)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error sending message: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Message sent successfully")
}

// handleGetMessages handles GET requests to /chat/messages
func (h *ApiHandler) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GetChatMessagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	messages, err := h.chatService.GetMessages(req.PeerId)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting message: %v", err), http.StatusInternalServerError)
		return
	}

	responseBytes, err := json.Marshal(messages)
	if err != nil {
		log.Printf("API Handler: Error marshalling chat messages to JSON: %v", err)
		http.Error(w, "Failed to prepare chat messages response", http.StatusInternalServerError)
	}

	w.Write(responseBytes)
}
