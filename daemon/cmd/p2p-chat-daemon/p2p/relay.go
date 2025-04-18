package p2p

import (
	// ... imports libp2p autorelay, context, log, config, core, host, dht, peer ...
	"context"
	"github.com/libp2p/go-libp2p" // Need top-level for options
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	circuitv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/proto"
	"log"
	"p2p-chat-daemon/cmd/config"
	// ... other needed imports (cid, circuitv2 proto)
)

// RelayManager configures AutoRelay. It doesn't run continuously itself,
// but provides the configuration options to the Node.
type RelayManager struct {
	cfg          *config.P2PConfig
	dhtProvider  func() (*dht.IpfsDHT, bool) // Function to get DHT when ready
	hostProvider func() (host.Host, bool)    // Function to get Host when ready
}

func NewRelayManager(cfg *config.P2PConfig, dhtProv func() (*dht.IpfsDHT, bool), hostProv func() (host.Host, bool)) *RelayManager {
	return &RelayManager{cfg: cfg, dhtProvider: dhtProv, hostProvider: hostProv}
}

// Libp2pOptions returns the libp2p functional options needed to configure relays.
func (rm *RelayManager) Libp2pOptions() ([]libp2p.Option, error) {
	opts := []libp2p.Option{
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
	}

	// Use PeerSource approach (requires DHT and Host to be available later)
	log.Println("P2P Relay: Configuring AutoRelay with PeerSource.")
	opts = append(opts, libp2p.EnableAutoRelayWithPeerSource(
		rm.peerSource,
		autorelay.WithMinCandidates(len(rm.cfg.BootstrapPeers)),
		autorelay.WithNumRelays(len(rm.cfg.BootstrapPeers)),
		//autorelay.WithBootDelay(15*time.Second), // Give DHT time to connect
	))

	return opts, nil
}

// peerSource finds potential relays using the DHT.
func (rm *RelayManager) peerSource(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
	peerChan := make(chan peer.AddrInfo, numPeers)
	go func() {
		defer close(peerChan)
		dhtInstance, ok := rm.dhtProvider()
		hostInstance, hostOK := rm.hostProvider()
		if !ok || !hostOK {
			log.Println("AutoRelay PeerSource: Host or DHT not ready.")
			return // Dependencies not ready
		}
		selfID := hostInstance.ID()

		// Simplified: Use bootstrap peers first if available
		for _, pid := range dhtInstance.RoutingTable().ListPeers() {
			if pid == selfID {
				continue
			}

			// Get fresh addresses from DHT
			addrs, err := dhtInstance.FindPeer(ctx, pid)
			if err != nil {
				continue
			}

			// Check if peer supports relay protocol
			protos, err := hostInstance.Peerstore().GetProtocols(pid)
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
		// Could add DHT query logic here as well if needed
		log.Println("AutoRelay PeerSource: Finished providing candidates.")

	}()
	return peerChan
}
