package config

import (
	"errors"
	"flag"
	"fmt"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"log"
	"os"
	"path/filepath"
)

const (
	defaultKeyFileName = "private-key.key"
	defaultAPIAddr     = "127.0.0.1:0"
)

var defaultBootstrapNodes = []string{
	fmt.Sprintf("/ip4/13.61.254.164/tcp/4001/p2p/12D3KooWFujV1a69zhXj7DZeQGKh96ubEVvPBqptHAGYpd6TGdFn"),
	fmt.Sprintf("/ip4/51.21.217.209/tcp/4001/p2p/12D3KooWDW4onEGqyg7Tu9HP8zgnJKZvbo2hgPin63XSVVTsd2eN"),
}

// P2PConfig holds settings related to the libp2p node and network.
type P2PConfig struct {
	ListenAddrs         []string
	BootstrapPeers      []peer.AddrInfo
	UsePublicBootstraps bool
	PrivateKeyPath      string
	DHTProtocolID       string // Only used if not using public bootstrap nodes
	DiscoveryServiceID  string
	EnableMDNS          bool
	MDNSServiceTag      string
}

// APIConfig holds settings for the control API.
type APIConfig struct {
	ListenAddr string
}

// Config holds the overall application configuration.
type Config struct {
	P2P P2PConfig
	API APIConfig
}

// Load reads configuration from flags/env/files.
func Load() (*Config, error) {
	usePubBootstraps := flag.Bool("pub", false, "Use public bootstrap nodes instead of private ones.")
	dhtProtoID := flag.String("dhtproto", "/p2p-chat-daemon/kad/1.0.0", "DHT Protocol ID (used only if -private=true).")
	discoverySvcID := flag.String("discoverysvc", "p2p-chat-daemon", "Service name tag for DHT discovery.")
	enableMDNS := flag.Bool("mdns", true, "Enable mDNS local discovery.")
	mdnsTag := flag.String("mdnstag", "p2p-chat-daemon.local", "Service tag for mDNS discovery.")
	// Example: Add flag for API address if needed
	apiListenAddr := flag.String("api", defaultAPIAddr, "Host and port for the API server (e.g., 127.0.0.1:0)")

	flag.Parse()

	// Determine App Data Directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine user config directory: %w", err)
	}

	appDataDir := filepath.Join(configDir, "p2p-chat-daemon")
	if err := os.MkdirAll(appDataDir, 0700); err != nil {
		return nil, fmt.Errorf("could not create app data directory %s: %w", appDataDir, err)
	}

	keyPath := filepath.Join(appDataDir, defaultKeyFileName)

	var bootstrapPeers []peer.AddrInfo
	if *usePubBootstraps {
		log.Println("Using Public Libp2p Bootstrap Peers.")
		bootstrapPeers = dht.GetDefaultBootstrapPeerAddrInfos()
	} else {
		bootstrapPeers = AddrInfosFromStrings(defaultBootstrapNodes)
		if len(bootstrapPeers) == 0 {
			return nil, errors.New("no valid private bootstrap peers could be parsed from -peers flag")
		}
		log.Printf("Using Private Bootstrap Peers.")
	}

	cfg := &Config{
		P2P: P2PConfig{
			ListenAddrs: []string{ // Default listeners
				"/ip4/0.0.0.0/tcp/0",
				"/ip6/::/tcp/0",
				"/ip4/0.0.0.0/udp/0/quic-v1",
				"/ip6/::/udp/0/quic-v1",
				// Add WebRTC/WebTransport/WS
			},
			BootstrapPeers:      bootstrapPeers,
			UsePublicBootstraps: *usePubBootstraps,
			PrivateKeyPath:      keyPath,
			DHTProtocolID:       *dhtProtoID,
			DiscoveryServiceID:  *discoverySvcID,
			EnableMDNS:          *enableMDNS,
			MDNSServiceTag:      *mdnsTag,
		},
		API: APIConfig{
			ListenAddr: *apiListenAddr,
		},
	}

	return cfg, nil
}

// AddrInfosFromStrings parses a slice of multiaddr strings into AddrInfo objects.
func AddrInfosFromStrings(addrStrings []string) []peer.AddrInfo {
	var addrInfos []peer.AddrInfo
	for _, addrStr := range addrStrings {
		addrInfo, err := peer.AddrInfoFromString(addrStr)
		if err != nil {
			log.Printf("Error parsing bootstrap peer addr %s: %v", addrStr, err)
			continue
		}
		addrInfos = append(addrInfos, *addrInfo)
	}
	return addrInfos
}
