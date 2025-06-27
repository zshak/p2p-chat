package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
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

const dhtProtocol = "/p2p-chat-daemon/kad/1.0.0"

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
	connectAddr := flag.String("connect", "", "Multiaddress of a peer to connect to")
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
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.Security(noise.ID, noise.New),
		libp2p.EnableRelay(),
		libp2p.EnableRelayService(),
		libp2p.ForceReachabilityPublic(),
		libp2p.EnableNATService(),
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

	if *connectAddr != "" {
		log.Printf("Attempting to connect to peer: %s\n", *connectAddr)
		targetAddr, err := multiaddr.NewMultiaddr(*connectAddr)
		if err != nil {
			log.Fatalf("Failed to parse connect multiaddress: %v", err)
		}

		// Extract the peer ID from the multiaddress
		addrInfo, err := peer.AddrInfoFromP2pAddr(targetAddr)
		if err != nil {
			log.Fatalf("Failed to get peer AddrInfo from multiaddress: %v", err)
		}

		err = node.Connect(ctx, *addrInfo)
		if err != nil {
			log.Printf("Failed to connect to peer %s: %v", addrInfo.ID, err)
		} else {
			log.Printf("Successfully connected to peer: %s\n", addrInfo.ID)
		}
	}

	kadDHT, err := dht.New(ctx, node,
		dht.Mode(dht.ModeServer), // This node is a DHT server
		dht.ProtocolPrefix(dhtProtocol),
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
