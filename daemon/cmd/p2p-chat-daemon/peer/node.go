package peer

import (
	"fmt"
	"github.com/caddyserver/certmagic"
	"github.com/libp2p/go-libp2p/core/crypto"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/discovery"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"time"

	p2pforge "github.com/ipshipyard/p2p-forge/client"
	"github.com/libp2p/go-libp2p"                      // The main libp2p package
	"github.com/libp2p/go-libp2p/core/host"            // The Host interface definition
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

	//getDHT := func() (*dht.IpfsDHT, bool) { // Simplified signature for PeerSource usage
	//	appState.Mu.Lock()
	//	defer appState.Mu.Unlock()
	//	isReady := appState.Dht != nil
	//	if !isReady {
	//		log.Println("AutoRelay PeerSource: DHT instance not yet available.")
	//	}
	//	return appState.Dht, isReady
	//}

	//peerSource := func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
	//	log.Printf("AutoRelay PeerSource: Finding relay candidates, need %d peers", numPeers)
	//	peerChan := make(chan peer.AddrInfo, numPeers)
	//
	//	go func() {
	//		defer close(peerChan)
	//
	//		// Get Host ID safely *after* node is created and stored in AppState
	//		appState.Mu.Lock()
	//		selfID := appState.Node.ID()
	//		appState.Mu.Unlock()
	//		if selfID == "" {
	//			log.Println("AutoRelay PeerSource: Cannot get self ID, node not ready in AppState.")
	//			return
	//		}
	//
	//		// Check if DHT is available
	//		dhtInstance, ok := getDHT()
	//		if !ok {
	//			log.Println("AutoRelay PeerSource: DHT not ready, using bootstrap peers as fallback only.")
	//			// Immediately send bootstrap peers if DHT isn't ready
	//			return
	//		}
	//
	//		// --- Find Peers from DHT/Network ---
	//		rtPeers := dhtInstance.RoutingTable().ListPeers()
	//		connectedPeers := appState.Node.Network().Peers() // Access Node safely
	//
	//		allPeers := make(map[peer.ID]struct{})
	//		for _, p := range rtPeers {
	//			allPeers[p] = struct{}{}
	//		}
	//		for _, p := range connectedPeers {
	//			allPeers[p] = struct{}{}
	//		}
	//		delete(allPeers, selfID) // Remove self
	//
	//		log.Printf("AutoRelay PeerSource: Found %d unique candidate peers from RT/Connections", len(allPeers))
	//
	//		// --- Check peers for relay capability ---
	//		var wg sync.WaitGroup
	//		// Use atomic types correctly
	//		var peersChecked int32 // Starts at 0
	//		var peersFound int32   // Starts at 0
	//		mutex := &sync.Mutex{} // Mutex for the testedPeers map
	//		testedPeers := make(map[peer.ID]bool)
	//
	//		// Process peers concurrently (maybe limit concurrency later if needed)
	//		for pid := range allPeers {
	//			// Early exit check
	//			if atomic.LoadInt32(&peersFound) >= int32(numPeers) {
	//				break
	//			}
	//
	//			wg.Add(1)
	//			go func(p peer.ID) {
	//				defer wg.Done()
	//
	//				// Check context cancellation
	//				if ctx.Err() != nil {
	//					return
	//				}
	//
	//				// Increment checked count atomically
	//				atomic.AddInt32(&peersChecked, 1)
	//
	//				// Avoid re-testing (check needs lock)
	//				mutex.Lock()
	//				if _, tested := testedPeers[p]; tested {
	//					mutex.Unlock()
	//					return
	//				}
	//				testedPeers[p] = true
	//				mutex.Unlock()
	//
	//				// Exit early if we already found enough
	//				if atomic.LoadInt32(&peersFound) >= int32(numPeers) {
	//					return
	//				}
	//
	//				// --- Check methods ---
	//				isRelay := false
	//				addrInfo := peer.AddrInfo{ID: p} // Prepare AddrInfo
	//
	//				// Method 1: Check supported protocols
	//				protos, err := appState.Node.Peerstore().GetProtocols(p) // Access Node safely
	//				if err == nil {
	//					for _, proto := range protos {
	//						if proto == circuitv2.ProtoIDv2Hop { // Use correct constant
	//							isRelay = true
	//							log.Printf("AutoRelay PeerSource: Peer %s supports relay protocol.", p.ShortString())
	//							break
	//						}
	//					}
	//				} else {
	//					log.Printf("AutoRelay PeerSource: Error getting protocols for %s: %v", p.ShortString(), err)
	//				}
	//				//
	//				//// Method 2: Check for relay addresses (less reliable indicator of being a *good* relay)
	//				//if !isRelay {
	//				//	addrs := appState.Node.Peerstore().Addrs(p) // Access Node safely
	//				//	addrInfo.Addrs = addrs                      // Store addresses if we found them
	//				//	for _, addr := range addrs {
	//				//		if strings.Contains(addr.String(), "/p2p-circuit") {
	//				//			isRelay = true
	//				//			log.Printf("AutoRelay PeerSource: Peer %s has p2p-circuit address.", p.ShortString())
	//				//			break
	//				//		}
	//				//	}
	//				//}
	//
	//				// --- If identified as a potential relay, send it ---
	//				if isRelay {
	//					// Atomically check if we still need peers *before* incrementing
	//					// This reduces sending slightly more than numPeers in rare races.
	//					if atomic.LoadInt32(&peersFound) < int32(numPeers) {
	//						// Increment found count atomically *and* check the new value
	//						newFoundCount := atomic.AddInt32(&peersFound, 1)
	//						// Send only if this increment didn't exceed the limit
	//						if newFoundCount <= int32(numPeers) {
	//							// Ensure we have addresses for the peer
	//							if len(addrInfo.Addrs) == 0 {
	//								addrInfo.Addrs = appState.Node.Peerstore().Addrs(p)
	//							}
	//							if len(addrInfo.Addrs) > 0 { // Only send if we have addresses
	//								select {
	//								case peerChan <- addrInfo:
	//									log.Printf("AutoRelay PeerSource: Added peer %s as relay candidate (%d/%d)",
	//										p.ShortString(), newFoundCount, numPeers)
	//								case <-ctx.Done():
	//									return
	//								}
	//							} else {
	//								// Couldn't get addresses, decrement count as we didn't actually send it
	//								atomic.AddInt32(&peersFound, -1)
	//								log.Printf("AutoRelay PeerSource: Peer %s identified as relay but has no addresses in peerstore.", p.ShortString())
	//							}
	//						} else {
	//							// We incremented, but it pushed us over the limit. Decrement back.
	//							atomic.AddInt32(&peersFound, -1)
	//						}
	//					}
	//				}
	//			}(pid) // Pass pid to the goroutine
	//		} // End of loop over peers
	//
	//		// Wait for all checks to complete
	//		wg.Wait()
	//
	//		log.Printf("AutoRelay PeerSource: Finished checking %d peers, found %d valid relay candidates.",
	//			atomic.LoadInt32(&peersChecked), atomic.LoadInt32(&peersFound)) // Use atomic load
	//	}() // End of main goroutine for peerSource
	//
	//	return peerChan
	//} // End of peerSource definition

	limiter := rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits)
	rm, err := rcmgr.NewResourceManager(limiter)

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
		libp2p.EnableAutoRelayWithStaticRelays(discovery.DefaultBootstrapPeers),
		//libp2p.EnableAutoRelayWithPeerSource(peerSource),
		libp2p.ResourceManager(rm),
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
