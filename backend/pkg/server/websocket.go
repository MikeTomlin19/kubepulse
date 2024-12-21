package server

import (
	"context"
	"log"
	"net/http"
	"sync"

	"kubepulse/pkg/k8s"
	"kubepulse/pkg/types"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type WSServer struct {
	k8sClient *k8s.Client
	clients   map[*websocket.Conn]bool
	mutex     sync.RWMutex
}

func NewWSServer(k8sClient *k8s.Client) *WSServer {
	return &WSServer{
		k8sClient: k8sClient,
		clients:   make(map[*websocket.Conn]bool),
	}
}

func (s *WSServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

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
	if err == nil {
		msg := types.WSMessage{
			Type:    "state",
			Payload: initialState,
		}
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("Failed to send initial state: %v", err)
			return
		}
	}

	// Listen for state updates
	for state := range stateChan {
		msg := types.WSMessage{
			Type:    "state",
			Payload: state,
		}
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("Failed to send state update: %v", err)
			return
		}
	}
}

func (s *WSServer) Start(ctx context.Context, addr string) error {
	http.HandleFunc("/ws", s.HandleWebSocket)
	server := &http.Server{Addr: addr}

	go func() {
		<-ctx.Done()
		server.Close()
	}()

	return server.ListenAndServe()
}
