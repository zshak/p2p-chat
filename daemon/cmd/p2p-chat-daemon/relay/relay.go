package relay

import (
	"context"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	circuitv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/proto"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/config"
)

// RelayManager configures AutoRelay. It doesn't run continuously itself,
// but provides the configuration options to the Node.
type RelayManager struct {
	cfg          *config.P2PConfig
	dhtProvider  func() (*dht.IpfsDHT, bool)
	hostProvider func() (host.Host, bool)
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

	log.Println("P2P Relay: Configuring AutoRelay with PeerSource.")
	opts = append(opts, libp2p.EnableAutoRelayWithPeerSource(
		rm.peerSource,
		autorelay.WithMinCandidates(len(rm.cfg.BootstrapPeers)),
		autorelay.WithNumRelays(len(rm.cfg.BootstrapPeers)),
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
			return
		}
		selfID := hostInstance.ID()

		for _, pid := range dhtInstance.RoutingTable().ListPeers() {
			if pid == selfID {
				continue
			}

			addrs, err := dhtInstance.FindPeer(ctx, pid)
			if err != nil {
				continue
			}

			protos, err := hostInstance.Peerstore().GetProtocols(pid)
			if err != nil {
				continue
			}

			for _, proto := range protos {
				if proto == circuitv2.ProtoIDv2Stop {
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
		log.Println("AutoRelay PeerSource: Finished providing candidates.")

	}()
	return peerChan
}
