package discovery

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	p2pPeer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

// Common bootstrap nodes that your application will use
// These are well-known libp2p bootstrap nodes - might want to run one just for this app for Prod
var defaultBootstrapPeers = dht.GetDefaultBootstrapPeerAddrInfos()

// The namespace for our DHT routing
const dhtProtocol = "/p2p-chat-daemon/kad/1.0.0"

// The service name for our application
const discoveryServiceName = "p2p-chat-daemon"

// SetupGlobalDiscovery initializes the DHT and discovery service for global peer finding
func SetupGlobalDiscovery(ctx context.Context, node host.Host) (*dht.IpfsDHT, error) {
	log.Println("Setting up global DHT discovery...")

	// Create a DHT client mode or server mode based on need
	kadDHT, err := dht.New(ctx, node,
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix(dhtProtocol),
		dht.BootstrapPeers(defaultBootstrapPeers...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	// Connect to bootstrap peers
	if err = connectToBootstrapPeers(ctx, node, defaultBootstrapPeers); err != nil {
		log.Printf("Warning: %v", err)
		// Continue anyway, as we might connect to some later
	}

	// Bootstrap the DHT to start discovering peers
	if err = kadDHT.Bootstrap(ctx); err != nil {
		return nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// Start discovering peers regularly
	go startDiscoveryAdvertisement(ctx, node, kadDHT)

	log.Println("Global DHT discovery started successfully")
	return kadDHT, nil
}

// startDiscoveryAdvertisement continuously advertises our presence and discovers peers
func startDiscoveryAdvertisement(ctx context.Context, node host.Host, dht *dht.IpfsDHT) {
	// Create a discovery service that uses the DHT to find peers
	discovery := routing.NewRoutingDiscovery(dht)

	// Advertise our presence
	_, err := discovery.Advertise(ctx, discoveryServiceName)
	if err != nil {
		log.Printf("Error: %s", err)
	}

	// Run an initial peer discovery immediately
	findAndConnectPeers(ctx, node, discovery)

	// Then continue with periodic discovery
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			findAndConnectPeers(ctx, node, discovery)
		}
	}
}

// findAndConnectPeers discovers and connects to peers using the discovery service
func findAndConnectPeers(ctx context.Context, node host.Host, discovery *routing.RoutingDiscovery) {
	log.Println("Looking for peers...")

	// Find peers advertising the same service
	peerChan, err := discovery.FindPeers(ctx, discoveryServiceName)
	if err != nil {
		log.Printf("Error finding peers: %v", err)
		return
	}

	// Connect to discovered peers
	var wg sync.WaitGroup
	var connectedPeers int32

	for peer := range peerChan {
		// Skip if no addresses or it's ourselves
		if len(peer.Addrs) == 0 || peer.ID == node.ID() {
			continue
		}

		wg.Add(1)
		go func(p p2pPeer.AddrInfo) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			log.Printf("Attempting to connect to discovered peer: %s", p.ID)
			if err := node.Connect(ctx, p); err != nil {
				log.Printf("Failed to connect to peer %s: %v", p.ID, err)
			} else {
				log.Printf("Connected to peer: %s", p.ID)
				atomic.AddInt32(&connectedPeers, 1)
			}
		}(peer)
	}
	wg.Wait()

	if connectedPeers > 0 {
		log.Printf("Connected to %d new peers", connectedPeers)
	}
}

// connectToBootstrapPeers connects to the well-known bootstrap peers
func connectToBootstrapPeers(ctx context.Context, node host.Host, bootstrapPeers []p2pPeer.AddrInfo) error {
	log.Println("Connecting to bootstrap peers...")

	var wg sync.WaitGroup
	var failed int32
	var success int32

	for _, addr := range bootstrapPeers {
		if len(addr.Addrs) == 0 {
			atomic.AddInt32(&failed, 1)
			continue
		}

		wg.Add(1)
		go func(pi p2pPeer.AddrInfo) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			log.Printf("Connecting to bootstrap peer: %s", pi.ID)
			if err := node.Connect(ctx, pi); err != nil {
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
		log.Printf("Failed to connect to %d out of %d bootstrap peers", failed, len(bootstrapPeers))
	}

	return nil
}
