package ui_api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core/events"

	"p2p-chat-daemon/cmd/p2p-chat-daemon/identity"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
)

// handleCreateKey handles the POST /setup/create-key endpoint.
func (h *apiHandler) handleCreateKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.appState.Mu.Lock()
	if h.appState.State != core.StateWaitingForKey {
		stateStr := h.appState.State.String()
		h.appState.Mu.Unlock()
		http.Error(w, fmt.Sprintf("Invalid state (%s) for creating key", stateStr), http.StatusConflict)
		return
	}
	keyPath := h.appState.KeyPath
	h.appState.Mu.Unlock()

	var req SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Password == "" {
		http.Error(w, "Password cannot be empty", http.StatusBadRequest)
		return
	}

	privKey, err := identity.GenerateAndSaveEncryptedKey(keyPath, []byte(req.Password))
	if err != nil {
		h.eventBus.PublishAsync(events.KeyGenerationFailedEvent{Err: err})
		http.Error(w, fmt.Sprintf("Failed to create key: %v", err), http.StatusInternalServerError)
		return
	}

	h.eventBus.PublishAsync(events.KeyGeneratedEvent{Key: privKey})

	log.Printf("API: Key created and saved successfully.")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, "Key created successfully.")
}

// handleUnlockKey handles the POST /setup/unlock-key endpoint.
func (h *apiHandler) handleUnlockKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.appState.Mu.Lock()
	if h.appState.State != core.StateWaitingForPassword {
		stateStr := h.appState.State.String()
		h.appState.Mu.Unlock()
		http.Error(w, fmt.Sprintf("Invalid state (%s) for unlocking key", stateStr), http.StatusConflict)
		return
	}
	keyPath := h.appState.KeyPath
	h.appState.Mu.Unlock()

	var req SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, "Password cannot be empty", http.StatusBadRequest)
		return
	}

	// Attempt to load and decrypt
	privKey, err := identity.LoadAndDecryptKey(keyPath, []byte(req.Password))
	if err != nil {
		h.eventBus.PublishAsync(events.KeyLoadingFailedEvent{Err: err})
		http.Error(w, fmt.Sprintf("Failed to unlock key: %v", err), http.StatusUnauthorized)
		return
	}

	// Key unlocked successfully
	h.eventBus.PublishAsync(events.UserAuthenticatedEvent{Key: privKey})

	log.Printf("API: Key unlocked successfully.")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Key unlocked successfully.")
}
