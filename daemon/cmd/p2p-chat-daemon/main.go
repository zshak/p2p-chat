package main

import (
	"context"   /* Used for managing cancellation signals */
	"fmt"       /* For printing formatted output */
	"log"       /* For logging messages */
	"os"        /* Provides OS functionality (like signals) */
	"os/signal" /* Allows handling incoming OS signals */
	"p2p-chat-daemon/cmd/p2p-chat-daemon/discovery"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/peer"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/ui-api"
	"syscall" /* Contains low-level OS primitives (for SIGTERM) */
	"time"    /* Provides time functionality (for shutdown timeout) */

	"github.com/libp2p/go-libp2p/core/host" /* Only needed here for closeNode type hint */
)

/* Define the address for the API server. */
/* Using :0 makes the OS assign a random port */
/* Consider making this configurable via flags */
const apiAddr = "127.0.0.1:0"

/* NOTE: discoveryServiceTag is now defined in mDNS.go */

func main() {
	fmt.Println("Starting P2P Chat Daemon...")

	/* Setup Context for cancellation */
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	/* Create the libp2p Node (logic in node.go) */
	node, err := peer.CreateLibp2pNode() /* Assuming no ctx needed based on previous state */
	if err != nil {
		log.Fatalf("Failed to create libp2p node: %v", err)
	}
	/* Defer closing the node right after creation */
	defer closeNode(node)

	/* Log node details (logic presumably in node.go) */
	peer.LogNodeDetails(node)

	/* Setup mDNS discovery (logic now in mDNS.go) */
	/* We call the setup function, passing the created node */
	err = discovery.SetupMDNSDiscovery(node)
	if err != nil {
		/* Log warning but continue if mDNS fails */
		log.Printf("WARN: mDNS setup failed: %v. Local discovery might not work.", err)
	}

	/* Setup DHT-based global discovery */
	dht, err := discovery.SetupGlobalDiscovery(ctx, node)
	if err != nil {
		/* Log warning but continue if DHT setup fails */
		log.Printf("WARN: Global DHT discovery setup failed: %v. Global discovery might not work.", err)
	} else {
		/* Defer DHT shutdown for cleanup */
		defer func() {
			log.Println("Closing DHT...")
			if err := dht.Close(); err != nil {
				log.Printf("Error closing DHT: %v", err)
			} else {
				log.Println("DHT closed successfully.")
			}
		}()
	}

	/* Start the API Server */
	listener, apiServer, err := ui_api.StartAPIServer(ctx, apiAddr, node)
	if err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}

	actualApiAddr := listener.Addr().String()

	/* Setup signal handling */
	setupCloseHandler(cancel)

	log.Printf("===============================================================")
	log.Printf(" Daemon is running. API and UI accessible at: %s ", actualApiAddr)
	log.Printf("===============================================================")
	log.Println("Press Ctrl+C to stop.")

	/* Block until termination signal received */
	<-ctx.Done()

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

	/* Node shutdown handled by defer */
	log.Println("Daemon shut down gracefully.")
}

/* setupCloseHandler listens for OS interrupt signals (like Ctrl+C) */
/* and calls the cancel function to trigger a graceful shutdown. */
func setupCloseHandler(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Printf("\r- Received signal %s. Triggering shutdown...", sig)
		cancel()
	}()
}

/* closeNode provides a dedicated function to close the node, called via defer. */
func closeNode(node host.Host) {
	log.Println("Closing libp2p node...")
	if err := node.Close(); err != nil {
		log.Printf("Error closing libp2p node: %v", err)
	} else {
		log.Println("Libp2p node closed.")
	}
}

/* NOTE: discoveryNotifee struct and HandlePeerFound method are now in mDNS.go */
/* NOTE: Assuming createLibp2pNode, logNodeDetails, startAPIServer are defined */
/* in other files (node.go, server.go etc.) within the same 'main' package. */
