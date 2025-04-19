package types

import "time"

type ChatMessage struct {
	ID              int64     // Database ID
	RecipientPeerId string    // Peer ID string of the other participant
	SenderPeerID    string    // Actual sender's Peer ID string
	SendTime        time.Time // Use time.Time for easier handling
	Content         string
	IsOutgoing      bool
}
