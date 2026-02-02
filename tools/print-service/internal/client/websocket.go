// Package client provides HTTP and WebSocket clients for the print service.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"go.uber.org/zap"
)

// WSMessage represents a WebSocket message from the server.
type WSMessage struct {
	Type      string          `json:"type"`
	EventType string          `json:"event_type,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp int64           `json:"timestamp"`
	ID        string          `json:"id,omitempty"`
}

// PrintingEventPayload represents the payload for printing events.
type PrintingEventPayload struct {
	EventID        string `json:"event_id"`
	EventType      string `json:"event_type"`
	AggregateID    string `json:"aggregate_id"`
	AggregateType  string `json:"aggregate_type"`
	TenantID       string `json:"tenant_id"`
	OccurredAt     string `json:"occurred_at"`
	DocumentType   string `json:"document_type,omitempty"`
	DocumentID     string `json:"document_id,omitempty"`
	DocumentNumber string `json:"document_number,omitempty"`
	JobID          string `json:"job_id,omitempty"`
	TemplateID     string `json:"template_id,omitempty"`
	Status         string `json:"status,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	PdfURL         string `json:"pdf_url,omitempty"`
	Copies         int    `json:"copies,omitempty"`
}

// EventHandler is a function that handles incoming events.
type EventHandler func(ctx context.Context, event PrintingEventPayload) error

// WSClientConfig holds WebSocket client configuration.
type WSClientConfig struct {
	// URL is the WebSocket server URL.
	URL string

	// Token is the authentication token.
	Token string

	// ReconnectInterval is the interval between reconnection attempts.
	ReconnectInterval time.Duration

	// MaxReconnectAttempts is the maximum number of reconnection attempts.
	// 0 means unlimited.
	MaxReconnectAttempts int

	// PingInterval is the interval between ping messages.
	PingInterval time.Duration

	// PongTimeout is the timeout for pong responses.
	PongTimeout time.Duration

	// Logger is the logger instance.
	Logger *zap.Logger
}

// Constants for WebSocket client timeouts.
const (
	// reconnectCheckInterval is the interval between connection checks.
	reconnectCheckInterval = 100 * time.Millisecond
	// writeDeadlineTimeout is the timeout for write operations.
	writeDeadlineTimeout = 10 * time.Second
	// maxConcurrentHandlers limits concurrent event handler goroutines.
	maxConcurrentHandlers = 10
)

// WSClient is a WebSocket client for receiving printing events.
type WSClient struct {
	config    WSClientConfig
	conn      net.Conn
	connMu    sync.RWMutex
	handler   EventHandler
	handlerMu sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
	logger    *zap.Logger
	running   atomic.Bool
	lastPong  time.Time
	pongMu    sync.RWMutex
	workerSem chan struct{} // Semaphore for limiting concurrent handlers
}

// NewWSClient creates a new WebSocket client.
func NewWSClient(config WSClientConfig) *WSClient {
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	if config.ReconnectInterval == 0 {
		config.ReconnectInterval = 5 * time.Second
	}
	if config.PingInterval == 0 {
		config.PingInterval = 30 * time.Second
	}
	if config.PongTimeout == 0 {
		config.PongTimeout = 60 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &WSClient{
		config:    config,
		ctx:       ctx,
		cancel:    cancel,
		done:      make(chan struct{}),
		logger:    config.Logger,
		workerSem: make(chan struct{}, maxConcurrentHandlers),
	}
}

// SetHandler sets the event handler function.
// Must be called before Start().
func (c *WSClient) SetHandler(handler EventHandler) error {
	if c.running.Load() {
		return fmt.Errorf("cannot set handler while client is running")
	}
	c.handlerMu.Lock()
	c.handler = handler
	c.handlerMu.Unlock()
	return nil
}

// Start connects to the WebSocket server and starts listening for events.
func (c *WSClient) Start() error {
	if c.running.Load() {
		return fmt.Errorf("WebSocket client already running")
	}

	if err := c.connect(); err != nil {
		return fmt.Errorf("initial connection failed: %w", err)
	}

	c.running.Store(true)

	// Start goroutines
	go c.readLoop()
	go c.pingLoop()
	go c.reconnectLoop()

	c.logger.Info("WebSocket client started",
		zap.String("url", c.config.URL))

	return nil
}

// Stop stops the WebSocket client.
func (c *WSClient) Stop() {
	if !c.running.Load() {
		return
	}

	c.running.Store(false)
	c.cancel()

	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connMu.Unlock()

	close(c.done)
	c.logger.Info("WebSocket client stopped")
}

// IsConnected returns true if the client is connected.
func (c *WSClient) IsConnected() bool {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.conn != nil
}

// connect establishes a WebSocket connection.
func (c *WSClient) connect() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	// Close existing connection if any
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	// Use only the base URL - token is sent via Authorization header
	url := c.config.URL

	// Create dialer with custom headers
	dialer := ws.Dialer{
		Header: ws.HandshakeHeaderHTTP(http.Header{
			"Authorization": []string{"Bearer " + c.config.Token},
		}),
	}

	// Connect
	conn, _, _, err := dialer.Dial(c.ctx, url)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	c.conn = conn
	c.updatePong()

	c.logger.Info("WebSocket connected",
		zap.String("url", c.config.URL))

	return nil
}

// readLoop reads messages from the WebSocket connection.
func (c *WSClient) readLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.done:
			return
		default:
		}

		c.connMu.RLock()
		conn := c.conn
		c.connMu.RUnlock()

		if conn == nil {
			time.Sleep(reconnectCheckInterval)
			continue
		}

		// Set read deadline
		conn.SetReadDeadline(time.Now().Add(c.config.PongTimeout))

		// Read message
		data, op, err := wsutil.ReadServerData(conn)
		if err != nil {
			c.logger.Warn("WebSocket read error",
				zap.Error(err))
			c.handleDisconnect()
			continue
		}

		// Handle message based on opcode
		switch op {
		case ws.OpText:
			c.handleMessage(data)
		case ws.OpPing:
			c.handlePing(conn, data)
		case ws.OpPong:
			c.updatePong()
		case ws.OpClose:
			c.handleDisconnect()
		}
	}
}

// handleMessage processes a text message.
func (c *WSClient) handleMessage(data []byte) {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.logger.Warn("Failed to unmarshal WebSocket message",
			zap.Error(err))
		return
	}

	switch msg.Type {
	case "connected":
		c.logger.Info("WebSocket connection confirmed",
			zap.String("client_id", msg.ID))

	case "heartbeat":
		c.updatePong()
		c.sendPong()

	case "pong":
		c.updatePong()

	case "event":
		c.handleEvent(msg)

	default:
		c.logger.Debug("Unknown message type",
			zap.String("type", msg.Type))
	}
}

// handleEvent processes an event message.
func (c *WSClient) handleEvent(msg WSMessage) {
	c.handlerMu.RLock()
	handler := c.handler
	c.handlerMu.RUnlock()

	if handler == nil {
		c.logger.Warn("No event handler registered")
		return
	}

	var payload PrintingEventPayload
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		c.logger.Warn("Failed to unmarshal event payload",
			zap.Error(err))
		return
	}

	c.logger.Debug("Received event",
		zap.String("event_type", msg.EventType),
		zap.String("event_id", payload.EventID))

	// Use semaphore to limit concurrent handlers
	select {
	case c.workerSem <- struct{}{}:
		go func() {
			defer func() { <-c.workerSem }()
			if err := handler(c.ctx, payload); err != nil {
				c.logger.Error("Event handler error",
					zap.String("event_type", msg.EventType),
					zap.Error(err))
			}
		}()
	default:
		c.logger.Warn("Handler queue full, dropping event",
			zap.String("event_type", msg.EventType),
			zap.String("event_id", payload.EventID))
	}
}

// handlePing responds to a ping message.
func (c *WSClient) handlePing(conn net.Conn, data []byte) {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn == nil {
		return
	}

	conn.SetWriteDeadline(time.Now().Add(writeDeadlineTimeout))
	if err := wsutil.WriteClientMessage(conn, ws.OpPong, data); err != nil {
		c.logger.Warn("Failed to send pong",
			zap.Error(err))
	}
	c.updatePong() // Use updatePong instead of updatePongLocked
}

// handleDisconnect handles a disconnection.
func (c *WSClient) handleDisconnect() {
	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connMu.Unlock()

	c.logger.Warn("WebSocket disconnected")
}

// pingLoop sends periodic ping messages.
func (c *WSClient) pingLoop() {
	ticker := time.NewTicker(c.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.done:
			return
		case <-ticker.C:
			c.sendPing()
		}
	}
}

// sendPing sends a ping message.
func (c *WSClient) sendPing() {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn == nil {
		return
	}

	msg := WSMessage{
		Type:      "ping",
		Timestamp: time.Now().UnixMilli(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		c.logger.Error("Failed to marshal ping message", zap.Error(err))
		return
	}

	c.conn.SetWriteDeadline(time.Now().Add(writeDeadlineTimeout))
	if err := wsutil.WriteClientMessage(c.conn, ws.OpText, data); err != nil {
		c.logger.Warn("Failed to send ping",
			zap.Error(err))
	}
}

// sendPong sends a pong message.
func (c *WSClient) sendPong() {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn == nil {
		return
	}

	msg := WSMessage{
		Type:      "pong",
		Timestamp: time.Now().UnixMilli(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		c.logger.Error("Failed to marshal pong message", zap.Error(err))
		return
	}

	c.conn.SetWriteDeadline(time.Now().Add(writeDeadlineTimeout))
	if err := wsutil.WriteClientMessage(c.conn, ws.OpText, data); err != nil {
		c.logger.Warn("Failed to send pong",
			zap.Error(err))
	}
}

// reconnectLoop handles automatic reconnection.
func (c *WSClient) reconnectLoop() {
	attempts := 0

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.done:
			return
		default:
		}

		// Check if connected
		if c.IsConnected() {
			// Check pong timeout
			c.pongMu.RLock()
			lastPong := c.lastPong
			c.pongMu.RUnlock()

			if time.Since(lastPong) > c.config.PongTimeout {
				c.logger.Warn("Pong timeout, reconnecting")
				c.handleDisconnect()
			} else {
				attempts = 0
				time.Sleep(c.config.ReconnectInterval)
				continue
			}
		}

		// Check max attempts
		if c.config.MaxReconnectAttempts > 0 && attempts >= c.config.MaxReconnectAttempts {
			c.logger.Error("Max reconnection attempts reached",
				zap.Int("attempts", attempts))
			return
		}

		// Wait before reconnecting
		time.Sleep(c.config.ReconnectInterval)

		// Attempt reconnection
		attempts++
		c.logger.Info("Attempting reconnection",
			zap.Int("attempt", attempts))

		if err := c.connect(); err != nil {
			c.logger.Warn("Reconnection failed",
				zap.Int("attempt", attempts),
				zap.Error(err))
		} else {
			c.logger.Info("Reconnection successful",
				zap.Int("attempt", attempts))
			attempts = 0
		}
	}
}

// updatePong updates the last pong time.
func (c *WSClient) updatePong() {
	c.pongMu.Lock()
	c.lastPong = time.Now()
	c.pongMu.Unlock()
}
