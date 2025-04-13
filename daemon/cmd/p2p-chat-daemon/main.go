package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/network"
	"log"
	"net/http"
	"os"
	"os/signal"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/discovery"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/identity"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/peer"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/ui-api"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
)

const (
	defaultKeyPath = "private-key.key"
	apiAddr        = "127.0.0.1:0"
)

func main() {
	//golog.SetLogLevel("dht", "debug")         // Very verbose DHT operations
	//golog.SetLogLevel("dht/Provide", "debug") // Focus on Provide operations
	//golog.SetLogLevel("discovery", "debug")   // Logs from routing discovery
	golog.SetLogLevel("*", "debug")
	golog.SetLogLevel("autorelay", "debug")

	usePublicBootstraps := flag.Bool("pub", false, "Use public bootstrap nodes")
	flag.Parse()

	fmt.Println("Starting P2P Chat Daemon...")

	/* Setup Context for cancellation */
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err, keyPath := getOrCreateAppDataDir()

	appState := core.NewAppState(keyPath)

	apiServer := SetUpUiApi(err, ctx, appState)

	go initializeP2P(ctx, appState, *usePublicBootstraps)

	//WaitForUserAuthenticationOrRegistration(appState)

	setupCloseHandler(cancel, ctx)

	HandleShutdown(apiServer, appState)
}

func HandleShutdown(apiServer *http.Server, state *core.AppState) {
	/* --- Graceful Shutdown Sequence --- */
	log.Println("Shutting down daemon...")

	/* Shutdown API server */
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	log.Println("Shutting down API server...")
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("API server shutdown error: %v", err)
	} else {
		log.Println("API server stopped.")
	}

	state.Mu.Lock()

	closeNode(state.Node)

	log.Println("Daemon shut down gracefully.")
}

/* setupCloseHandler listens for OS interrupt signals (like Ctrl+C) */
/* and calls the cancel function to trigger a graceful shutdown. */
func setupCloseHandler(cancel context.CancelFunc, ctx context.Context) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Printf("\r- Received signal %s. Triggering shutdown...", sig)
		cancel()
	}()

	/* Block until termination signal received */
	<-ctx.Done()
}

func WaitForUserAuthenticationOrRegistration(appState *core.AppState) {
	appState.Mu.Lock()
	if identity.KeyExists(appState.KeyPath) {
		appState.State = core.StateWaitingForPassword
		log.Printf("Key file found at %s. Waiting for password via API.", appState.KeyPath)
	} else {
		appState.State = core.StateWaitingForKey
		log.Printf("Key file not found at %s. Waiting for key setup via API.", appState.KeyPath)
	}
	appState.Mu.Unlock()
}

func SetUpUiApi(err error, ctx context.Context, appState *core.AppState) *http.Server {
	/* Start the API Server */
	listener, apiServer, err := ui_api.StartAPIServer(ctx, apiAddr, appState)
	if err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}

	actualApiAddr := listener.Addr().String()

	log.Printf("===============================================================")
	log.Printf(" Daemon is running. API and UI accessible at: %s ", actualApiAddr)
	log.Printf("===============================================================")
	log.Println("Press Ctrl+C to stop.")
	return apiServer
}

func getOrCreateAppDataDir() (error, string) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Could not determine user config directory: %v", err)
	}
	appDataDir := filepath.Join(configDir, "p2p-chat-daemon")
	if err := os.MkdirAll(appDataDir, 0700); err != nil {
		log.Fatalf("Could not create app data directory %s: %v", appDataDir, err)
	}
	keyPath := filepath.Join(appDataDir, defaultKeyPath)
	return err, keyPath
}

// This function runs in a separate goroutine and waits for the key signal
func initializeP2P(ctx context.Context, appState *core.AppState, usePublicDHT bool) {
	//log.Println("P2P Initializer: Waiting for key and password signal...")
	//select {
	//case <-appState.KeyReadyChan:
	//	log.Println("P2P Initializer: Key signal received.")
	//case <-ctx.Done():
	//	log.Println("P2P Initializer: Shutdown signal received before key was ready.")
	//	return
	//}
	//
	//appState.Mu.Lock()
	//if appState.PrivKey == nil {
	//	appState.State = core.StateError
	//	appState.LastError = fmt.Errorf("key signal received but private key is nil")
	//	log.Println("P2P Initializer: ERROR - Key signal received but private key is nil")
	//	appState.Mu.Unlock()
	//	panic(appState.LastError)
	//}
	//
	//appState.State = core.StateInitializingP2P
	privKey := appState.PrivKey
	//appState.Mu.Unlock()

	log.Println("P2P Initializer: Creating libp2p node...")
	node, err := peer.CreateLibp2pNode(privKey, appState)
	if err != nil {
		log.Printf("P2P Initializer: ERROR - Failed to create libp2p node: %v", err)
		appState.Mu.Lock()
		appState.State = core.StateError
		appState.LastError = err
		appState.Mu.Unlock()
		panic(appState.LastError)
	}

	appState.Mu.Lock()
	appState.Node = node
	appState.Mu.Unlock()

	log.Printf("P2P Initializer: Registering chat protocol handler (%s)...", core.ChatProtocolID)
	node.SetStreamHandler(core.ChatProtocolID, chatStreamHandler)

	peer.LogNodeDetails(node)

	// Setup mDNS Discovery
	//log.Println("P2P Initializer: Setting up mDNS discovery...")
	//err = discovery.SetupMDNSDiscovery(node)
	//if err != nil {
	//	log.Printf("P2P Initializer: WARN - mDNS setup failed: %v", err)
	//}

	// Setup DHT Discovery
	log.Println("P2P Initializer: Setting up DHT discovery...")
	dht, err := discovery.SetupGlobalDiscovery(ctx, node, usePublicDHT)
	if err != nil {
		log.Printf("P2P Initializer: WARN - Global DHT discovery setup failed: %v", err)
		panic(err)
	} else {
		log.Println("P2P Initializer: DHT setup successful.")
		appState.Mu.Lock()
		appState.Dht = dht // Store DHT instance
		appState.Mu.Unlock()
	}

	// P2P setup complete
	appState.Mu.Lock()
	if appState.State != core.StateShuttingDown && appState.State != core.StateError {
		appState.State = core.StateRunning
		log.Println("P2P Initializer: Node is now running.")
	}
	appState.Mu.Unlock()
}

/* closeNode provides a dedicated function to close the node, called via defer. */
func closeNode(node host.Host) {
	if node != nil {
		log.Println("Closing libp2p node...")
		if err := node.Close(); err != nil {
			log.Printf("Error closing libp2p node: %v", err)
		} else {
			log.Println("Libp2p node closed.")
		}
	}
}

// chatStreamHandler handles incoming chat streams.
func chatStreamHandler(stream network.Stream) {
	peerID := stream.Conn().RemotePeer()
	log.Printf("Chat: Received new stream from %s", peerID.ShortString())

	// Use a buffered reader for efficiency
	reader := bufio.NewReader(stream)

	// Read the message (assuming one message per stream, ending with newline for this simple example)
	message, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Chat: Error reading from stream from %s: %v", peerID.ShortString(), err)
		stream.Reset() // Abruptly close the stream on error
		return
	}

	// Trim trailing newline
	message = strings.TrimSpace(message)

	// Log the received message (replace with actual message handling later)
	log.Printf("Chat: Received message from %s: <<< %s >>>", peerID.ShortString(), message)

	// For this simple test, we can just close the stream after reading.
	// Alternatively, the sender could close it.
	stream.Close() // Gracefully close our side
}
