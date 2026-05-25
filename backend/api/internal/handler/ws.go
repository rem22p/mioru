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
			"http://localhost:5173":         true,
			"http://127.0.0.1:5173":         true,
			"http://localhost:5174":         true,
			"http://127.0.0.1:5174":         true,
			"http://localhost:8080":         true,
			"http://127.0.0.1:8080":         true,
			"https://admin.mioru.store":     true,
			"https://www.admin.mioru.store": true,
		}
		return allowed[origin]
	},
}

type WSHandler struct {
	store  *store.Store
	secret string
	// clients maps each live connection to its authenticated username so notes
	// are only ever delivered to their owner (notes are private per user).
	clients map[*websocket.Conn]string
	mu      sync.RWMutex
}

func NewWSHandler(s *store.Store, secret string) *WSHandler {
	return &WSHandler{
		store:   s,
		secret:  secret,
		clients: make(map[*websocket.Conn]string),
	}
}

func (h *WSHandler) HandleNotes(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	username, err := auth.ParseToken(token, h.secret, auth.TokenTypeUser)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	h.mu.Lock()
	h.clients[conn] = username
	h.mu.Unlock()

	// Send existing notes — only this user's own notes.
	ids, _ := h.store.GetAllNoteIDs(r.Context())
	var notes []model.Note
	for _, id := range ids {
		b, err := h.store.GetRawNote(r.Context(), id)
		if err != nil {
			continue
		}
		var n model.Note
		json.Unmarshal(b, &n)
		if n.Author != username {
			continue
		}
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
		// Broadcast position updates only to this user's other sessions.
		var data map[string]interface{}
		if json.Unmarshal(msg, &data) == nil {
			if t, ok := data["type"].(string); ok && t == "pos" {
				b, _ := json.Marshal(data)
				h.mu.RLock()
				for c, user := range h.clients {
					if c != conn && user == username {
						c.WriteMessage(websocket.TextMessage, b)
					}
				}
				h.mu.RUnlock()
			}
		}
	}
}

// BroadcastRedis reads note events from Redis pub/sub and delivers each event
// only to the connections owned by the note's author.
func (h *WSHandler) BroadcastRedis() {
	pubsub := h.store.SubscribeNotes()
	ch := pubsub.Channel()
	for msg := range ch {
		b := []byte(msg.Payload)
		var env model.WSMessage
		if json.Unmarshal(b, &env) != nil || env.Note == nil {
			// Malformed or authorless event — drop it rather than leak to all.
			continue
		}
		author := env.Note.Author
		h.mu.RLock()
		for c, user := range h.clients {
			if user == author {
				c.WriteMessage(websocket.TextMessage, b)
			}
		}
		h.mu.RUnlock()
	}
}
