package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type string `json:"type"` // "join" | "msg" | "leave"
	User string `json:"user"`
	Text string `json:"text"`
	Time string `json:"time"`
}

type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]string // conn -> username
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]string)}
}

func (h *Hub) broadcast(sender *websocket.Conn, m Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for c := range h.clients {
		if err := c.WriteJSON(m); err != nil {
			_ = c.Close()
			delete(h.clients, c)
		}
	}
}

func (h *Hub) add(conn *websocket.Conn, user string) {
	h.mu.Lock()
	h.clients[conn] = user
	h.mu.Unlock()

	h.broadcast(nil, Message{
		Type: "join",
		User: user,
		Text: user + " joined",
		Time: time.Now().Format(time.RFC3339),
	})
}

func (h *Hub) remove(conn *websocket.Conn) {
	h.mu.Lock()
	user, ok := h.clients[conn]
	if ok {
		delete(h.clients, conn)
	}
	h.mu.Unlock()

	if ok {
		h.broadcast(nil, Message{
			Type: "leave",
			User: user,
			Text: user + " left",
			Time: time.Now().Format(time.RFC3339),
		})
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	hub := NewHub()

	// serve frontend
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	// WebSocket endpoint.
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("upgrade:", err)
			return
		}

		// first message should be "join" with a username
		var first Message
		if err := conn.ReadJSON(&first); err != nil || first.User == "" {
			_ = conn.Close()
			return
		}

		user := first.User
		hub.add(conn, user)

		// message loop
		for {
			var m Message
			if err := conn.ReadJSON(&m); err != nil {
				break
			}

			if m.Type == "" {
				m.Type = "msg"
			}
			m.User = user
			m.Time = time.Now().Format(time.RFC3339)

			if m.Type == "msg" && len(m.Text) == 0 {
				continue
			}

			hub.broadcast(conn, m)
		}

		hub.remove(conn)
		_ = conn.Close()
	})

	addr := ":8080"
	log.Println("Chat server running on http://localhost" + addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
