package ws

import (
	"fmt"
	"net/http"

	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eos"

	"github.com/iryonetwork/network-poc/logger"

	"github.com/gorilla/websocket"
	"github.com/iryonetwork/network-poc/config"
)

type Storage struct {
	conn   *websocket.Conn
	config *config.Config
	ehr    *ehr.Storage
	eos    *eos.Storage
	hub    *Hub
	log    *logger.Log
}

// NewStorage is APIside storage
func NewStorage(conn *websocket.Conn, config *config.Config, log *logger.Log, hub *Hub, eos *eos.Storage) *Storage {
	return &Storage{conn: conn, config: config, log: log, hub: hub, eos: eos}
}

// Connect connects client to api
func Connect(config *config.Config, log *logger.Log, ehr *ehr.Storage, eos *eos.Storage, token string) (*Storage, error) {
	addr := fmt.Sprintf("ws%s/ws?token=%s", config.IryoAddr[4:], token)
	log.Debugf("WS:: Connecting to ws")

	// Call API's WS
	c, _, err := websocket.DefaultDialer.Dial(addr, http.Header{"Cookie": []string{fmt.Sprintf("token=%s", token)}})
	if err != nil {
		return nil, err
	}

	// Check if authorized
	_, msg, err := c.ReadMessage()
	if err != nil {
		return nil, err
	}
	if string(msg) == "Authorized" {
		return &Storage{conn: c, config: config, log: log, ehr: ehr, eos: eos}, nil
	}
	return nil, fmt.Errorf("Error authorizing: %s", string(msg))
}

func (s *Storage) Close() error {
	s.log.Debugf("WS:: Closing connection")
	err := s.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	return err
}
