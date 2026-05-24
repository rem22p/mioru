package handler

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"mioru/internal/auth"
	"mioru/internal/model"
	"mioru/internal/store"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		allowed := map[string]bool{
			"http://localhost:5173":             true,
			"http://127.0.0.1:5173":            true,
			"http://localhost:5174":             true,
			"http://127.0.0.1:5174":            true,
			"http://localhost:8080":             true,
			"http://127.0.0.1:8080":            true,
			"https://admin.mioru.store":         true,
			"https://www.admin.mioru.store":     true,
		}
		return allowed[origin]
	},
}

type WSHandler struct {
	store  *store.Store
	secret string
	clients map[*websocket.Conn]bool
	mu     sync.RWMutex
}

func NewWSHandler(s *store.Store, secret string) *WSHandler {
	return &WSHandler{
		store:   s,
		secret:  secret,
		clients: make(map[*websocket.Conn]bool),
	}
}

func (h *WSHandler) HandleNotes(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	_, err := auth.ParseToken(token, h.secret)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	// Send existing notes (all for WS broadcast)
	ids, _ := h.store.GetAllNoteIDs(r.Context())
	var notes []model.Note
	for _, id := range ids {
		b, err := h.store.GetRawNote(r.Context(), id)
		if err != nil {
			continue
		}
		var n model.Note
		json.Unmarshal(b, &n)
		notes = append(notes, n)
	}
	if notes == nil {
		notes = []model.Note{}
	}
	b, _ := json.Marshal(map[string]interface{}{"action": "init", "notes": notes})
	conn.WriteMessage(websocket.TextMessage, b)

	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		// Broadcast position updates to other clients
		var data map[string]interface{}
		if json.Unmarshal(msg, &data) == nil {
			if t, ok := data["type"].(string); ok && t == "pos" {
				b, _ := json.Marshal(data)
				h.mu.RLock()
				for c := range h.clients {
					if c != conn {
						c.WriteMessage(websocket.TextMessage, b)
					}
				}
				h.mu.RUnlock()
			}
		}
	}
}

// BroadcastRedis reads from Redis pub/sub and broadcasts to all WS clients
func (h *WSHandler) BroadcastRedis() {
	pubsub := h.store.SubscribeNotes()
	ch := pubsub.Channel()
	for msg := range ch {
		b := []byte(msg.Payload)
		h.mu.RLock()
		for c := range h.clients {
			c.WriteMessage(websocket.TextMessage, b)
		}
		h.mu.RUnlock()
	}
}
