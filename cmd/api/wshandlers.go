package main

import (
	"net/http"
	"time"

	"github.com/iryonetwork/network-poc/storage/ws"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func (h *handlers) wsHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade connection
	c, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)

	if err != nil {
		h.log.Debugf("Error upgrading request: %v", err)
		return
	}
	defer c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Closing connection"))

	// Authentication
	tokench := make(chan string)
	go func() {
		_, msg, err := c.ReadMessage()
		if err != nil {
			h.log.Debugf("Error recieving inital message: %v", err)
			return
		}
		tokench <- string(msg)
	}()

	var token string

	select {
	case rec := <-tokench:
		token = rec

	case <-time.After(5 * time.Second):
		c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "No token recieved"))
		return
	}

	if err != nil {
		h.log.Debugf("Error getting initial message: %v", err)
		return
	}
	// verify the token
	if token == "" || len(token) != 36 {
		h.log.Debugf("No token sent")
		c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "No token recieved"))
		return
	}
	if !h.token.IsValid(token) {
		h.log.Debugf("Invalid token")
		c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Unatuhorized"))
		return
	}
	h.log.Debugf("Token ok")
	c.WriteMessage(websocket.BinaryMessage, []byte("Authorized"))
	user := h.token.GetID(token)
	// Add user to hub
	h.hub.Register(c, user)
	ws := ws.NewStorage(c, h.config, h.log, h.hub, h.eos)
	defer h.hub.Unregister(c, user)

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
				h.log.Debugf("Error reading message: %v", err)
			} else {
				h.log.Debugf("User %s disconnected", user)
			}
			break
		}
		err = ws.HandleRequest(message, user)
		if err != nil {
			h.log.Debugf("Error HandlingRequest: %v", err)
			break
		}

	}
}
