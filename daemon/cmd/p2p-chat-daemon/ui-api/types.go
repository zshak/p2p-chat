package ui_api

// Contains API request/response struct definitions

// StatusResponse represents the data returned by the /status endpoint.
type StatusResponse struct {
	State       string   `json:"state"`
	PeerID      string   `json:"peer_id,omitempty"`
	ListenAddrs []string `json:"listen_addrs,omitempty"`
	LastError   string   `json:"last_error,omitempty"`
}

// SetupRequest represents the data needed for key setup/unlock endpoints.
type SetupRequest struct {
	Password string `json:"password"`
}

// ChatMessageRequest defines the structure for sending a chat message via the API.
type ChatMessageRequest struct {
	TargetPeerID string `json:"target_peer_id"`
	Message      string `json:"message"`
}
