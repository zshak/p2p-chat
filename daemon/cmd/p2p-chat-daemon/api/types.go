package api

import (
	"encoding/json"
)

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

// MessageRequest defines the structure for sending a chat message via the API.
type MessageRequest struct {
	Payload json.RawMessage `json:"payload"`
	Type    WsMessageType   `json:"type"`
}

// WsDirectMessageRequestPayload for when a new direct message is sent
type WsDirectMessageRequestPayload struct {
	TargetPeerID string `json:"target_peer_id"`
	Message      string `json:"message"`
}

// WsGroupMessageRequestPayload for when a new group message is sent
type WsGroupMessageRequestPayload struct {
	GroupId string `json:"group_id"`
	Message string `json:"message"`
}

// FriendRequest defines the structure for sending a friends request via the API.
type FriendRequest struct {
	ReceiverPeerId string `json:"receiver_peer_id"`
}

// FriendRequestResponse defines the structure for accepting or rejecting a friends request via the API.
type FriendRequestResponse struct {
	PeerId     string `json:"peer_id"`
	IsAccepted bool   `json:"is_accepted"`
}

type CreateGroupChatRequest struct {
	MemberPeerIds []string `json:"member_peers"`
	ChatName      string   `json:"name"`
}

type SendGroupChatMessageRequest struct {
	Message string `json:"message"`
	GroupId string `json:"group_id"`
}

type GetGroupChatMessagesRequest struct {
	GroupId string `json:"group_id"`
}

type GetChatMessagesRequest struct {
	PeerId string `json:"peer_id"`
}

type WsMessageType string

const (
	WsMsgTypeDirectMessage WsMessageType = "DIRECT_MESSAGE"
	WsMsgTypeGroupMessage  WsMessageType = "GROUP_MESSAGE"
)

// WsMessage wrapper
type WsMessage struct {
	Type    WsMessageType   `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// WsDirectMessagePayload for when a new direct message is received
type WsDirectMessagePayload struct {
	SenderPeerId string `json:"sender_peer_id"`
	Message      string `json:"message"`
}

// WsGroupMessagePayload for when a new group message is received
type WsGroupMessagePayload struct {
	GroupId      string `json:"group_id"`
	SenderPeerId string `json:"sender_peer_id"`
	Message      string `json:"message"`
}
