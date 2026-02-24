// server/peer/peer.go
package peer

import (
	"encoding/json"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vinizap/lumi/server/filesystem"
	"github.com/vinizap/lumi/server/ws"
)

const reconnectDelay = 5 * time.Second

// PeerManager maintains outbound WebSocket connections to peer servers.
type PeerManager struct {
	peerURLs []string
	hub      *ws.Hub
	rootDir  string
	serverID string
}

// NewPeerManager creates a manager that will connect to the given peer URLs.
func NewPeerManager(peerURLs []string, hub *ws.Hub, rootDir, serverID string) *PeerManager {
	return &PeerManager{
		peerURLs: peerURLs,
		hub:      hub,
		rootDir:  rootDir,
		serverID: serverID,
	}
}

// Start launches a goroutine for each peer that connects and stays connected.
func (pm *PeerManager) Start() {
	for _, peerURL := range pm.peerURLs {
		go pm.connectLoop(peerURL)
	}
}

func (pm *PeerManager) connectLoop(peerURL string) {
	for {
		pm.connectToPeer(peerURL)
		log.Printf("Peer connection to %s lost, reconnecting in %v...", peerURL, reconnectDelay)
		time.Sleep(reconnectDelay)
	}
}

func (pm *PeerManager) connectToPeer(peerURL string) {
	u, err := url.Parse(peerURL)
	if err != nil {
		log.Printf("Invalid peer URL %s: %v", peerURL, err)
		return
	}

	// Add server_id query parameter
	q := u.Query()
	q.Set("server_id", pm.serverID)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("Failed to connect to peer %s: %v", peerURL, err)
		return
	}

	pm.hub.RegisterPeer(conn)
	defer pm.hub.UnregisterPeer(conn)

	log.Printf("Connected to peer %s", peerURL)

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Peer read error from %s: %v", peerURL, err)
			return
		}

		var msg ws.Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("Peer message parse error: %v", err)
			continue
		}

		// Skip messages that originated from this server (prevent echo)
		if msg.Origin == pm.serverID {
			continue
		}

		pm.applyPeerMessage(msg)
	}
}

func (pm *PeerManager) applyPeerMessage(msg ws.Message) {
	if msg.Note == nil {
		return
	}

	switch msg.Type {
	case "note_updated":
		notePath, err := filesystem.FindNotePath(pm.rootDir, msg.Note.ID)
		if err != nil {
			log.Printf("Peer sync: note %s not found locally for update", msg.Note.ID)
			return
		}
		existing, err := filesystem.ReadNote(notePath)
		if err != nil {
			log.Printf("Peer sync: failed to read %s: %v", notePath, err)
			return
		}
		existing.Title = msg.Note.Title
		existing.Content = msg.Note.Content
		existing.Tags = msg.Note.Tags
		existing.UpdatedAt = msg.Note.UpdatedAt
		if err := filesystem.WriteNote(existing); err != nil {
			log.Printf("Peer sync: failed to write %s: %v", notePath, err)
			return
		}
		pm.hub.BroadcastLocal(msg)

	case "note_created":
		note, err := filesystem.CreateNote(pm.rootDir, msg.Note.ID, msg.Note.Title)
		if err != nil {
			log.Printf("Peer sync: failed to create note %s: %v", msg.Note.ID, err)
			return
		}
		note.Content = msg.Note.Content
		note.Tags = msg.Note.Tags
		if err := filesystem.WriteNote(note); err != nil {
			log.Printf("Peer sync: failed to write created note %s: %v", msg.Note.ID, err)
			return
		}
		pm.hub.BroadcastLocal(msg)

	case "note_deleted":
		notePath, err := filesystem.FindNotePath(pm.rootDir, msg.Note.ID)
		if err != nil {
			log.Printf("Peer sync: note %s not found locally for delete", msg.Note.ID)
			return
		}
		if err := filesystem.DeleteNote(notePath); err != nil {
			log.Printf("Peer sync: failed to delete %s: %v", notePath, err)
			return
		}
		pm.hub.BroadcastLocal(msg)
	}
}
