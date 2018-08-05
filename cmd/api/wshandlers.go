package main

import (
	"net/http"

	"github.com/iryonetwork/network-poc/storage/ws"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func (h *handlers) wsHandler(w http.ResponseWriter, r *http.Request) {
	h.log.Printf("Got ws request: %v", r.Header)
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Debugf("Error setting up ws: %v", err)
		return
	}
	_, u, err := c.ReadMessage()
	user := string(u)
	h.hub.Register(c, user)
	ws := ws.NewStorage(c, h.config, h.log, h.hub)
	defer func() {
		h.hub.Unregister(c, user)
		c.Close()
	}()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			h.log.Debugf("read: %v", err)
			break
		}
		err = ws.HandleRequest(message, user)
		if err != nil {
			h.log.Debugf("HandleRequest: %v", err)
			break
		}

	}
}
