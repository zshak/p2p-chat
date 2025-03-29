package main

import (
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p"           // The main libp2p package
	"github.com/libp2p/go-libp2p/core/host" // The Host interface definition
	"github.com/multiformats/go-multiaddr"  // For parsing and creating multiaddresses
)

/* createLibp2pNode initializes and returns a new libp2p Host */
func createLibp2pNode() (host.Host, error) {
	log.Println("Initializing libp2p node...")

	// Define the listening addresses for the node.
	// Multiaddr is a format for representing network addresses used by libp2p.
	// "/ip4/0.0.0.0/tcp/0" means:
	// - /ip4/0.0.0.0: Listen on all available IPv4 interfaces.
	// - /tcp/0: Use the TCP protocol and let the OS choose a random available port.
	listenAddr, err := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
	if err != nil {
		// If we can't even parse this basic address, something is wrong.
		log.Printf("Error creating listen multiaddr: %v", err)
		return nil, fmt.Errorf("failed to parse listen multiaddr: %w", err)
	}

	// libp2p.New is the primary function to create a libp2p Host (our node).
	// It takes Option functions to configure the node.
	node, err := libp2p.New(
		// libp2p.ListenAddrs specifies which addresses the node should listen on.
		libp2p.ListenAddrs(listenAddr),

		// More options will be added here later:
		// - libp2p.Identity(...) to load/save a persistent node identity.
		// - libp2p.DefaultSecurity for encrypted connections.
		// - libp2p.NATPortMap() or other NAT traversal options.
		// - libp2p.Transport(...) for specific network transports (TCP, QUIC, WebSockets).
		// - libp2p.Routing(...) for peer discovery (e.g., DHT).
	)

	// Check if node creation failed.
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err) // Wrap error for context
	}

	return node, nil // Return the created node and nil error on success
}

// logNodeDetails prints the node's connection information.
// This function now resides in node.go, logically grouped with node creation.
func logNodeDetails(node host.Host) {
	log.Printf("Node setup successful!")
	log.Printf("Node Peer ID: %s", node.ID()) // node.ID() returns peer.ID which has a String() method
	log.Printf("Connect to me on:")
	// Iterate through the addresses the node is listening on.
	for _, addr := range node.Addrs() {
		// Print the full multiaddress including the Peer ID.
		log.Printf("  %s/p2p/%s", addr, node.ID())
	}
}
