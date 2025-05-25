package main

import (
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/config"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/setup"
)

func main() {
	// --- Initial Setup ---
	// Use standard logger for initial messages
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Println("Process Starting...")

	//golog.SetAllLoggers(golog.LevelWarn)

	//golog.SetLogLevel("pubsub", "debug")
	//golog.SetLogLevel("autorelay", "debug")
	// golog.SetLogLevel("dht", "warn")

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

	// --- Run Services ---
	go app.Start()

	// --- Wait for Shutdown ---
	app.WaitForShutdown()

	app.Stop()
}
