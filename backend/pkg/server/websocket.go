package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"kubepulse/pkg/k8s"
	"kubepulse/pkg/types"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WSServer struct {
	k8sClient *k8s.Client
	clients   map[*websocket.Conn]bool
	mutex     sync.RWMutex
	server    *http.Server
}

func NewWSServer(k8sClient *k8s.Client) *WSServer {
	return &WSServer{
		k8sClient: k8sClient,
		clients:   make(map[*websocket.Conn]bool),
	}
}

func (s *WSServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Add CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check if it's a WebSocket upgrade request
	if r.Header.Get("Upgrade") != "websocket" {
		http.Error(w, "Not a websocket handshake", http.StatusBadRequest)
		return
	}

	// Upgrade the connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Set write deadline for initial state
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	// Configure WebSocket
	conn.SetPingHandler(func(data string) error {
		return conn.WriteControl(websocket.PongMessage, []byte(data), time.Now().Add(time.Second))
	})

	s.mutex.Lock()
	s.clients[conn] = true
	s.mutex.Unlock()

	// Subscribe to cluster state updates
	stateChan := s.k8sClient.Subscribe()

	// Clean up on disconnect
	defer func() {
		s.mutex.Lock()
		delete(s.clients, conn)
		s.mutex.Unlock()
		s.k8sClient.Unsubscribe(stateChan)
		conn.Close()
	}()

	// Send initial state
	initialState, err := s.k8sClient.GetClusterState()
	if err != nil {
		log.Printf("Failed to get initial state: %v", err)
	} else {
		msg := types.WSMessage{
			Type:    "state",
			Payload: initialState,
		}

		// Marshal the message outside of WriteJSON to check for errors
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal initial state: %v", err)
		} else {
			err = conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				log.Printf("Failed to send initial state: %v", err)
				return
			}
		}
	}

	// Reset write deadline for subsequent messages
	conn.SetWriteDeadline(time.Time{})

	// Start a goroutine to send periodic pings
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second)); err != nil {
					log.Printf("Failed to send ping: %v", err)
					return
				}
			}
		}
	}()

	// Listen for state updates
	for state := range stateChan {
		msg := types.WSMessage{
			Type:    "state",
			Payload: state,
		}

		// Set write deadline for each message
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("Failed to send state update: %v", err)
			return
		}

		// Reset write deadline
		conn.SetWriteDeadline(time.Time{})
	}
}

func (s *WSServer) Start(ctx context.Context, addr string) error {
	mux := http.NewServeMux()

	// Add a health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// WebSocket endpoint
	mux.HandleFunc("/ws", s.HandleWebSocket)

	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Create a channel to signal when the server has started
	serverStarted := make(chan struct{})

	// Create a channel for server errors
	errChan := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		log.Printf("Starting WebSocket server on %s", addr)
		serverStarted <- struct{}{} // Signal that we're about to start
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for server to start or context to be cancelled
	select {
	case <-serverStarted:
		// Server started successfully
	case err := <-errChan:
		// Server failed to start
		return err
	case <-ctx.Done():
		// Context was cancelled before server could start
		return ctx.Err()
	}

	// Wait for shutdown signal
	go func() {
		<-ctx.Done()
		log.Println("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
		}
	}()

	// Wait for either an error or context cancellation
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return nil
	}
}
