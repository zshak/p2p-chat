package p2p

import (
	"context"
	"errors"
	"fmt"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/protocol"
	"log"
	"p2p-chat-daemon/cmd/config"
)

// DHT manages the Kademlia Distributed Hash Table instance.
type DHT struct {
	instance *dht.IpfsDHT
	host     host.Host         // Dependency
	cfg      *config.P2PConfig // Dependency
}

// NewDHT creates and bootstraps the DHT.
func NewDHT(ctx context.Context, cfg *config.P2PConfig, host host.Host) (*DHT, error) {
	log.Println("P2P DHT: Initializing...")
	if host == nil {
		return nil, errors.New("host is required for DHT")
	}

	opts := []dhtopts.Option{
		dht.Mode(dht.ModeAuto),
		dht.BootstrapPeers(cfg.BootstrapPeers...),
	}
	if !cfg.UsePublicBootstraps {
		log.Printf("P2P DHT: Configuring for Private Network (Protocol: %s)", cfg.DHTProtocolID)
		opts = append(opts, dht.ProtocolPrefix(protocol.ID(cfg.DHTProtocolID)))
	}

	kadDHT, err := dht.New(ctx, host, opts...)
	if err != nil {
		return nil, fmt.Errorf("dht.New failed: %w", err)
	}

	log.Println("P2P DHT: Bootstrapping...")
	if err = kadDHT.Bootstrap(ctx); err != nil {
		kadDHT.Close()
		return nil, fmt.Errorf("dht bootstrap failed: %w", err)
	}
	log.Println("P2P DHT: Bootstrap successful.")

	return &DHT{instance: kadDHT, host: host, cfg: cfg}, nil
}

// Instance returns the underlying DHT instance.
func (d *DHT) Instance() *dht.IpfsDHT {
	return d.instance
}

// Close shuts down the DHT instance.
func (d *DHT) Close() error {
	log.Println("P2P DHT: Closing...")
	if d.instance == nil {
		return nil
	}
	err := d.instance.Close()
	if err == nil {
		log.Println("P2P DHT: Closed.")
	} else {
		log.Printf("P2P DHT: Error closing: %v", err)
	}
	return err
}
