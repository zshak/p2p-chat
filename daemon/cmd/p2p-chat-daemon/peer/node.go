package peer

import (
	"context"
	"fmt"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	circuitv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/proto"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"log"
	//"p2p-chat-daemon/cmd/p2p-chat-daemon/discovery"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"                      // The main libp2p package
	"github.com/libp2p/go-libp2p/core/host"            // The Host interface definition
	tls "github.com/libp2p/go-libp2p/p2p/security/tls" // TLS for encryption
	"github.com/multiformats/go-multiaddr"             // For parsing and creating multiaddresses
)

/* CreateLibp2pNode initializes and returns a new libp2p Host */
func CreateLibp2pNode(privKey crypto.PrivKey, appState *core.AppState) (host.Host, error) {
	//if privKey == nil {
	//	return nil, fmt.Errorf("cannot create node with nil private key")
	//}

	log.Println("Initializing libp2p node...")

	// Define the listening addresses for the node.
	// We'll listen on multiple interfaces for better connectivity
	listenAddrs := []string{
		"/ip4/0.0.0.0/tcp/0",      // IPv4 TCP
		"/ip6/::/tcp/0",           // IPv6 TCP
		"/ip4/0.0.0.0/udp/0/quic", // IPv4 QUIC for better NAT traversal
		"/ip6/::/udp/0/quic",      // IPv6 QUIC
	}

	// Create multiaddrs from our strings
	var multiaddrs []multiaddr.Multiaddr
	for _, addr := range listenAddrs {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			log.Printf("Error creating multiaddr %s: %v", addr, err)
			continue
		}
		multiaddrs = append(multiaddrs, ma)
	}

	if len(multiaddrs) == 0 {
		return nil, fmt.Errorf("failed to create any valid listen multiaddrs")
	}

	getDHT := func() (*dht.IpfsDHT, bool) { // Simplified signature for PeerSource usage
		appState.Mu.Lock()
		defer appState.Mu.Unlock()
		isReady := appState.Dht != nil
		if !isReady {
			log.Println("AutoRelay PeerSource: DHT instance not yet available.")
		}
		return appState.Dht, isReady
	}

	peerSource := func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
		peerChan := make(chan peer.AddrInfo, numPeers)

		go func() {
			defer close(peerChan)

			// Get fresh addresses for peers we find
			dhtInstance, ok := getDHT()
			if !ok {
				return
			}

			// Use DHT's FindPeer to get fresh addresses
			for _, pid := range dhtInstance.RoutingTable().ListPeers() {
				if pid == appState.Node.ID() {
					continue
				}

				// Get fresh addresses from DHT
				addrs, err := dhtInstance.FindPeer(ctx, pid)
				if err != nil {
					continue
				}

				// Check if peer supports relay protocol
				protos, err := appState.Node.Peerstore().GetProtocols(pid)
				if err != nil {
					continue
				}

				for _, proto := range protos {
					if proto == circuitv2.ProtoIDv2Hop {
						select {
						case peerChan <- peer.AddrInfo{
							ID:    pid,
							Addrs: addrs.Addrs,
						}:
						case <-ctx.Done():
							return
						}
						break
					}
				}
			}
		}()
		return peerChan
	}
	// libp2p.New is the primary function to create a libp2p Host (our node).
	// It takes Option functions to configure the node.
	node, err := libp2p.New(
		// Listen on multiple addresses
		//libp2p.Identity(privKey),

		libp2p.ListenAddrs(multiaddrs...),

		// Enable multiple security protocols for broader compatibility
		libp2p.Security(tls.ID, tls.New),
		libp2p.Security(noise.ID, noise.New),

		// Enable NAT port mapping for better connectivity behind NATs
		libp2p.NATPortMap(),

		// Enable AutoNAT service to help peers determine their NAT status
		libp2p.EnableNATService(),

		// Enable relay client to connect through relay servers if direct connection fails
		libp2p.EnableRelay(),

		// Enable hole punching for NAT traversal
		libp2p.EnableHolePunching(),

		libp2p.EnableRelayService(),

		//libp2p.EnableAutoRelayWithStaticRelays(discovery.DefaultBootstrapPeers),
		libp2p.EnableAutoRelayWithPeerSource(peerSource, autorelay.WithMinCandidates(2)),

		libp2p.EnableAutoNATv2(),
	)

	// Check if node creation failed.
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err) // Wrap error for context
	}

	return node, nil // Return the created node and nil error on success
}

// LogNodeDetails prints the node's connection information.
// This function now resides in node.go, logically grouped with node creation.
func LogNodeDetails(node host.Host) {
	log.Printf("Node setup successful!")
	log.Printf("Node Peer ID: %s", node.ID()) // node.ID() returns peer.ID which has a String() method
	log.Printf("Connect to me on:")
	// Iterate through the addresses the node is listening on.
	for _, addr := range node.Addrs() {
		// Print the full multiaddress including the Peer ID.
		log.Printf("  %s/p2p/%s", addr, node.ID())
	}

	go PrintConnectedPeers(node)
}

func PrintConnectedPeers(node host.Host) {
	for {
		time.Sleep(30 * time.Second)

		ownAddrs := node.Addrs()                     // External Addrs
		psAddrs := node.Peerstore().Addrs(node.ID()) // Peerstore Addrs
		hasCircuitAddr := false
		log.Printf("Periodic Check: Own addresses for %s:", node.ID().ShortString())
		log.Println("  External (node.Addrs):")
		for _, addr := range ownAddrs {
			log.Printf("    - %s", addr)
			if strings.Contains(addr.String(), "/p2p-circuit") {
				hasCircuitAddr = true
			}
		}
		log.Println("  Peerstore (node.Peerstore().Addrs(self)):")
		for _, addr := range psAddrs {
			log.Printf("    - %s", addr)
			if strings.Contains(addr.String(), "/p2p-circuit") {
				hasCircuitAddr = true
			}
		}
		if hasCircuitAddr {
			log.Println("  -> Relay circuit address detected.")
		} else {
			log.Println("  -> No relay circuit address detected yet.")
		}

		log.Printf("connected peers of Peer ID %s are:", node.ID())
		for _, peerId := range node.Network().Peers() {
			log.Printf("  %s", peerId)
		}
	}
}
