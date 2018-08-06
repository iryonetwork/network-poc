package main

import (
	"net/http"

	"github.com/iryonetwork/network-poc/storage/ws"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func (h *handlers) wsHandler(w http.ResponseWriter, r *http.Request) {
	h.log.Debugf("Got ws request: %v", r.Header)
	// Upgrade connection
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Debugf("Error upgrading request: %v", err)
		return
	}
	defer c.Close()
	_, message, err := c.ReadMessage()
	// Authentication
	token := r.Header.Get("Sec-Websocket-Key")
	auth, user, err := ws.Authenticate([]byte(token), message, h.eos)
	if err != nil {
		h.log.Printf("Error during authentication: %v", err)
		c.Close()
		return
	}
	if !auth {
		h.log.Printf("User %s could not be verified", user)
		c.Close()
		return
	}

	// Add user to hub
	h.hub.Register(c, user)
	ws := ws.NewStorage(c, h.config, h.log, h.hub)
	defer h.hub.Unregister(c, user)

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			h.log.Debugf("Error reading message: %v", err)
			break
		}
		err = ws.HandleRequest(message, user)
		if err != nil {
			h.log.Debugf("Error HandlingRequest: %v", err)
			break
		}

	}
}
