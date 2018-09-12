package main

import (
	"net/http"

	"github.com/iryonetwork/network-poc/storage/ws"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: checkOrigin,
}

func checkOrigin(r *http.Request) bool {
	// TODO: check origin??
	return true
}
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
	user, exists := h.f.token.ValidateGetInfo(token)
	if !exists {
		h.log.Debugf("Invalid token")
		c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Unatuhorized"))
		return
	}
	h.log.Debugf("Token ok")
	c.WriteMessage(websocket.BinaryMessage, []byte("Authorized"))

	// Add user to hub
	h.f.hub.Register(c, user)
	ws := ws.NewStorage(c, h.config, h.log, h.f.hub, h.f.eos)
	defer h.f.hub.Unregister(c, user)

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
		err = ws.HandleRequest(message, user, h.f.db)
		if err != nil {
			h.log.Debugf("Error HandlingRequest: %v", err)
			break
		}

	}
}
