package peer

import (
	"context"
	"fmt"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/config"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"strings"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	circuitv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/proto"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/multiformats/go-multiaddr"
)

// NodeManager handles the creation and management of the libp2p node
type NodeManager struct {
	ctx      context.Context
	appState *core.AppState
	cfg      *config.P2PConfig
	node     *host.Host
}

// NewNodeManager creates a new NodeManager
func NewNodeManager(ctx context.Context, appState *core.AppState, cfg *config.P2PConfig) *NodeManager {
	return &NodeManager{
		ctx:      ctx,
		appState: appState,
		cfg:      cfg,
	}
}

// Initialize creates and initializes the libp2p node
func (nm *NodeManager) Initialize() (*host.Host, error) {
	log.Println("Initializing libp2p node...")

	listenAddrs := []string{
		"/ip4/0.0.0.0/tcp/0",      // IPv4 TCP
		"/ip6/::/tcp/0",           // IPv6 TCP
		"/ip4/0.0.0.0/udp/0/quic", // IPv4 QUIC
		"/ip6/::/udp/0/quic",      // IPv6 QUIC
	}

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

	peerSource := nm.createPeerSourceFunc()

	opts := []libp2p.Option{
		libp2p.Identity(nm.appState.PrivKey),
		libp2p.ListenAddrs(multiaddrs...),
		libp2p.Security(tls.ID, tls.New),
		libp2p.Security(noise.ID, noise.New),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelayService(),
		libp2p.EnableAutoRelayWithPeerSource(peerSource, autorelay.WithMinCandidates(len(nm.cfg.BootstrapPeers))),
		libp2p.EnableAutoNATv2(),
	}

	node, err := libp2p.New(opts...)

	addrs := node.Addrs()
	for _, addr := range addrs {
		fmt.Printf("Listening on: %s\n", addr.String())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	nm.node = &node
	return &node, nil
}

// createPeerSourceFunc creates a function that provides peers for auto relay
func (nm *NodeManager) createPeerSourceFunc() func(context.Context, int) <-chan peer.AddrInfo {
	return func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
		peerChan := make(chan peer.AddrInfo, numPeers)
		log.Printf("Looking for relay %v relay nodes", numPeers)

		go func() {
			defer close(peerChan)

			nm.appState.Mu.Lock()
			dhtInstance := nm.appState.Dht
			nodeInstance := *nm.appState.Node
			nm.appState.Mu.Unlock()

			if dhtInstance == nil || nodeInstance == nil {
				log.Println("AutoRelay PeerSource: DHT or node instance not yet available.")
				return
			}

			found := int32(0)
			for _, pid := range dhtInstance.RoutingTable().ListPeers() {
				if pid == nodeInstance.ID() {
					continue
				}

				addrInfo, err := dhtInstance.FindPeer(ctx, pid)
				if err != nil {
					continue
				}

				protos, err := nodeInstance.Peerstore().GetProtocols(pid)
				if err != nil {
					continue
				}

				for _, proto := range protos {
					if proto == circuitv2.ProtoIDv2Stop {
						atomic.AddInt32(&found, 1)
						select {
						case peerChan <- peer.AddrInfo{
							ID:    pid,
							Addrs: addrInfo.Addrs,
						}:
						case <-ctx.Done():
							return
						}
						break
					}
				}
			}

			log.Printf("Found %v relay nodes", found)
		}()

		return peerChan
	}
}

func (nm *NodeManager) LogNodeDetails() {
	log.Printf("Node setup successful!")
	log.Printf("Node Peer ID: %s", (*nm.node).ID())
	log.Printf("Connect to me on:")
	for _, addr := range (*nm.node).Addrs() {
		log.Printf("  %s/p2p/%s", addr, (*nm.node).ID())
	}

	go nm.MonitorConnectedPeers()
}

// MonitorConnectedPeers periodically logs information about connected peers
func (nm *NodeManager) MonitorConnectedPeers() {
	for {
		select {
		case <-nm.ctx.Done():
			return
		case <-time.After(30 * time.Second):
			nm.logPeerStatus()
		}
	}
}

// logPeerStatus logs the current peer connection status
func (nm *NodeManager) logPeerStatus() {
	ownAddrs := (*nm.node).Addrs()
	psAddrs := (*nm.node).Peerstore().Addrs((*nm.node).ID())
	hasCircuitAddr := false

	log.Printf("Periodic Check: Own addresses for %s:", (*nm.node).ID().ShortString())
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

	log.Printf("Connected peers of Peer ID %s are:", (*nm.node).ID())
	for _, peerId := range (*nm.node).Network().Peers() {
		log.Printf("  %s", peerId)
	}
}

func (nm *NodeManager) Close() error {
	if nm.node != nil {
		log.Println("Closing libp2p node...")
		err := (*nm.node).Close()
		if err != nil {
			log.Printf("Error closing libp2p node: %v", err)
			return err
		}
		log.Println("Libp2p node closed.")
	}
	return nil
}
