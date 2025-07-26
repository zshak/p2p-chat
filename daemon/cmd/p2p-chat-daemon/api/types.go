package api

import (
	"encoding/json"
)

type StatusResponse struct {
	State       string   `json:"state"`
	PeerID      string   `json:"peer_id,omitempty"`
	ListenAddrs []string `json:"listen_addrs,omitempty"`
	LastError   string   `json:"last_error,omitempty"`
}

type SetupRequest struct {
	Password string `json:"password"`
}

type MessageRequest struct {
	Payload json.RawMessage `json:"payload"`
	Type    WsMessageType   `json:"type"`
}

type WsDirectMessageRequestPayload struct {
	TargetPeerID string `json:"target_peer_id"`
	Message      string `json:"message"`
}

type WsGroupMessageRequestPayload struct {
	GroupId string `json:"group_id"`
	Message string `json:"message"`
}

type FriendRequest struct {
	ReceiverPeerId string `json:"receiver_peer_id"`
}

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

type WsMessage struct {
	Type    WsMessageType   `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type WsDirectMessagePayload struct {
	TargetPeerId string `json:"target_peer_id"`
	SenderPeerId string `json:"sender_peer_id"`
	Message      string `json:"message"`
}

type WsGroupMessagePayload struct {
	GroupId      string `json:"group_id"`
	SenderPeerId string `json:"sender_peer_id"`
	Message      string `json:"message"`
}
