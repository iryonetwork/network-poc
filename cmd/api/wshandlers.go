package main

import (
	"net/http"

	"github.com/iryonetwork/network-poc/storage/ws"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: origin,
}

func origin(r *http.Request) bool { return true }
func (h *handlers) wsHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	// Upgrade connection
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Debugf("Error upgrading request: %v", err)
		return
	}
	defer c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Closing connection"))

	// Authentication
	token := r.Form["token"][0]
	h.log.Debugf("Token: %s", token)
	if token == "" {
		h.log.Debugf("Token field empty")
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
