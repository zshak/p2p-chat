package discovery

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p/core/host"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/config"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/bus"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core/events"
)

// Manager handles all peer discovery mechanisms
type Manager struct {
	ctx           context.Context
	node          *host.Host
	cfg           *config.Config
	dhtDiscovery  *DHTDiscovery
	mdnsDiscovery *MDNSDiscovery
	eventBus      *bus.EventBus
}

// NewDiscoveryManager creates a new discovery manager*
func NewDiscoveryManager(ctx context.Context, node *host.Host, cfg *config.Config, bus *bus.EventBus) (*Manager, error) {
	dhtDisc, err := NewDHTDiscovery(ctx, &cfg.P2P, node)

	if err != nil {
		return nil, fmt.Errorf("dht set up failed")
	}

	mdnsDiscovery := NewMDNSDiscovery(ctx, &cfg.P2P, node)

	return &Manager{
		ctx:           ctx,
		node:          node,
		cfg:           cfg,
		eventBus:      bus,
		dhtDiscovery:  dhtDisc,
		mdnsDiscovery: mdnsDiscovery,
	}, nil
}

// Initialize sets up DHT-based discovery
func (dm *Manager) Initialize() error {
	dm.dhtDiscovery.connectToBootstrapPeers()
	dm.eventBus.PublishAsync(events.DhtCreatedEvent{Dht: dm.dhtDiscovery.dht})

	go dm.mdnsDiscovery.Run()
	go dm.dhtDiscovery.Run()

	return nil
}
