// Package ws provides WebSocket transport for live editing sessions.
package ws

import (
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
)

// Client represents a connected WebSocket client.
type Client struct {
	conn      *websocket.Conn
	send      chan []byte
	done      chan struct{}
	sessionID model.SessionID
}

// Hub manages all active WebSocket connections.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
	logger  *zap.Logger
}

// NewHub creates a WebSocket hub.
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
		logger:  logger,
	}
}

// Register adds a client to the hub and generates a session ID.
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	client.sessionID = model.NewSessionID()
	h.clients[client] = struct{}{}
	h.logger.Debug("ws client connected", zap.String("session_id", client.sessionID.String()), zap.Int("total", len(h.clients)))
}

// Unregister removes a client from the hub.
// Does not close client.done — read pump owns that channel.
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, client)
	h.logger.Debug("ws client disconnected", zap.Int("total", len(h.clients)))
}
