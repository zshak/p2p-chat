package main

import (
	"log"
	"os"
	"p2p-chat-daemon/cmd/config"

	golog "github.com/ipfs/go-log/v2"
)

func main() {
	// --- Initial Setup ---
	// Use standard logger for initial messages
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Println("Process Starting...")

	// Configure structured logging (optional but good)
	//golog.SetAllLoggers(golog.LevelWarn)  // Sensible default
	golog.SetLogLevel("main", "info")     // Log info from main package
	golog.SetLogLevel("p2p", "info")      // Log info from p2p package
	golog.SetLogLevel("api", "info")      // Log info from api package
	golog.SetLogLevel("identity", "info") // Log info from identity package
	// Examples for debugging:
	// golog.SetLogLevel("autonat", "debug")
	// golog.SetLogLevel("autorelay", "debug")
	// golog.SetLogLevel("dht", "warn") // Reduce DHT noise unless debugging

	// --- Load Configuration ---
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: Configuration error: %v", err)
	}

	// --- Create Application Coordinator & Services ---
	log.Println("Initializing application...")
	app, err := NewApp(cfg) // Assumes App struct and NewApp are in this 'main' package
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize application: %v", err)
	}

	// --- Start Services (in background) ---
	if err := app.Start(); err != nil {
		log.Fatalf("FATAL: Failed to start application services: %v", err)
	}

	// --- Wait for Shutdown ---
	app.WaitForShutdown() // Blocks until Ctrl+C or fatal internal error

	// --- Stop Services ---
	log.Println("Process Stopping...")
	if err := app.Stop(); err != nil {
		log.Printf("ERROR: Shutdown completed with error: %v", err)
		os.Exit(1) // Indicate error on exit
	}

	log.Println("Process Exited Gracefully.")
}
