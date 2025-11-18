package api

import (
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// ClientMessage represents a message from client to server
type ClientMessage struct {
	Action string `json:"action"` // "subscribe", "unsubscribe", "ping"
	TaskID string `json:"task_id"`
}

// ServerMessage represents a message from server to client
type ServerMessage struct {
	Type    string `json:"type"` // "log", "complete", "error"
	TaskID  string `json:"task_id"`
	Content string `json:"content"`
	Time    string `json:"time"`
}

// Client represents a connected WebSocket client
type Client struct {
	conn           *websocket.Conn
	subscribedTask string
	lastActivity   time.Time
	send           chan ServerMessage
	mu             sync.Mutex
}

// WebSocketHub manages all WebSocket connections and broadcasts
type WebSocketHub struct {
	// Map of client ID to client
	clients map[*Client]bool

	// Map of task ID to list of subscribed clients
	taskSubscribers map[string][]*Client

	// Register/unregister channels
	register   chan *Client
	unregister chan *Client

	mu     sync.RWMutex
	stopCh chan struct{}
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	hub := &WebSocketHub{
		clients:         make(map[*Client]bool),
		taskSubscribers: make(map[string][]*Client),
		register:        make(chan *Client, 16),
		unregister:      make(chan *Client, 16),
		stopCh:          make(chan struct{}),
	}

	go hub.run()
	go hub.cleanupIdleClients()

	return hub
}

// run handles the main event loop
func (h *WebSocketHub) run() {
	for {
		select {
		case <-h.stopCh:
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client registered")

		case client := <-h.unregister:
			h.removeClient(client)
		}
	}
}

// removeClient removes a client from all subscriptions
func (h *WebSocketHub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; !ok {
		return
	}

	delete(h.clients, client)

	if client.subscribedTask != "" {
		clients := h.taskSubscribers[client.subscribedTask]
		for i, c := range clients {
			if c == client {
				h.taskSubscribers[client.subscribedTask] = append(clients[:i], clients[i+1:]...)
				break
			}
		}

		if len(h.taskSubscribers[client.subscribedTask]) == 0 {
			delete(h.taskSubscribers, client.subscribedTask)
		}

		log.Printf("Client unsubscribed from task %s, remaining clients: %d",
			client.subscribedTask, len(h.taskSubscribers[client.subscribedTask]))
	}

	close(client.send)
}

// subscribeClient subscribes a client to a task
func (h *WebSocketHub) subscribeClient(client *Client, taskID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Unsubscribe from previous task if any
	if client.subscribedTask != "" && client.subscribedTask != taskID {
		clients := h.taskSubscribers[client.subscribedTask]
		for i, c := range clients {
			if c == client {
				h.taskSubscribers[client.subscribedTask] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
	}

	// Subscribe to new task
	client.subscribedTask = taskID
	client.lastActivity = time.Now()
	h.taskSubscribers[taskID] = append(h.taskSubscribers[taskID], client)

	log.Printf("Client subscribed to task %s, total subscribers: %d",
		taskID, len(h.taskSubscribers[taskID]))
}

// sendToTaskSubscribers sends a message to all clients subscribed to the task
func (h *WebSocketHub) sendToTaskSubscribers(taskID string, msg ServerMessage) {
	h.mu.RLock()
	clients := make([]*Client, len(h.taskSubscribers[taskID]))
	copy(clients, h.taskSubscribers[taskID])
	h.mu.RUnlock()

	if len(clients) == 0 {
		return
	}

	// Send to all subscribers
	for _, client := range clients {
		select {
		case client.send <- msg:
			client.mu.Lock()
			client.lastActivity = time.Now()
			client.mu.Unlock()
		default:
			// Channel full, client is slow, skip
			log.Printf("Warning: Client send channel full for task %s", taskID)
		}
	}
}

// BroadcastLog broadcasts a log message to all clients watching a task
func (h *WebSocketHub) BroadcastLog(taskID, content string) {
	msg := ServerMessage{
		Type:    "log",
		TaskID:  taskID,
		Content: content,
		Time:    time.Now().Format(time.RFC3339),
	}
	h.sendToTaskSubscribers(taskID, msg)
}

// BroadcastTaskComplete notifies clients that a task has completed
func (h *WebSocketHub) BroadcastTaskComplete(taskID string) {
	msg := ServerMessage{
		Type:   "complete",
		TaskID: taskID,
		Time:   time.Now().Format(time.RFC3339),
	}
	h.sendToTaskSubscribers(taskID, msg)

	// Close connections after a delay to ensure message delivery
	time.AfterFunc(2*time.Second, func() {
		h.closeTaskConnections(taskID)
	})
}

// closeTaskConnections closes all WebSocket connections for a specific task
func (h *WebSocketHub) closeTaskConnections(taskID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients := h.taskSubscribers[taskID]
	for _, client := range clients {
		// Send close message
		select {
		case client.send <- ServerMessage{
			Type:   "close",
			TaskID: taskID,
		}:
		default:
		}
	}

	// Remove all subscribers
	delete(h.taskSubscribers, taskID)
	log.Printf("Closed all connections for task %s", taskID)
}

// cleanupIdleClients periodically checks for idle clients and closes them
func (h *WebSocketHub) cleanupIdleClients() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.checkIdleClients()
		}
	}
}

// checkIdleClients removes clients that have been idle for too long
func (h *WebSocketHub) checkIdleClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	idleTimeout := 5 * time.Minute
	now := time.Now()

	for taskID, clients := range h.taskSubscribers {
		activeClients := make([]*Client, 0, len(clients))

		for _, client := range clients {
			client.mu.Lock()
			lastActivity := client.lastActivity
			client.mu.Unlock()

			if now.Sub(lastActivity) > idleTimeout {
				log.Printf("Closing idle client for task %s (last activity: %v ago)",
					taskID, now.Sub(lastActivity))
				close(client.send)
				delete(h.clients, client)
			} else {
				activeClients = append(activeClients, client)
			}
		}

		if len(activeClients) == 0 {
			delete(h.taskSubscribers, taskID)
		} else {
			h.taskSubscribers[taskID] = activeClients
		}
	}
}

// Stop stops the WebSocket hub
func (h *WebSocketHub) Stop() {
	close(h.stopCh)
}

// HandleWebSocket handles WebSocket connections
func (s *Server) HandleWebSocket(c *fiber.Ctx) error {
	return websocket.New(func(conn *websocket.Conn) {
		defer conn.Close()

		// Create client
		client := &Client{
			conn:         conn,
			lastActivity: time.Now(),
			send:         make(chan ServerMessage, 16),
		}

		// Register client
		s.wsHub.register <- client

		// Start write pump
		go client.writePump(s.wsHub)

		// Read pump (blocking)
		client.readPump(s.wsHub)

		// Unregister when done
		s.wsHub.unregister <- client
	})(c)
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump(hub *WebSocketHub) {
	for {
		var msg ClientMessage
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		c.mu.Lock()
		c.lastActivity = time.Now()
		c.mu.Unlock()

		switch msg.Action {
		case "subscribe":
			if msg.TaskID != "" {
				hub.subscribeClient(c, msg.TaskID)

				// Send acknowledgment
				c.send <- ServerMessage{
					Type:   "subscribed",
					TaskID: msg.TaskID,
					Time:   time.Now().Format(time.RFC3339),
				}
			}

		case "unsubscribe":
			hub.unregister <- c

		case "ping":
			c.send <- ServerMessage{
				Type: "pong",
				Time: time.Now().Format(time.RFC3339),
			}
		}
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump(hub *WebSocketHub) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				// Channel closed
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if msg.Type == "close" {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteJSON(msg)
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
