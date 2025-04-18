package p2p

import (
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"p2p-chat-daemon/cmd/config"
	"time"

	// ... libp2p core imports, crypto, config ...
	"github.com/libp2p/go-libp2p"
	// ... other necessary imports (transports, security, rcmgr)
	"fmt"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"log"
)

// Node manages the creation and lifecycle of the core libp2p host.
type Node struct {
	host host.Host
	cfg  *config.P2PConfig // Relevant P2P config
}

// NewNode creates the libp2p host.
func NewNode(cfg *config.P2PConfig, privKey crypto.PrivKey) (*Node, error) {
	log.Println("P2P Node: Initializing...")
	if cfg == nil {
		return nil, errors.New("p2p config cannot be nil")
	}

	var idOpt libp2p.Option
	if privKey != nil {
		idOpt = libp2p.Identity(privKey)
	}

	// --- Resource Manager ---
	limiter := rcmgr.NewFixedLimiter(rcmgr.DefaultLimits.AutoScale())
	rm, err := rcmgr.NewResourceManager(limiter)
	if err != nil {
		return nil, fmt.Errorf("resource manager creation failed: %w", err)
	}

	// --- Base Host Options ---
	opts := []libp2p.Option{
		idOpt,
		libp2p.ListenAddrStrings(cfg.ListenAddrs...),
		// Include Transports needed
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport),
		// ... other transports ...
		// Include Security
		libp2p.Security(tls.ID, tls.New),
		libp2p.Security(noise.ID, noise.New),
		// Include Resource Manager
		libp2p.ResourceManager(rm),

		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		libp2p.EnableAutoNATv2(),
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("libp2p.New failed: %w", err)
	}

	log.Printf("P2P Node: Host created with ID %s", h.ID().ShortString())
	logNodeDetails(h)

	return &Node{host: h, cfg: cfg}, nil
}

// Host returns the underlying libp2p host.
func (n *Node) Host() host.Host {
	return n.host
}

// Close shuts down the host.
func (n *Node) Close() error {
	log.Println("P2P Node: Closing host...")
	if n.host == nil {
		return nil
	}
	err := n.host.Close()
	if err == nil {
		log.Println("P2P Node: Host closed.")
	} else {
		log.Printf("P2P Node: Error closing host: %v", err)
	}
	return err
}

// logNodeDetails logs node addresses (moved to helper)
func logNodeDetails(node host.Host) {
	for {
		time.Sleep(30 * time.Second)

		ownAddrs := node.Addrs()                     // External Addrs
		psAddrs := node.Peerstore().Addrs(node.ID()) // Peerstore Addrs
		log.Printf("Periodic Check: Own addresses for %s:", node.ID().ShortString())
		log.Println("  External (node.Addrs):")
		for _, addr := range ownAddrs {
			log.Printf("    - %s", addr)
		}
		log.Println("  Peerstore (node.Peerstore().Addrs(self)):")
		for _, addr := range psAddrs {
			log.Printf("    - %s", addr)
		}

		log.Printf("connected peers of Peer ID %s are:", node.ID())
		for _, peerId := range node.Network().Peers() {
			log.Printf("  %s", peerId)
		}
	}
}
