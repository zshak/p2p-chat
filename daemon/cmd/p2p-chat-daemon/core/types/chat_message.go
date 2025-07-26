package types

import (
	"fmt"
	"time"
)

type ChatMessage struct {
	ID              int64
	RecipientPeerId string
	SenderPeerID    string
	SendTime        time.Time
	Content         string
	IsOutgoing      bool
}

type StoredMessage struct {
	ID              int64
	RecipientPeerId string
	SenderPeerID    string
	SendTime        time.Time
	Content         []byte
	IsOutgoing      bool
}

type FriendStatus int

const (
	FriendStatusNone     FriendStatus = iota // 0 - Default, no relationship/request exists
	FriendStatusSent                         // 1 - Request SENT by us to them
	FriendStatusPending                      // 2 - Request RECEIVED by us from them, awaiting our action
	FriendStatusApproved                     // 3 - Friends (request accepted by us or them)
	FriendStatusRejected                     // 4 - Friends (request rejected)
)

// String makes FriendStatus implement fmt.Stringer
func (s FriendStatus) String() string {
	switch s {
	case FriendStatusNone:
		return "None"
	case FriendStatusSent:
		return "Sent"
	case FriendStatusPending:
		return "Pending"
	case FriendStatusApproved:
		return "Approved"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

// FriendRelationship represents the stored state between two peers.
type FriendRelationship struct {
	PeerID      string
	Status      FriendStatus
	RequestedAt time.Time
	ApprovedAt  time.Time
	IsOnline    bool
	DisplayName string `json:"display_name,omitempty"`
}

type GroupKey struct {
	GroupId   string
	Key       []byte
	Name      string
	CreatedAt time.Time
}

type StoredGroupMessage struct {
	ID               int64
	GroupID          string
	SenderPeerID     string
	EncryptedContent []byte
	SentAt           time.Time
}

type GroupChatMessage struct {
	Id           string
	SenderPeerId string
	Message      string
	Time         time.Time
}
