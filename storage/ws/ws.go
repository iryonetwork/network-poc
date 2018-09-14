package ws

import (
	"fmt"
	"net/http"

	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eos"
	"github.com/iryonetwork/network-poc/storage/hub"

	"github.com/iryonetwork/network-poc/logger"

	"github.com/gorilla/websocket"
	"github.com/iryonetwork/network-poc/config"
)

type Storage struct {
	conn   *websocket.Conn
	config *config.Config
	ehr    *ehr.Storage
	eos    *eos.Storage
	hub    *hub.Hub
	log    *logger.Log
}

// NewStorage is APIside storage
func NewStorage(conn *websocket.Conn, config *config.Config, log *logger.Log, hub *hub.Hub, eos *eos.Storage) *Storage {
	return &Storage{conn: conn, config: config, log: log, hub: hub, eos: eos}
}

// Connect connects client to api
func Connect(config *config.Config, log *logger.Log, ehr *ehr.Storage, eos *eos.Storage) (*Storage, error) {
	addr := fmt.Sprintf("ws%s/ws?token=%s", config.IryoAddr[4:], config.Token)
	log.Debugf("WS:: Connecting to ws")

	// Call API's WS
	c, _, err := websocket.DefaultDialer.Dial(addr, http.Header{"Cookie": []string{fmt.Sprintf("token=%s", config.Token)}})
	if err != nil {
		return nil, err
	}

	// Check if authorized
	_, msg, err := c.ReadMessage()
	if err != nil {
		return nil, err
	}
	if string(msg) == "Authorized" {
		out := &Storage{conn: c, config: config, log: log, ehr: ehr, eos: eos}
		if !config.Subscribed {
			out.Subscribe()
		}
		config.Connceted = true
		return out, nil
	}

	return nil, fmt.Errorf("Error authorizing: %s", string(msg))
}

func (s *Storage) Close() error {
	s.log.Debugf("WS:: Closing connection")
	return s.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

func (s *Storage) Reconnect() error {
	temp, err := Connect(s.config, s.log, s.ehr, s.eos)
	if err != nil {
		return err
	}

	*s = *temp
	return nil
}

func (s *Storage) Subscribe() {
	s.log.Debugf("Client::subscribe() called")

	//subscribe to key sent event
	switch s.config.ClientType {
	default:
		s.log.Fatalf("Unknown client type")
	case "Patient":
		s.SubscribePatient()
	case "Doctor":
		s.SubscribeDoctor()
	}
}
