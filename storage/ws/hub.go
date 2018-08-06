package ws

import (
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/iryonetwork/network-poc/logger"
)

// Hub maintains the set of active clients
type Hub struct {
	log        *logger.Log
	clients    map[string]*websocket.Conn
	register   chan *client
	unregister chan *client
}
type client struct {
	conn *websocket.Conn
	name string
}

func NewHub(log *logger.Log) *Hub {
	return &Hub{
		log:        log,
		register:   make(chan *client),
		unregister: make(chan *client),
		clients:    make(map[string]*websocket.Conn),
	}
}

func (h *Hub) Run() {
	h.log.Debugf("HUB:: Starting the hub")
	for {
		select {
		case client := <-h.register:
			h.log.Debugf("HUB:: Registering user %s", client.name)
			h.clients[client.name] = client.conn
		case client := <-h.unregister:
			h.log.Debugf("HUB:: Unegistering user %s", client.name)
			if _, ok := h.clients[client.name]; ok {
				delete(h.clients, client.name)
				client.conn.Close()
				h.log.Debugf("HUB:: %s unregistered", client.name)

			}
		}
	}
}

func (h *Hub) Register(c *websocket.Conn, name string) {
	h.register <- &client{c, name}
}

func (h *Hub) Unregister(c *websocket.Conn, name string) {
	h.unregister <- &client{c, name}
}

func (h *Hub) Connected(who string) bool {
	_, ok := h.clients[who]
	return ok
}

func (h *Hub) GetConn(user string) (*websocket.Conn, error) {
	if h.Connected(user) {
		return h.clients[user], nil
	} else {
		return nil, fmt.Errorf("Connection for user %s found", user)
	}
}
