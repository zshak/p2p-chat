package types

import (
	"fmt"
	"time"
)

type ChatMessage struct {
	ID              int64     // Database ID
	RecipientPeerId string    // Peer ID string of the other participant
	SenderPeerID    string    // Actual sender's Peer ID string
	SendTime        time.Time // Use time.Time for easier handling
	Content         string
	IsOutgoing      bool
}

type StoredMessage struct {
	ID              int64     // Database ID
	RecipientPeerId string    // Peer ID string of the other participant
	SenderPeerID    string    // Actual sender's Peer ID string
	SendTime        time.Time // Use time.Time for easier handling
	Content         []byte
	IsOutgoing      bool
}

type FriendStatus int // Use an integer underlying type

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
	PeerID      string       // The Peer ID string of the other party
	Status      FriendStatus // Our view of the relationship status
	RequestedAt time.Time    // Timestamp when request was sent/received
	ApprovedAt  time.Time    // Timestamp when approved
	IsOnline    bool         // is user Online currently
}

// GroupKey represents the group key.
type GroupKey struct {
	GroupId   string
	Key       []byte
	CreatedAt time.Time
}

// StoredGroupMessage represents the group message.
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
