package client

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/requests"
	"github.com/iryonetwork/network-poc/state"
	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eos"
)

type (
	Ws struct {
		conn           *websocket.Conn
		frontendConn   []*websocket.Conn
		messageHandler MessageHandler
		config         *config.Config
		state          *state.State
		ehr            *ehr.Storage
		eos            *eos.Storage
		log            *logger.Log
	}

	MessageHandler interface {
		SetClient(client *Client) MessageHandler
		SetWs(ws *Ws) MessageHandler
		SetRequests(requests *requests.Requests) MessageHandler
		SetConnecter(connecter) MessageHandler
		ImportKey(r *requests.Request)
		RevokeKey(r *requests.Request)
		SubReencrypt(r *requests.Request)
		AccessWasGranted(r *requests.Request)
		NotifyKeyRequested(r *requests.Request)
		NewUpload(r *requests.Request)
	}
)

// Connect connects client to api
func ConnectWs(config *config.Config, state *state.State, log *logger.Log, messageHandler MessageHandler, ehr *ehr.Storage, eos *eos.Storage) (*Ws, error) {
	addr := fmt.Sprintf("ws%s/ws?token=%s", config.IryoAddr[4:], state.Token)
	log.Debugf("WS:: Connecting to ws")

	// Call API's WS
	c, _, err := websocket.DefaultDialer.Dial(addr, http.Header{"Cookie": []string{fmt.Sprintf("token=%s", state.Token)}})
	if err != nil {
		return nil, err
	}

	// Check if authorized
	_, msg, err := c.ReadMessage()
	if err != nil {
		return nil, err
	}
	if string(msg) == "Authorized" {
		out := &Ws{conn: c, config: config, state: state, log: log, messageHandler: messageHandler, ehr: ehr, eos: eos}
		messageHandler.SetWs(out)
		if !state.Subscribed {
			out.Subscribe()
		}
		state.Connected = true
		return out, nil
	}

	return nil, fmt.Errorf("Error authorizing: %s", string(msg))
}

func (s *Ws) Subscribe() {
	s.log.Debugf("WS::Subscribe called")

	s.state.Subscribed = true
	defer func() {
		s.state.Subscribed = false
	}()
	go func() {
		for {
			// Decode the message
			r, err := s.readMessage()
			if websocket.IsCloseError(err, 1000) {
				s.log.Printf("SUBSCRIBE:: Connection closed")
				break
			}
			if err != nil {
				s.log.Printf("SUBSCRIBE:: Read message error: %v", err)
				break
			}

			// Handle the request
			switch r.Name {
			case "ImportKey":
				s.messageHandler.ImportKey(r)

			// Revoke key
			// Remove all entries connected to user
			case "RevokeKey":
				s.messageHandler.RevokeKey(r)

			// Data was reencrypted
			// make a new key request and delete old data
			case "Reencrypt":
				s.messageHandler.SubReencrypt(r)

			// User has granted access to doctor
			// Make a notification that access has been granted
			case "NotifyGranted":
				s.messageHandler.AccessWasGranted(r)

			// Key has beed request from another user
			// Notify me
			case "RequestKey":
				s.messageHandler.NotifyKeyRequested(r)

			case "NewUpload":
				s.messageHandler.NewUpload(r)

			default:
				s.log.Debugf("SUBSCRIPTION:: Got unknown request %v", r.Name)
			}
		}
	}()
}

func (s *Ws) readMessage() (*requests.Request, error) {
	// Read the message
	_, message, err := s.conn.ReadMessage()
	if err != nil {
		if !websocket.IsUnexpectedCloseError(err, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015) {
			s.state.Connected = false
			s.log.Printf("WS:: Closing due to closed connection")
			s.log.Printf("WS:: Trying to reastablish connection")
			if err2 := s.retry(2*time.Second, 5, s.Reconnect); err2 != nil {
				return nil, err2
			}
		}
		return nil, err
	}
	return requests.Decode(message)
}

func (s *Ws) Close() error {
	s.log.Debugf("WS:: Closing connection")
	return s.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

func (s *Ws) Conn() *websocket.Conn {
	return s.conn
}

func (s *Ws) Reconnect() error {
	temp, err := ConnectWs(s.config, s.state, s.log, s.messageHandler, s.ehr, s.eos)
	if err != nil {
		return err
	}

	s.conn = temp.conn
	return nil
}

func (s *Ws) retry(wait time.Duration, attempts int, f func() error) (err error) {
	for i := 0; i < attempts; i++ {
		if err = f(); err == nil {
			s.log.Debugf("Function called successfully")
			return nil
		}

		time.Sleep(wait)

		s.log.Println("retrying after error:", err)
	}

	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
