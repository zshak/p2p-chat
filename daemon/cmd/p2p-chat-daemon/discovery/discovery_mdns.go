package discovery

import (
	"context"
	"errors"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/config"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

// MDNSDiscovery manages peer discovery via mDNS
type MDNSDiscovery struct {
	host    host.Host
	cfg     *config.P2PConfig
	service mdns.Service
	ctx     context.Context
}

// discoveryNotifee handles mDNS peer found events.
type discoveryNotifee struct {
	h                  host.Host
	ctx                context.Context
	connectionAttempts map[peer.ID]time.Time
	mutex              sync.Mutex
}

func newDiscoveryNotifee(ctx context.Context, h host.Host) *discoveryNotifee {
	return &discoveryNotifee{
		h:                  h,
		ctx:                ctx,
		connectionAttempts: make(map[peer.ID]time.Time),
	}
}

// NewMDNSDiscovery creates a new mDNS discovery manager.
func NewMDNSDiscovery(ctx context.Context, cfg *config.P2PConfig, host *host.Host) *MDNSDiscovery {
	if host == nil || cfg == nil {
		return nil
	}
	return &MDNSDiscovery{
		ctx:  ctx,
		host: *host,
		cfg:  cfg,
	}
}

// Run initializes and starts the mDNS service.
func (m *MDNSDiscovery) Run() error {
	if !m.cfg.EnableMDNS {
		log.Println("P2P mDNS Discovery: Skipping setup as it's disabled in config.")
		return nil
	}

	log.Println("P2P mDNS Discovery: Setting up...")
	notifee := newDiscoveryNotifee(m.ctx, m.host)
	svc := mdns.NewMdnsService(m.host, m.cfg.MDNSServiceTag, notifee)
	m.service = svc

	if err := m.service.Start(); err != nil {
		log.Printf("P2P mDNS Discovery: WARN - Error starting mDNS service: %v", err)
		m.service = nil
		return err
	}

	log.Println("P2P mDNS Discovery: Service started successfully.")
	return nil
}

// Stop closes the mDNS service.
func (m *MDNSDiscovery) Stop() error {
	log.Println("P2P mDNS Discovery: Stopping...")
	if m.service == nil {
		return nil
	}

	err := m.service.Close()
	m.service = nil
	if err != nil {
		log.Printf("P2P mDNS Discovery: Error closing service: %v", err)
	} else {
		log.Println("P2P mDNS Discovery: Stopped.")
	}
	return err
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if n.ctx.Err() != nil {
		return
	}

	if pi.ID == n.h.ID() {
		return
	}
	log.Printf("P2P mDNS Discovery: Found peer %s, addrs: %v", pi.ID.ShortString(), pi.Addrs)

	if !n.shouldConnect(pi.ID) {
		log.Printf("P2P mDNS Discovery: Skipping connection to %s (waiting)", pi.ID.ShortString())
		return
	}

	n.recordConnectionAttempt(pi.ID)

	ctx, cancel := context.WithTimeout(n.ctx, 10*time.Second)
	defer cancel()

	log.Printf("P2P mDNS Discovery: Connecting to %s...", pi.ID.ShortString())
	if err := n.h.Connect(ctx, pi); err != nil {
		if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "swarm closed") {
			log.Printf("P2P mDNS Discovery: WARN - Failed connection to %s: %v", pi.ID.ShortString(), err)
		}
	} else {
		log.Printf("P2P mDNS Discovery: Connected to %s", pi.ID.ShortString())
	}
}

func (n *discoveryNotifee) shouldConnect(p peer.ID) bool {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.h.Network().Connectedness(p) == network.Connected {
		return false
	}
	lastAttempt, exists := n.connectionAttempts[p]
	if exists && time.Since(lastAttempt) < 15*time.Second {
		return false
	}
	return n.h.ID().String() < p.String()
}

func (n *discoveryNotifee) recordConnectionAttempt(p peer.ID) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.connectionAttempts[p] = time.Now()

	if len(n.connectionAttempts) > 50 {
		cutoff := time.Now().Add(-3 * time.Minute)
		for id, ts := range n.connectionAttempts {
			if ts.Before(cutoff) {
				delete(n.connectionAttempts, id)
			}
		}
	}
}
