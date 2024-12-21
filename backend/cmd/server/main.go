package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"kubepulse/pkg/k8s"
	"kubepulse/pkg/server"
)

func main() {
	// Create Kubernetes client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Create WebSocket server
	wsServer := server.NewWSServer(k8sClient)

	// Create context that will be canceled on SIGINT/SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Start watching cluster state
	k8sClient.StartWatching(ctx)

	// Get server address from environment or use default
	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("Starting WebSocket server on %s", addr)
	if err := wsServer.Start(ctx, addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
