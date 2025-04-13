package discovery

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p/core/network"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	p2pPeer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

var DefaultBootstrapPeers = addrInfosFromStrings([]string{
	fmt.Sprintf("/ip4/13.61.254.164/tcp/4001/p2p/12D3KooWFujV1a69zhXj7DZeQGKh96ubEVvPBqptHAGYpd6TGdFn"),
	fmt.Sprintf("/ip4/51.21.217.209/tcp/4001/p2p/12D3KooWDW4onEGqyg7Tu9HP8zgnJKZvbo2hgPin63XSVVTsd2eN"),
})

const dhtProtocol = "/p2p-chat-daemon/kad/1.0.0"

const discoveryServiceName = "p2p-chat-daemon"

func addrInfosFromStrings(addrStrings []string) []p2pPeer.AddrInfo {
	var addrInfos []p2pPeer.AddrInfo
	for _, addrStr := range addrStrings {
		addrInfo, err := p2pPeer.AddrInfoFromString(addrStr)
		if err != nil {
			log.Printf("Error parsing bootstrap peer addr %s: %v", addrStr, err)
			continue
		}
		addrInfos = append(addrInfos, *addrInfo)
	}
	return addrInfos
}

// SetupGlobalDiscovery initializes the DHT and discovery service for global peer finding
func SetupGlobalDiscovery(ctx context.Context, node host.Host, shouldUsePublicBts bool) (*dht.IpfsDHT, error) {
	//if shouldUsePublicBts {
	//	DefaultBootstrapPeers = dht.GetDefaultBootstrapPeerAddrInfos()
	//	log.Println("Setting up global DHT discovery with public bootstrap peers...")
	//} else {
	log.Println("Setting up global DHT discovery with private bootstrap peers...")
	//}

	//resources := relayv2.DefaultResources()
	//resources.MaxReservations = 256
	//_, err := relayv2.New(node, relayv2.WithResources(resources))
	//if err != nil {
	//	panic(err)
	//}

	// Create a DHT client mode or server mode based on need
	kadDHT, err := dht.New(ctx, node,
		dht.Mode(dht.ModeAuto),
		dht.ProtocolPrefix(dhtProtocol),
		dht.BootstrapPeers(DefaultBootstrapPeers...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	// Bootstrap the DHT to start discovering peers
	if err = kadDHT.Bootstrap(ctx); err != nil {
		return nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// Connect to bootstrap peers
	if err = connectToBootstrapPeers(ctx, node, DefaultBootstrapPeers); err != nil {
		log.Printf("Warning: %v", err)
		// Continue anyway, as we might connect to some later
	}

	// Start discovering peers regularly
	go startDiscoveryAdvertisement(ctx, node, kadDHT)

	log.Println("Global DHT discovery started successfully")
	return kadDHT, nil
}

// startDiscoveryAdvertisement continuously advertises our presence and discovers peers
func startDiscoveryAdvertisement(ctx context.Context, node host.Host, dht *dht.IpfsDHT) {
	discovery := routing.NewRoutingDiscovery(dht)

	waitForDHTReadiness(ctx, dht)

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

// Wait for the DHT to be minimally ready before advertising
func waitForDHTReadiness(ctx context.Context, dht *dht.IpfsDHT) {
	// Try to get peers from the DHT until we have at least one
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Check if we have any peers in the routing table
			peers := dht.RoutingTable().ListPeers()
			if len(peers) > 0 {
				log.Printf("DHT has %d peers in routing table, ready for advertising", len(peers))
				return
			}

			log.Println("Waiting for DHT to be ready...")
			time.Sleep(5 * time.Second)
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

		if node.Network().Connectedness(peer.ID) == network.Connected {
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
