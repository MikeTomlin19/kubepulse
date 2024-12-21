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
	// Create context that will be canceled on SIGINT/SIGTERM
	ctx, cancel := context.WithCancel(context.Background())

	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Handle shutdown signals
	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, initiating shutdown...", sig)
		cancel()
	}()

	// Create Kubernetes client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Create WebSocket server
	wsServer := server.NewWSServer(k8sClient)

	// Start watching cluster state
	k8sClient.StartWatching(ctx)

	// Get server address from environment or use default
	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	// Start the server
	if err := wsServer.Start(ctx, addr); err != nil {
		log.Printf("Server error: %v", err)
		cancel()
	}

	// Wait for shutdown to complete
	<-ctx.Done()
	log.Println("Shutdown complete")
}
