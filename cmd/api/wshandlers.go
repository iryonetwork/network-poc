package main

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/iryonetwork/network-poc/requests"
)

type wsStruct struct {
	*handlers
}

var upgrader = websocket.Upgrader{
	CheckOrigin: checkOrigin,
}

func checkOrigin(r *http.Request) bool {
	// TODO: check origin??
	return true
}

func (h *handlers) wsHandler(w http.ResponseWriter, r *http.Request) {
	ws := wsStruct{h}
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
	user, exists := h.token.ValidateGetInfo(token)
	if !exists {
		h.log.Debugf("Invalid token")
		c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Unatuhorized"))
		return
	}
	h.log.Debugf("Token ok")
	c.WriteMessage(websocket.BinaryMessage, []byte("Authorized"))

	// Add user to hub
	h.hub.Register(c, user)
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
		err = ws.HandleRequest(message, user, h.db)
		if err != nil {
			h.log.Debugf("Error HandlingRequest: %v", err)
		}
	}
}

// notify all users that are online and connected to `owner` that new file has been uploaded
func (s *storage) notifyConnectedUpload(owner, uploader string) {
	// generate message
	notification := s.notifyUploadRequest(owner)

	// list users to notify
	connected, err := s.eos.ListConnected(owner)
	if err != nil {
		s.log.Printf("Error getting list of connections, %v", err)
	}
	connected = append(connected, owner)

	// notify all connected users and skip the creator of this request
	for _, v := range connected {
		if v == uploader {
			continue
		}

		// if user is connected send notification
		if s.hub.Connected(v) {
			c, err := s.hub.GetConn(v)
			if err != nil {
				s.log.Printf("Error getting ws.conn; %v", err)
			}
			c.WriteMessage(websocket.BinaryMessage, notification)
		}
	}
}

func (s *storage) notifyUploadRequest(owner string) []byte {
	req := requests.NewReq("NewUpload")
	req.Append("user", owner)

	out, err := req.Encode()
	if err != nil {
		s.log.Printf("Error encoding request NewUpload; %v", err)
	}

	return out
}
