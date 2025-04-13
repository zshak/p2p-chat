package peer

import (
	"context"
	"fmt"
	"github.com/caddyserver/certmagic"
	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multihash"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"time"

	p2pforge "github.com/ipshipyard/p2p-forge/client"
	"github.com/libp2p/go-libp2p"           // The main libp2p package
	"github.com/libp2p/go-libp2p/core/host" // The Host interface definition
	circuitv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/proto"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls" // TLS for encryption
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	webrtc "github.com/libp2p/go-libp2p/p2p/transport/webrtc"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	webtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"
)

/* CreateLibp2pNode initializes and returns a new libp2p Host */
func CreateLibp2pNode(privKey crypto.PrivKey, appState *core.AppState) (host.Host, error) {
	//if privKey == nil {
	//	return nil, fmt.Errorf("cannot create node with nil private key")
	//}

	log.Println("Initializing libp2p node...")

	certLoaded := make(chan bool, 1)

	certManager, err := p2pforge.NewP2PForgeCertMgr(
		p2pforge.WithCertificateStorage(&certmagic.FileStorage{Path: "p2p-forge-certs"}),
		p2pforge.WithUserAgent("go-libp2p/example/autotls"),
		p2pforge.WithCAEndpoint(p2pforge.DefaultCAEndpoint),
		p2pforge.WithOnCertLoaded(func() { certLoaded <- true }), // Signal when cert is loaded
	)
	if err != nil {
		panic(err)
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
		log.Printf("AutoRelay PeerSource: Finding %d relay candidates using DHT...", numPeers)
		peerChan := make(chan peer.AddrInfo, numPeers) // Buffer slightly

		go func() {
			defer close(peerChan)
			dhtInstance, ok := getDHT()
			if !ok {
				log.Println("AutoRelay PeerSource: DHT not ready, cannot find relays.")
				return // DHT not ready
			}

			// Find peers advertising the HOP protocol
			// Note: Using ProtoHop is conceptual, use the actual exported constant
			// from the circuitv2 package. It might be ProtoID_v2_hop or similar.
			// Let's assume circuitv2.ProtoHop exists for this example.
			relayProtoCid, _ := cid.V1Builder{Codec: cid.Raw, MhType: multihash.IDENTITY}.Sum([]byte(circuitv2.ProtoIDv2Hop)) // Adjust based on actual constant
			log.Printf("AutoRelay PeerSource: Querying DHT for providers of CID %s", relayProtoCid)

			provCtx, cancel := context.WithTimeout(ctx, 2*time.Minute) // Timeout for provider query
			defer cancel()
			providers := dhtInstance.FindProvidersAsync(provCtx, relayProtoCid, numPeers*2) // Find slightly more initially

			count := 0
			for p := range providers {
				// Simple filtering: exclude self
				if p.ID == appState.Node.ID() { // Access node ID via appState safely
					continue
				}

				// More advanced filtering could go here (check connectivity, etc.)

				log.Printf("AutoRelay PeerSource: Found potential relay %s", p.ID.ShortString())
				select {
				case peerChan <- p:
					count++
					if count >= numPeers {
						log.Printf("AutoRelay PeerSource: Found sufficient relay candidates (%d).", count)
						return // Found enough peers
					}
				case <-ctx.Done():
					log.Println("AutoRelay PeerSource: Context cancelled during provider search.")
					return // Context cancelled
				}
			}
			log.Printf("AutoRelay PeerSource: Finished DHT query, found %d candidates.", count)
		}()

		return peerChan
	}

	// libp2p.New is the primary function to create a libp2p Host (our node).
	// It takes Option functions to configure the node.
	node, err := libp2p.New(
		// Listen on multiple addresses
		//libp2p.Identity(privKey),

		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/9095",
			"/ip4/0.0.0.0/udp/9095/quic-v1",
			"/ip4/0.0.0.0/udp/9095/quic-v1/webtransport",
			"/ip4/0.0.0.0/udp/9095/webrtc-direct",
			"/ip6/::/tcp/9095",
			"/ip6/::/udp/9095/quic-v1",
			"/ip6/::/udp/9095/quic-v1/webtransport",
			"/ip6/::/udp/9095/webrtc-direct",
			fmt.Sprintf("/ip4/0.0.0.0/tcp/9095/tls/sni/*.%s/ws", p2pforge.DefaultForgeDomain),
			fmt.Sprintf("/ip6/::/tcp/9095/tls/sni/*.%s/ws", p2pforge.DefaultForgeDomain),
		),

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

		libp2p.Transport(webtransport.New),
		libp2p.Transport(quic.NewTransport),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(webrtc.New),

		// Share the same TCP listener between the TCP and WS transports
		libp2p.ShareTCPListener(),

		// Configure the WS transport with the AutoTLS cert manager
		libp2p.Transport(ws.New, ws.WithTLSConfig(certManager.TLSConfig())),
		libp2p.EnableAutoNATv2(),
		libp2p.EnableAutoRelayWithPeerSource(peerSource),
	)
	certManager.ProvideHost(node)

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
		log.Printf("connected peers of Peer ID %s are:", node.ID())
		for _, peerId := range node.Network().Peers() {
			log.Printf("  %s", peerId)
		}
	}
}
