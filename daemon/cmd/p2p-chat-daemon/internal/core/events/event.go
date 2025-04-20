package events

import (
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/core/types"
)

type Event interface{}

type KeyGenerationFailedEvent struct {
	Err error
}

type KeyLoadingFailedEvent struct {
	Err error
}

type KeyGeneratedEvent struct {
	Key crypto.PrivKey
}

type UserAuthenticatedEvent struct {
	Key crypto.PrivKey
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

type MessageSentEvent struct {
	Message types.ChatMessage
}

type MessageReceivedEvent struct {
	Message types.ChatMessage
}
