package main

import (
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/config"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/setup"

	golog "github.com/ipfs/go-log/v2"
)

func main() {
	// --- Initial Setup ---
	// Use standard logger for initial messages
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Println("Process Starting...")

	// Configure structured logging (optional but good)
	//golog.SetAllLoggers(golog.LevelWarn)  // Sensible default

	golog.SetLogLevel("autonat", "debug")
	golog.SetLogLevel("autorelay", "debug")
	// golog.SetLogLevel("dht", "warn") // Reduce DHT noise unless debugging

	// --- Load Configuration ---
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: Configuration error: %v", err)
	}

	// --- Create Application Coordinator & Services ---
	log.Println("Initializing application...")
	app, err := setup.NewApplication(cfg)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize application: %v", err)
	}

	// --- Start Services (in background) ---
	if err := app.Start(); err != nil {
		log.Fatalf("FATAL: Failed to start application services: %v", err)
	}

	// --- Wait for Shutdown ---
	//app.WaitForShutdown() // Blocks until Ctrl+C or fatal internal error

	// --- Stop Services ---
	//log.Println("Process Stopping...")
	//if err := app.Stop(); err != nil {
	//	log.Printf("ERROR: Shutdown completed with error: %v", err)
	//	os.Exit(1) // Indicate error on exit
	//}

	log.Println("Process Exited Gracefully.")
}
