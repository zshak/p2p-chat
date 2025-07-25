package core

import (
	"sync"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
)

// DaemonState represents the possible operational states of the daemon.
type DaemonState int

const (
	GroupChatProtocolID          = "/p2p-chat-daemon/group-chat/1.0.0"
	ChatProtocolID               = "/p2p-chat-daemon/chat/1.0.0"
	FriendRequestProtocolID      = "/p2p-chat-daemon/friends-request/1.0.0"
	FriendResponseProtocolID     = "/p2p-chat-daemon/friends-response/1.0.0"
	FriendResponsePollProtocolId = "/p2p-chat-daemon/friends-response-poll/1.0.0"
	OnlineAnnouncementTopic      = "p2p-chat/online-announcements"
)

const (
	GroupChatTopic = "/p2p-chat-daemon/group-chat/1.0.0/"
)

const (
	StateInitializing DaemonState = iota
	StateWaitingForKey
	StateWaitingForPassword
	StateInitializingP2P
	StateRunning
	StateShuttingDown
	StateError
)

func (s DaemonState) String() string {
	switch s {
	case StateInitializing:
		return "Initializing"
	case StateWaitingForKey:
		return "Waiting for Key Setup via API"
	case StateWaitingForPassword:
		return "Waiting for Password via API"
	case StateInitializingP2P:
		return "Initializing P2P Network"
	case StateRunning:
		return "Running"
	case StateShuttingDown:
		return "Shutting Down"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// AppState holds the shared state accessible by different parts of the daemon.
type AppState struct {
	Mu           sync.Mutex
	State        DaemonState
	Node         *host.Host
	Dht          *dht.IpfsDHT
	PubSub       *pubsub.PubSub
	KeyPath      string
	PrivKey      crypto.PrivKey
	DbKey        []byte
	LastError    error
	KeyReadyChan chan struct{}
}

// NewAppState creates and initializes a new AppState.
func NewAppState(keyPath string) *AppState {
	return &AppState{
		State:        StateInitializing,
		KeyPath:      keyPath,
		KeyReadyChan: make(chan struct{}),
	}
}
