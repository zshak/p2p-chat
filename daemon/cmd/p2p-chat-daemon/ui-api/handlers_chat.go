package ui_api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	// Import core types and protocol ID
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"

	"github.com/libp2p/go-libp2p/core/peer" // Need peer package
)

// handleSendMessage handles POST requests to /chat/send
func (h *apiHandler) handleSendMessage(w http.ResponseWriter, r *http.Request) {
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

	// Parse PeerID
	targetPID, err := peer.Decode(req.TargetPeerID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid target PeerID format: %v", err), http.StatusBadRequest)
		return
	}

	// Get node from app state and check if running
	h.appState.Mu.Lock()
	node := h.appState.Node
	state := h.appState.State
	h.appState.Mu.Unlock() // Unlock before potentially blocking P2P operation

	if state != core.StateRunning || node == nil {
		http.Error(w, fmt.Sprintf("Node is not ready (state: %s)", state), http.StatusServiceUnavailable)
		return
	}

	// Don't send to self
	if targetPID == node.ID() {
		http.Error(w, "Cannot send chat message to self", http.StatusBadRequest)
		return
	}

	// --- Open Stream to Peer ---
	log.Printf("Chat API: Attempting to open stream to %s for protocol %s", targetPID.ShortString(), core.ChatProtocolID)

	// Use a timeout for stream opening
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Increased timeout for P2P ops
	defer cancel()

	stream, err := node.NewStream(ctx, targetPID, core.ChatProtocolID)
	if err != nil {
		log.Printf("Chat API: Failed to open stream to %s: %v", targetPID.ShortString(), err)
		http.Error(w, fmt.Sprintf("Failed to connect/open stream to peer %s: %v", targetPID.ShortString(), err), http.StatusNotFound) // 404 might indicate peer not found/reachable
		return
	}
	log.Printf("Chat API: Stream opened successfully to %s", targetPID.ShortString())

	// --- Send Message ---
	writer := bufio.NewWriter(stream)
	_, err = writer.WriteString(req.Message + "\n") // Add newline delimiter
	if err == nil {
		err = writer.Flush() // Ensure data is sent
	}

	if err != nil {
		log.Printf("Chat API: Failed to write message to %s: %v", targetPID.ShortString(), err)
		stream.Reset() // Reset stream on write error
		http.Error(w, fmt.Sprintf("Failed to send message: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Chat API: Message sent successfully to %s", targetPID.ShortString())

	// --- Close Stream (optional, good practice for simple request/response) ---
	// Closing the stream signals the other side we're done writing.
	// Our simple receiver closes after reading one line anyway.
	err = stream.Close()
	if err != nil {
		log.Printf("Chat API: Error closing stream to %s: %v", targetPID.ShortString(), err)
		// Continue anyway, message was sent
	}

	// --- Send Success Response ---
	w.WriteHeader(http.StatusOK)
	// Optionally return the message sent or just a simple success
	fmt.Fprintf(w, "Message sent to %s", targetPID.ShortString())
}
