package sync

import (
	"encoding/json"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vinizap/lumi/tui-client/config"
)

// Event represents a sync event received from the server.
type Event struct {
	Type string          `json:"type"`
	Note json.RawMessage `json:"note,omitempty"`
}

// Client manages a WebSocket connection to the lumi server for real-time sync.
type Client struct {
	cfg     *config.FolderConfig
	eventCh chan Event
	done    chan struct{}
}

// NewClient creates a new sync client from a folder config.
// Returns nil if no server is configured.
func NewClient(cfg *config.FolderConfig) *Client {
	if cfg == nil || cfg.ServerURL == "" {
		return nil
	}
	return &Client{
		cfg:     cfg,
		eventCh: make(chan Event, 32),
		done:    make(chan struct{}),
	}
}

// Start begins the WebSocket connection in a background goroutine with auto-reconnect.
func (c *Client) Start() {
	go c.connectLoop()
}

// Stop terminates the sync client.
func (c *Client) Stop() {
	close(c.done)
}

// Events returns the channel that receives sync events.
func (c *Client) Events() <-chan Event {
	return c.eventCh
}

func (c *Client) connectLoop() {
	for {
		select {
		case <-c.done:
			return
		default:
		}

		c.connect()

		// Wait before reconnecting
		select {
		case <-c.done:
			return
		case <-time.After(3 * time.Second):
		}
	}
}

func (c *Client) connect() {
	wsURL := c.wsURL()
	if wsURL == "" {
		return
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Printf("sync: failed to connect to %s: %v", wsURL, err)
		return
	}
	defer conn.Close()

	// Send subscribe message
	conn.WriteJSON(map[string]string{"type": "subscribe"})

	for {
		select {
		case <-c.done:
			return
		default:
		}

		_, raw, err := conn.ReadMessage()
		if err != nil {
			log.Printf("sync: connection lost: %v", err)
			return
		}

		var evt Event
		if err := json.Unmarshal(raw, &evt); err != nil {
			continue
		}

		select {
		case c.eventCh <- evt:
		default:
			// Drop event if channel is full to avoid blocking
		}
	}
}

func (c *Client) wsURL() string {
	serverURL := c.cfg.ServerURL
	if serverURL == "" {
		return ""
	}

	// Convert http(s) to ws(s)
	wsURL := strings.Replace(serverURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)

	u, err := url.Parse(wsURL + "/ws")
	if err != nil {
		return ""
	}

	return u.String()
}
