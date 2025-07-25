package events

import (
	"github.com/gorilla/websocket"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/core/types"
	"time"
)

type Event interface{}

type KeyGenerationFailedEvent struct {
	Err error
}

type KeyLoadingFailedEvent struct {
	Err error
}

type KeyGeneratedEvent struct {
	Key   crypto.PrivKey
	DbKey []byte
}

type UserAuthenticatedEvent struct {
	Key   crypto.PrivKey
	DbKey []byte
}

type ApiStartedEvent struct {
}

type HostInitializedEvent struct {
	Host *host.Host
}

type DhtCreatedEvent struct {
	Dht *dht.IpfsDHT
}

type SetupCompletedEvent struct{}

type WsConnectionEstablishedEvent struct {
	Conn *websocket.Conn
}

type MessageSentEvent struct {
	Message types.ChatMessage
}

type MessageReceivedEvent struct {
	Message types.ChatMessage
}

type GroupChatMessageReceivedEvent struct {
	Message GroupChatMessage
}

type GroupChatMessageSentEvent struct {
	Message GroupChatMessage
}

type GroupChatMessage struct {
	GroupId      string
	SenderPeerId string
	Message      string
	Time         time.Time
}
type FriendRequestReceived struct {
	FriendRequest types.FriendRequestData
}

type FriendResponseReceivedEvent struct {
	SenderPeerId string
	Status       types.FriendStatus
	Timestamp    string
}

type FriendRequestSentEvent struct {
	ReceiverPeerId string
	Timestamp      time.Time
}

type FriendResponseSentEvent struct {
	PeerId     string
	IsAccepted bool
}

type FriendOnlineStatusChangedEvent struct {
	PeerID   string
	IsOnline bool
	LastSeen time.Time
	RTT      time.Duration
}
