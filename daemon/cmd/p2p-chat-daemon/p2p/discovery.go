package p2p

import (
	// ... imports context, log, config, host, dht ...
	"context"
	"github.com/libp2p/go-libp2p/core/host"
	"log"
	"p2p-chat-daemon/cmd/config"
	"sync"
)

// Discovery manages different peer discovery mechanisms.
type Discovery struct {
	ctx    context.Context
	cancel context.CancelFunc
	cfg    *config.P2PConfig
	host   host.Host
	dht    *DHT // Use the DHT component type
	wg     sync.WaitGroup
	// Store references to specific discovery runners if they need explicit stopping
	dhtDiscovery  *DHTDiscovery
	mdnsDiscovery *MDNSDiscovery
}

func NewDiscovery(parentCtx context.Context, cfg *config.P2PConfig, host host.Host, dht *DHT) (*Discovery, error) {
	// ... nil checks ...
	ctx, cancel := context.WithCancel(parentCtx)
	return &Discovery{
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg,
		host:   host,
		dht:    dht,
	}, nil
}

// Start initiates the configured discovery mechanisms.
func (d *Discovery) Start() {
	log.Println("P2P Discovery: Starting mechanisms...")
	if d.dht != nil && d.dht.Instance() != nil {
		d.dhtDiscovery = NewDHTDiscovery(d.ctx, d.cfg, d.host, d.dht.Instance())
		d.wg.Add(1)
		go d.dhtDiscovery.Run(&d.wg)
	} else {
		log.Println("P2P Discovery: Skipping DHT discovery as DHT is not available.")
	}

	if d.cfg.EnableMDNS {
		d.mdnsDiscovery = NewMDNSDiscovery(d.ctx, d.cfg, d.host)
		err := d.mdnsDiscovery.Start() // mDNS might start synchronously or background
		if err != nil {
			log.Printf("P2P Discovery: WARN - Failed to start mDNS: %v", err)
		}
	}
}

// Stop signals discovery mechanisms to stop.
func (d *Discovery) Stop() {
	log.Println("P2P Discovery: Stopping mechanisms...")
	d.cancel() // Signal goroutines via context

	if d.mdnsDiscovery != nil {
		// MDNS might have its own Close method
		// d.mdnsDiscovery.Close()
	}
	// DHTDiscovery stops via context cancellation

	d.wg.Wait() // Wait for DHT discovery loop
	log.Println("P2P Discovery: Stopped.")
}
