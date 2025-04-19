package ui_api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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
		log.Printf("API: Failed to generate/save key: %v", err)
		h.appState.Mu.Lock()
		h.appState.LastError = err
		h.appState.State = core.StateError
		h.appState.Mu.Unlock()
		http.Error(w, fmt.Sprintf("Failed to create key: %v", err), http.StatusInternalServerError)
		return
	}

	// Key created
	h.appState.Mu.Lock()
	h.appState.PrivKey = privKey
	if h.appState.State == core.StateWaitingForKey {
		select {
		case <-h.appState.KeyReadyChan:
			// channel already closed
		default:
			// else close
			close(h.appState.KeyReadyChan)
		}
	}
	h.appState.Mu.Unlock()

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
		log.Printf("API: Failed to load/decrypt key: %v", err)
		h.appState.Mu.Lock()
		h.appState.LastError = err
		// allow retry, just return error
		h.appState.Mu.Unlock()
		http.Error(w, fmt.Sprintf("Failed to unlock key: %v", err), http.StatusUnauthorized)
		return
	}

	// Key unlocked successfully
	h.appState.Mu.Lock()
	h.appState.PrivKey = privKey
	if h.appState.State == core.StateWaitingForPassword {
		select {
		case <-h.appState.KeyReadyChan:
			// channel closed
		default:
			// else close
			close(h.appState.KeyReadyChan)
		}
	}
	h.appState.Mu.Unlock()

	log.Printf("API: Key unlocked successfully.")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Key unlocked successfully.")
}
