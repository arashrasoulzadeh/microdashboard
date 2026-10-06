package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"microdashboard/internal/auth"
	"microdashboard/internal/store"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	store      *store.Store
	auth       *auth.Auth
}

type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	send    chan []byte
	device  *store.DeviceRow
	dashboardID string
}

func NewHub(st *store.Store, authMiddleware *auth.Auth) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		store:      st,
		auth:       authMiddleware,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected: %s", client.device.DeviceID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected: %s", client.device.DeviceID)

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastDashboardUpdate(dashboardID string, rendered interface{}) {
	data, _ := json.Marshal(map[string]interface{}{
		"type":        "dashboard_update",
		"dashboard_id": dashboardID,
		"data":        rendered,
	})
	h.broadcast <- data
}

func (h *Hub) BroadcastAlert(alert map[string]interface{}) {
	data, _ := json.Marshal(map[string]interface{}{
		"type":  "alert",
		"alert": alert,
	})
	h.broadcast <- data
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Authenticate via query param token (WebSocket can't use headers easily)
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	device, err := h.store.GetDeviceByKey(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:    h,
		conn:   conn,
		send:   make(chan []byte, 256),
		device: device,
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if msgType, ok := msg["type"].(string); ok && msgType == "subscribe" {
			if dashID, ok := msg["dashboard_id"].(string); ok {
				c.dashboardID = dashID
			}
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}
}

func (c *Client) SendForTest(data []byte) {
	select {
	case c.send <- data:
	default:
	}
}

func (c *Client) DashboardID() string {
	return c.dashboardID
}

func (c *Client) ReadPumpForTest() {
	if c.conn == nil {
		return
	}
	c.conn.SetReadLimit(512)
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if msgType, ok := msg["type"].(string); ok && msgType == "subscribe" {
			if dashID, ok := msg["dashboard_id"].(string); ok {
				c.dashboardID = dashID
			}
		}
	}
}

func NewClientForTest(hub *Hub, device *store.DeviceRow) *Client {
	return &Client{
		hub:    hub,
		send:   make(chan []byte, 256),
		device: device,
	}
}