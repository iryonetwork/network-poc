package ws

import (
	"github.com/iryonetwork/network-poc/logger"

	"github.com/gorilla/websocket"
	"github.com/iryonetwork/network-poc/config"
)

type Storage struct {
	conn   *websocket.Conn
	config *config.Config
	log    *logger.Log
	hub    *Hub
}

// NewStorage is APIside storage
func NewStorage(conn *websocket.Conn, config *config.Config, log *logger.Log, hub *Hub) *Storage {
	return &Storage{conn, config, log, hub}
}

// Connect connects client to api
func Connect(config *config.Config, log *logger.Log) (*Storage, error) {
	addr := "ws://eosapi:8000/ws"
	log.Debugf("WS:: Connecting to %s", addr)

	c, _, err := websocket.DefaultDialer.Dial(addr, nil)
	if err != nil {
		return &Storage{}, err
	}
	// TODO: Figure something better out
	c.WriteMessage(2, []byte(config.EosAccount))
	return &Storage{conn: c, config: config, log: log}, nil
}

func (s *Storage) Close() error {
	s.log.Debugf("WS:: Closing connection")
	err := s.conn.Close()
	return err
}

func (s *Storage) Subscribe() {
	s.log.Debugf("WS::Subscribe called")

	go func() {
		for {
			_, message, err := s.conn.ReadMessage()
			if err != nil {
				s.log.Fatalf("Error while subscribing: %v", err, message)
			}
			r, err := decode(message)
			if err != nil {
				s.log.Fatalf("Error decoding: %v", err)
			}
			// Import key
			switch r.Name {
			default:
				s.log.Debugf("SUBSCRIBTION:: Got unknown request %v", r.Name)
			case "ImportKey":
				req, err := decode(message)
				if err != nil {
					s.log.Fatalf("Error during subscribtion: %v", err)
				}
				key, err := req.getData("key")
				if err != nil {
					s.log.Fatalf("Error getting `key`: %v", err)
				}
				from, err := req.getDataString("from")
				if err != nil {
					s.log.Fatalf("Error getting `from`: %v", err)
				}
				s.log.Debugf("Improting key %s from user %s", key, from)
				s.config.EncryptionKeys[from] = key
				s.config.Connections = append(s.config.Connections, from)
				s.log.Debugf("Improted key %s ", s.config.EncryptionKeys[from])

				// Revoke key
			case "RevokeKey":
				req, err := decode(message)
				if err != nil {
					s.log.Fatalf("Error during subscribtion: %v", err)
				}
				from, err := req.getDataString("from")
				if err != nil {
					s.log.Fatalf("Error getting `from`: %v", err)
				}
				s.log.Debugf("Revoking key from user %s", from)
				delete(s.config.EncryptionKeys, from)
				for i, v := range s.config.Connections {
					if v == from {
						s.config.Connections = append(s.config.Connections[:i], s.config.Connections[i+1:]...)

					}
				}
			}

		}
	}()
}
