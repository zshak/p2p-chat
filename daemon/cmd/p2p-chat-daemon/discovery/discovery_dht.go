package discovery

import (
	"context"
	"errors"
	"fmt"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	"github.com/libp2p/go-libp2p/core/protocol"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/config"
	"sync"
	"sync/atomic"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

// DHTDiscovery manages advertising and finding peers using the Kademlia DHT.
type DHTDiscovery struct {
	ctx       context.Context
	host      host.Host
	dht       *dht.IpfsDHT
	cfg       *config.P2PConfig
	discovery *routing.RoutingDiscovery
}

// NewDHTDiscovery creates a new DHT discovery manager.
func NewDHTDiscovery(ctx context.Context, cfg *config.P2PConfig, host host.Host) (*DHTDiscovery, error) {
	if host == nil || cfg == nil {
		log.Println("P2P DHT Discovery: ERROR - Cannot initialize with nil host, DHT, or config.")
		return nil, fmt.Errorf("p2P DHT Discovery: ERROR - Cannot initialize with nil host, DHT, or config")
	}

	log.Println("Setting up global DHT discovery...")

	// Create a DHT client mode or server mode based on need

	opts := []dhtopts.Option{
		dht.Mode(dht.ModeAuto), // Client/Auto mode usually sufficient
		dht.BootstrapPeers(cfg.BootstrapPeers...),
	}

	if !cfg.UsePublicBootstraps {
		opts = append(opts, dht.ProtocolPrefix(protocol.ID(cfg.DHTProtocolID)))
	}

	kadDHT, err := dht.New(ctx, host, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	// Bootstrap the DHT to start discovering peers
	if err = kadDHT.Bootstrap(ctx); err != nil {
		return nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	return &DHTDiscovery{
		ctx:       ctx,
		host:      host,
		dht:       kadDHT,
		cfg:       cfg,
		discovery: routing.NewRoutingDiscovery(kadDHT),
	}, nil
}

// connectToBootstrapPeers connects to the well-known bootstrap peers
func (d *DHTDiscovery) connectToBootstrapPeers() error {
	log.Println("Connecting to bootstrap peers...")

	var wg sync.WaitGroup
	var failed int32
	var success int32

	for _, addr := range d.cfg.BootstrapPeers {
		if len(addr.Addrs) == 0 {
			atomic.AddInt32(&failed, 1)
			continue
		}

		wg.Add(1)
		go func(pi peer.AddrInfo) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(d.ctx, 10*time.Second)
			defer cancel()

			log.Printf("Connecting to bootstrap peer: %s", pi.ID)
			if err := d.host.Connect(ctx, pi); err != nil {
				log.Printf("Failed to connect to bootstrap peer %s: %v", pi.ID, err)
				atomic.AddInt32(&failed, 1)
			} else {
				log.Printf("Connected to bootstrap peer: %s", pi.ID)
				atomic.AddInt32(&success, 1)
			}
		}(addr)
	}

	wg.Wait()

	if success == 0 {
		return fmt.Errorf("failed to connect to any bootstrap peers")
	}

	if failed > 0 {
		log.Printf("Failed to connect to %d out of %d bootstrap peers", failed, len(d.cfg.BootstrapPeers))
	}

	return nil
}

// Run starts the periodic advertising and peer finding loop.
// It assumes wg.Add(1) was called before launching this goroutine.
func (d *DHTDiscovery) Run() {
	log.Println("P2P DHT Discovery: Starting background loop...")

	// Wait until DHT is minimally ready
	if !d.waitForDHTReadiness() {
		log.Println("P2P DHT Discovery: Exiting because context was cancelled before DHT was ready.")
		return
	}

	// Initial advertise and find
	go d.advertise(d.ctx)
	go d.findPeers(d.ctx)

	// Periodic ticker
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			log.Println("P2P DHT Discovery: Stopping loop.")
			return
		case <-ticker.C:
			d.advertise(d.ctx)
			d.findPeers(d.ctx)
		}
	}
}

// waitForDHTReadiness waits until the DHT has peers or the context is cancelled.
// Returns true if DHT is ready, false if context cancelled first.
func (d *DHTDiscovery) waitForDHTReadiness() bool {
	for {
		// Use RoutingTable().Size() for a more accurate count of 'good' peers
		if d.dht.RoutingTable().Size() > 0 {
			log.Printf("P2P DHT Discovery: DHT Routing Table has %d peers, ready.", d.dht.RoutingTable().Size())
			return true
		}
		log.Println("P2P DHT Discovery: Waiting for DHT routing table to populate...")

		select {
		case <-d.ctx.Done():
			return false // Context cancelled
		case <-time.After(10 * time.Second): // Check every 10 seconds
			// Continue loop
		}
	}
}

// advertise announces presence on the DHT.
func (d *DHTDiscovery) advertise(ctx context.Context) {
	log.Printf("P2P DHT Discovery: Advertising service '%s'...", d.cfg.DiscoveryServiceID)
	_, err := d.discovery.Advertise(ctx, d.cfg.DiscoveryServiceID)
	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		log.Printf("P2P DHT Discovery: Error advertising service: %v", err)
	}
}

// findPeers looks for peers and attempts connections.
func (d *DHTDiscovery) findPeers(ctx context.Context) {
	log.Printf("P2P DHT Discovery: Finding peers for service '%s'...", d.cfg.DiscoveryServiceID)

	peerChan, err := d.discovery.FindPeers(ctx, d.cfg.DiscoveryServiceID)
	if err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			log.Printf("P2P DHT Discovery: Error finding peers: %v", err)
		}
		return
	}

	var wg sync.WaitGroup
	connectedCount := int32(0)

	for pinfo := range peerChan {
		// Run connection attempts in parallel
		wg.Add(1)
		go func(pi peer.AddrInfo) {
			defer wg.Done()

			// Skip self or peers with no addresses
			if pi.ID == d.host.ID() || len(pi.Addrs) == 0 {
				return
			}
			// Skip already connected peers
			if d.host.Network().Connectedness(pi.ID) == network.Connected {
				return
			}

			// Check context before attempting connection
			if d.ctx.Err() != nil {
				return
			}

			connectCtx, connectCancel := context.WithTimeout(d.ctx, 20*time.Second) // Use service context as parent
			defer connectCancel()

			// log.Printf("P2P DHT Discovery: Attempting connection to discovered peer: %s", pi.ID.ShortString())
			if err := d.host.Connect(connectCtx, pi); err == nil {
				log.Printf("P2P DHT Discovery: Connected to discovered peer: %s", pi.ID.ShortString())
				atomic.AddInt32(&connectedCount, 1)
				// Optional: Protect connection? d.host.ConnManager().Protect(pi.ID, d.cfg.DiscoveryServiceID)
			} else {
				// Log failures less verbosely?
				// if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				//     log.Printf("P2P DHT Discovery: Failed connection to %s: %v", pi.ID.ShortString(), err)
				// }
			}
		}(pinfo) // Pass pinfo by value to goroutine
	}
	wg.Wait() // Wait for all connection attempts in this round

	count := atomic.LoadInt32(&connectedCount)
	if count > 0 {
		log.Printf("P2P DHT Discovery: Connected to %d new peers this round.", count)
	} else {
		// log.Println("P2P DHT Discovery: No new peers connected this round.") // Can be noisy
	}
}
