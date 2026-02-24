// server/main.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/vinizap/lumi/server/auth"
	httphandlers "github.com/vinizap/lumi/server/http"
	"github.com/vinizap/lumi/server/peer"
	"github.com/vinizap/lumi/server/ws"
)

func main() {
	rootDir := os.Getenv("LUMI_ROOT")
	if rootDir == "" {
		rootDir = "./notes"
	}

	port := os.Getenv("LUMI_PORT")
	if port == "" {
		port = "8080"
	}

	serverID := os.Getenv("LUMI_SERVER_ID")
	if serverID == "" {
		serverID = generateID()
	}

	hub := ws.NewHub(serverID)
	go hub.Run()

	server := httphandlers.NewServer(rootDir, hub)

	mux := http.NewServeMux()

	// CORS middleware
	corsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Lumi-Token")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next(w, r)
		}
	}

	mux.HandleFunc("/api/folders", corsMiddleware(auth.Middleware(server.HandleFolders)))
	mux.HandleFunc("/api/notes", corsMiddleware(auth.Middleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			server.HandleNotes(w, r)
		case http.MethodPost:
			server.HandleCreateNote(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	mux.HandleFunc("/api/notes/", corsMiddleware(auth.Middleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			server.HandleGetNote(w, r)
		case http.MethodPut:
			server.HandleUpdateNote(w, r)
		case http.MethodDelete:
			server.HandleDeleteNote(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	mux.HandleFunc("/ws", corsMiddleware(server.HandleWebSocket))

	// Peer WebSocket endpoint
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	mux.HandleFunc("/ws/peer", func(w http.ResponseWriter, r *http.Request) {
		peerServerID := r.URL.Query().Get("server_id")
		if peerServerID == "" {
			http.Error(w, "server_id required", http.StatusBadRequest)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		hub.RegisterPeer(conn)
		log.Printf("Inbound peer connected: %s", peerServerID)

		defer hub.UnregisterPeer(conn)
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Inbound peer %s disconnected: %v", peerServerID, err)
				return
			}

			var msg ws.Message
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}

			// Skip messages from our own server
			if msg.Origin == serverID {
				continue
			}

			// Broadcast to local clients only
			hub.BroadcastLocal(msg)
		}
	})

	// Start peer connections
	peersEnv := os.Getenv("LUMI_PEERS")
	if peersEnv != "" {
		var peerURLs []string
		for _, u := range strings.Split(peersEnv, ",") {
			u = strings.TrimSpace(u)
			if u != "" {
				peerURLs = append(peerURLs, u)
			}
		}
		if len(peerURLs) > 0 {
			pm := peer.NewPeerManager(peerURLs, hub, rootDir, serverID)
			pm.Start()
			log.Printf("Peer sync enabled with %d peer(s)", len(peerURLs))
		}
	}

	log.Printf("Server starting on :%s with root: %s (id: %s)", port, rootDir, serverID)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func generateID() string {
	// Simple random ID using crypto/rand
	b := make([]byte, 8)
	f, _ := os.Open("/dev/urandom")
	f.Read(b)
	f.Close()
	const hex = "0123456789abcdef"
	id := make([]byte, 16)
	for i, v := range b {
		id[i*2] = hex[v>>4]
		id[i*2+1] = hex[v&0x0f]
	}
	return string(id)
}
