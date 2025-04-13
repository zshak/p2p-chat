package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/libp2p/go-libp2p/core/host"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	relayv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
)

//const dhtProtocol = "/p2p-chat-daemon/kad/1.0.1"

// getHostKey loads or generates a private key
func getHostKey(keyPath string) (crypto.PrivKey, error) {
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		log.Printf("Generating new host key: %s\n", keyPath)
		priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
		if err != nil {
			return nil, err
		}
		keyBytes, err := crypto.MarshalPrivateKey(priv)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(keyPath, keyBytes, 0600); err != nil {
			return nil, fmt.Errorf("failed to write key file: %w", err)
		}
		return priv, nil
	} else {
		log.Printf("Loading host key: %s\n", keyPath)
		keyBytes, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, err
		}
		return crypto.UnmarshalPrivateKey(keyBytes)
	}
}

func main() {
	listenAddr := flag.String("listen", "/ip4/0.0.0.0/tcp/4001", "Address to listen on")
	keyFile := flag.String("key", "bootstrap-node.key", "Path to host private key file")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	privKey, err := getHostKey(*keyFile)
	if err != nil {
		log.Fatalf("Failed to get host key: %v", err)
	}

	limiter := rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits)
	rm, err := rcmgr.NewResourceManager(limiter)
	if err != nil {
		log.Fatalf("Failed to create resource manager: %v", err)
	}
	log.Println("Resource Manager created with InfiniteLimits (for testing).")
	// ------------------------------------------------------------------------

	// --- 2. Configure Relay Service Options with high/infinite limits ---
	relayServiceOpts := []relayv2.Option{
		relayv2.WithResources(relayv2.Resources{ // Use explicit Resources struct
			// Settings for the relay node itself:
			MaxReservations: 100000, // Allow many reservations
			MaxCircuits:     10000,  // Allow many simultaneous relayed connections
			BufferSize:      2048,   // Default buffer size per circuit
		}),
	}
	log.Println("Relay Service configured with high/infinite resource limits (for testing).")
	// --------------------------------------------------------------------

	// --- 3. Create the libp2p host with RM and Relay options ---
	node, err := libp2p.New(
		libp2p.ResourceManager(rm),                     // Apply the resource manager
		libp2p.EnableNATService(),                      // Still useful for AutoNAT probes
		libp2p.EnableRelayService(relayServiceOpts...), // Apply configured relay service options
		libp2p.ListenAddrStrings(*listenAddr),
		libp2p.Identity(privKey),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.Security(noise.ID, noise.New),
		// Optional: Add transports if your clients might use them to connect *to* the bootstrap
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport), // Requires UDP port open
	)
	if err != nil {
		log.Fatalf("Failed to create libp2p host: %v", err)
	}
	defer func(node host.Host) {
		err := node.Close()
		if err != nil {
			log.Fatalf("Failed to close node: %v", err)
		}
	}(node)

	log.Printf("Bootstrap Node Host created with ID: %s", node.ID())
	log.Println("Listening addresses:")
	for _, addr := range node.Addrs() {
		log.Printf("- %s/p2p/%s\n", addr, node.ID())
	}

	kadDHT, err := dht.New(ctx, node,
		dht.Mode(dht.ModeServer), // This node is a DHT server
		//dht.ProtocolPrefix(dhtProtocol),
		// No need to specify bootstrap nodes, since they will automatically
		// discover each other through clients
	)
	if err != nil {
		log.Fatalf("Failed to create DHT: %v", err)
	}

	if err = kadDHT.Bootstrap(ctx); err != nil {
		log.Fatalf("Failed to bootstrap DHT: %v", err)
	}

	log.Println("Bootstrap node DHT started successfully.")
	log.Println("Waiting for connections...")

	go PrintConnectedPeers(node)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("Shutting down bootstrap node...")
}

func PrintConnectedPeers(node host.Host) {
	for {
		time.Sleep(10 * time.Second)
		log.Printf("connected peers of Peer ID %s are:", node.ID())
		for _, peerId := range node.Network().Peers() {
			log.Printf("  %s", peerId)
		}
	}
}
