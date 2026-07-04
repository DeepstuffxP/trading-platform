package main

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

func newHub() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]bool)}
}

func (h *Hub) addClient(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
	activeConnections.Inc()
	log.Println("client connected, total:", len(h.clients))
}

func (h *Hub) removeClient(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
	activeConnections.Dec()
	log.Println("client disconnected, total: ", len(h.clients))
}

func (h *Hub) broadcast(msg string) {
	h.mu.Lock()

	var dead []*websocket.Conn
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			dead = append(dead, conn)
		}
	}
	for _, conn := range dead {
		delete(h.clients, conn)
		activeConnections.Dec()
		conn.Close()
	}
	h.mu.Unlock()
}
