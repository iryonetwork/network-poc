package main

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func (h *handlers) wsHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	// Upgrade connection
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Debugf("Error upgrading request: %v", err)
		return
	}

	// add connection to list
	h.client.AddFrontendWS(c)
	defer func() {
		err = h.client.RemoveFrontendWS(c)
		if err != nil {
			h.log.Printf("Error closing ws: %v", err)
		}
	}()

	for {
		if _, _, err := c.ReadMessage(); err != nil {

			break
		}
	}
}
