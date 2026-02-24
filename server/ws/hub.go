// server/ws/hub.go
package ws

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/vinizap/lumi/server/domain"
)

type Message struct {
	Type   string       `json:"type"`
	Note   *domain.Note `json:"note,omitempty"`
	Origin string       `json:"origin,omitempty"`
}

type Hub struct {
	clients    map[*websocket.Conn]bool
	peers      map[*websocket.Conn]bool
	broadcast  chan Message
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	registerPeer   chan *websocket.Conn
	unregisterPeer chan *websocket.Conn
	serverID   string
	mu         sync.RWMutex
}

func NewHub(serverID string) *Hub {
	return &Hub{
		clients:        make(map[*websocket.Conn]bool),
		peers:          make(map[*websocket.Conn]bool),
		broadcast:      make(chan Message, 256),
		register:       make(chan *websocket.Conn),
		unregister:     make(chan *websocket.Conn, 16),
		registerPeer:   make(chan *websocket.Conn),
		unregisterPeer: make(chan *websocket.Conn, 16),
		serverID:       serverID,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			h.mu.Unlock()

		case conn := <-h.registerPeer:
			h.mu.Lock()
			h.peers[conn] = true
			h.mu.Unlock()
			log.Printf("Peer registered (total: %d)", len(h.peers))

		case conn := <-h.unregisterPeer:
			h.mu.Lock()
			if _, ok := h.peers[conn]; ok {
				delete(h.peers, conn)
				conn.Close()
			}
			h.mu.Unlock()
			log.Printf("Peer unregistered")

		case msg := <-h.broadcast:
			// Set origin if not already set
			if msg.Origin == "" {
				msg.Origin = h.serverID
			}

			h.mu.RLock()
			for conn := range h.clients {
				if err := conn.WriteJSON(msg); err != nil {
					log.Printf("WebSocket write error: %v", err)
					h.unregister <- conn
				}
			}
			for conn := range h.peers {
				if err := conn.WriteJSON(msg); err != nil {
					log.Printf("Peer write error: %v", err)
					h.unregisterPeer <- conn
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all clients and peers.
func (h *Hub) Broadcast(msgType string, note *domain.Note) {
	h.broadcast <- Message{
		Type:   msgType,
		Note:   note,
		Origin: h.serverID,
	}
}

// BroadcastLocal sends a message to local clients only (not peers).
// Used when receiving a message from a peer to avoid echo loops.
func (h *Hub) BroadcastLocal(msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.clients {
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("WebSocket write error: %v", err)
			h.unregister <- conn
		}
	}
}

func (h *Hub) Register(conn *websocket.Conn) {
	h.register <- conn
}

func (h *Hub) Unregister(conn *websocket.Conn) {
	h.unregister <- conn
}

func (h *Hub) RegisterPeer(conn *websocket.Conn) {
	h.registerPeer <- conn
}

func (h *Hub) UnregisterPeer(conn *websocket.Conn) {
	h.unregisterPeer <- conn
}

func (h *Hub) ServerID() string {
	return h.serverID
}

func (h *Hub) HandleConnection(conn *websocket.Conn) {
	defer h.Unregister(conn)

	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		if msgType, ok := msg["type"].(string); ok && msgType == "subscribe" {
			log.Printf("Client subscribed")
		}
	}
}
