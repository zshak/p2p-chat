package main

import (
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/config"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/setup"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Println("Process Starting...")
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: Configuration error: %v", err)
	}
	log.Println("Initializing application...")
	app, err := setup.NewApplication(cfg)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize application: %v", err)
	}
	go app.Start()
	app.WaitForShutdown()
	app.Stop()
}
