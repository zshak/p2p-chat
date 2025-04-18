package discovery

//
//import (
//	"context"
//	"github.com/libp2p/go-libp2p/core/network"
//	"log"
//	"sync"
//	"time"
//
//	"github.com/libp2p/go-libp2p/core/host"
//	"github.com/libp2p/go-libp2p/core/peer"
//	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
//)
//
///* discoveryServiceTag is the unique identifier for our application's service */
///* Using .local is a convention for mDNS services */
//const discoveryServiceTag = "p2p-chat-daemon-discovery.local"
//
//
///* newDiscoveryNotifee creates a new discoveryNotifee instance */
//func newDiscoveryNotifee(h host.Host) *discoveryNotifee {
//	return &discoveryNotifee{
//		h:                  h,
//		connectionAttempts: make(map[peer.ID]time.Time),
//	}
//}
//
///* shouldConnect determines if we should attempt a connection to this peer */
///* deciding who initiates the connection is needed because there is no classic */
///* client - server communication where client initializes connection */
//func (n *discoveryNotifee) shouldConnect(p peer.ID) bool {
//	n.mutex.Lock()
//	defer n.mutex.Unlock()
//
//	// Check if we're already connected to this peer
//	if n.h.Network().Connectedness(p) == network.Connected {
//		return false
//	}
//
//	// Check if we've attempted a connection recently
//	lastAttempt, exists := n.connectionAttempts[p]
//	if exists && time.Since(lastAttempt) < 10*time.Second {
//		return false
//	}
//
//	// Based on peer IDs, decide who initiates the connection
//	return n.h.ID().String() < p.String()
//}
//
///* recordConnectionAttempt marks that we've attempted to connect to a peer */
//func (n *discoveryNotifee) recordConnectionAttempt(p peer.ID) {
//	n.mutex.Lock()
//	defer n.mutex.Unlock()
//	n.connectionAttempts[p] = time.Now()
//
//	// Clean up old entries every so often
//	if len(n.connectionAttempts) > 100 {
//		for id, timestamp := range n.connectionAttempts {
//			if time.Since(timestamp) > 5*time.Minute {
//				delete(n.connectionAttempts, id)
//			}
//		}
//	}
//}
//
///* HandlePeerFound is called by the mDNS service when a new peer is discovered. */
//func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
//	/* Don't connect to ourselves */
//	if pi.ID == n.h.ID() {
//		return
//	}
//
//	log.Printf("discovery: Found peer %s, addresses: %v", pi.ID.ShortString(), pi.Addrs)
//
//	// Determine whether we should connect based on above algorithm
//	if !n.shouldConnect(pi.ID) {
//		log.Printf("discovery: Skipping connection to %s (waiting for peer to connect to us)", pi.ID.ShortString())
//		return
//	}
//
//	// Record this connection attempt
//	n.recordConnectionAttempt(pi.ID)
//
//	/* Create a context with a timeout for the connection attempt */
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
//	defer cancel() /* Ensure context resources are released */
//
//	log.Printf("discovery: Connecting to %s ...", pi.ID.ShortString())
//	/* Attempt connection using the host stored in the notifee */
//	err := n.h.Connect(ctx, pi)
//	if err != nil {
//		log.Printf("WARN: discovery: Failed to connect to %s: %v", pi.ID.ShortString(), err)
//	} else {
//		log.Printf("discovery: Successfully connected to %s", pi.ID.ShortString())
//	}
//}
//
///* SetupMDNSDiscovery initializes and starts the mDNS discovery service */
//func SetupMDNSDiscovery(node host.Host) error {
//	log.Println("Setting up mDNS discovery...")
//
//	/* Create an instance of our custom Notifee, passing the host */
//	notifee := newDiscoveryNotifee(node)
//
//	/* Setup the mDNS service using the host, service tag, and notifee */
//	service := mdns.NewMdnsService(node, discoveryServiceTag, notifee)
//
//	/* Start the service */
//	if err := service.Start(); err != nil {
//		log.Printf("WARN: Error starting mDNS discovery service: %v", err)
//		return err /* Return the error to the caller */
//	}
//
//	log.Println("mDNS discovery service started successfully.")
//	return nil /* Return nil on success */
//}
