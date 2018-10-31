package client

import (
	"fmt"
	"net/http"

	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eos"

	"github.com/iryonetwork/network-poc/logger"

	"github.com/gorilla/websocket"
	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/state"
)

type Ws struct {
	conn         *websocket.Conn
	frontendConn []*websocket.Conn
	config       *config.Config
	state        *state.State
	ehr          *ehr.Storage
	eos          *eos.Storage
	log          *logger.Log
}

// Connect connects client to api
func ConnectWs(config *config.Config, state *state.State, log *logger.Log, ehr *ehr.Storage, eos *eos.Storage) (*Ws, error) {
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
		out := &Ws{conn: c, config: config, state: state, log: log, ehr: ehr, eos: eos}
		if !state.Subscribed {
			out.Subscribe()
		}
		state.Connected = true
		return out, nil
	}

	return nil, fmt.Errorf("Error authorizing: %s", string(msg))
}

func (s *Ws) Close() error {
	s.log.Debugf("WS:: Closing connection")
	return s.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

func (s *Ws) Conn() *websocket.Conn {
	return s.conn
}

func (s *Ws) Reconnect() error {
	temp, err := ConnectWs(s.config, s.state, s.log, s.ehr, s.eos)
	if err != nil {
		return err
	}

	s.conn = temp.conn
	return nil
}

func (c *Client) AddFrontendWS(conn *websocket.Conn) {
	c.ws.frontendConn = append(c.ws.frontendConn, conn)
}

func (c *Client) RemoveFrontendWS(conn *websocket.Conn) error {
	deleted := false
	for i, v := range c.ws.frontendConn {
		if v == conn {
			c.ws.frontendConn = append(c.ws.frontendConn[:i], c.ws.frontendConn[i+1:]...)
			deleted = true
		}
	}
	if !deleted {
		return fmt.Errorf("Ws connection not found")
	}
	c.log.Debugf("Closing client's ws connection")
	return conn.Close()
}
