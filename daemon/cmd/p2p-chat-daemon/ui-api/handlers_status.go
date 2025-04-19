package ui_api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// handleStatus is the method implementing the /status endpoint logic.
func (h *apiHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.appState.Mu.Lock()
	defer h.appState.Mu.Unlock()

	resp := StatusResponse{
		State: h.appState.State.String(),
	}
	if h.appState.Node != nil {
		resp.PeerID = (*h.appState.Node).ID().String()
		addrs := (*h.appState.Node).Addrs()
		resp.ListenAddrs = make([]string, len(addrs))
		for i, addr := range addrs {
			resp.ListenAddrs[i] = fmt.Sprintf("%s/p2p/%s", addr.String(), resp.PeerID)
		}
	}
	if h.appState.LastError != nil {
		resp.LastError = h.appState.LastError.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
