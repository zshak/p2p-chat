package core

import (
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
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

type DhtCreatedEvent struct {
	Dht *dht.IpfsDHT
}
