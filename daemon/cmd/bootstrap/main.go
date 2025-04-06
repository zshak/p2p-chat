package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/libp2p/go-libp2p/core/host"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
)

// The SAME protocol prefix used in your main application
const dhtProtocol = "/p2p-chat-daemon/kad/1.0.0"

// Function to load or generate a private key
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

	// Create the libp2p host
	node, err := libp2p.New(
		libp2p.ListenAddrStrings(*listenAddr),
		libp2p.Identity(privKey),
		libp2p.Security(libp2ptls.ID, libp2ptls.New), // Add TLS security transport
		libp2p.Security(noise.ID, noise.New),         // Add Noise security transport
		// Add other options like NAT management if needed (libp2p.EnableAutoRelay(), etc.)
		// but for a bootstrap node with a public IP, it might not be strictly necessary.
	)
	if err != nil {
		log.Fatalf("Failed to create libp2p host: %v", err)
	}
	defer node.Close()

	log.Printf("Bootstrap Node Host created with ID: %s", node.ID())
	log.Println("Listening addresses:")
	for _, addr := range node.Addrs() {
		log.Printf("- %s/p2p/%s\n", addr, node.ID())
	}

	// Create and start the Kademlia DHT in Server mode
	// **IMPORTANT**: Use the SAME ProtocolPrefix as your main application
	kadDHT, err := dht.New(ctx, node,
		dht.Mode(dht.ModeServer),        // This node is a DHT server
		dht.ProtocolPrefix(dhtProtocol), // Use the custom protocol prefix
		// No need to specify bootstrap peers for the bootstrap node itself,
		// unless you want bootstrap nodes to connect to each other.
		// For simplicity, we'll omit it here. They will discover each other
		// if other nodes connect to them and share routing info.
	)
	if err != nil {
		log.Fatalf("Failed to create DHT: %v", err)
	}

	// Bootstrap the DHT. In server mode, this makes it ready to respond.
	// It won't actively connect outward unless peers are provided or discovered.
	if err = kadDHT.Bootstrap(ctx); err != nil {
		log.Fatalf("Failed to bootstrap DHT: %v", err)
	}

	log.Println("Bootstrap node DHT started successfully.")
	log.Println("Waiting for connections...")

	go PrintConnectedPeers(node)

	// Keep the node running until interrupted
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
