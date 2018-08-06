package ws

import (
	"crypto/sha256"

	"github.com/eoscanada/eos-go/ecc"
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
	addr := "ws://" + config.IryoAddr + "/ws"
	log.Debugf("WS:: Connecting to %s", addr)

	// Call API's WS
	c, response, err := websocket.DefaultDialer.Dial(addr, nil)
	if err != nil {
		return &Storage{}, err
	}
	// Time to authenticate
	token := response.Request.Header["Sec-WebSocket-Key"][0]
	log.Debugf("WS:: Sending authentiction request")
	auth, err := AuthenticateRequest(token, config)
	if err != nil {
		return &Storage{}, err
	}
	c.WriteMessage(2, auth)
	return &Storage{conn: c, config: config, log: log}, nil
}

func AuthenticateRequest(token string, config *config.Config) ([]byte, error) {

	sk, err := ecc.NewPrivateKey(config.EosPrivate)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	h.Write([]byte(token))
	sum := h.Sum(nil)
	sign, err := sk.Sign(sum)
	if err != nil {
		return []byte{}, err
	}

	r := newReq("Authenticate")
	r.append("signature", []byte(sign.String()))
	r.append("user", []byte(config.EosAccount))
	r.append("key", []byte(config.GetEosPublicKey()))
	return r.encode()
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
				s.log.Debugf("SUBSCRIBTION:: SUBSCRIBTION:: Got unknown request %v", r.Name)
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
				s.log.Debugf("SUBSCRIBTION:: Improting key %s from user %s", key, from)
				s.config.EncryptionKeys[from] = key
				s.config.Connections = append(s.config.Connections, from)
				s.log.Debugf("SUBSCRIBTION:: Improted key %s ", s.config.EncryptionKeys[from])

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
				s.log.Debugf("SUBSCRIBTION:: Revoking %s's key", from)
				delete(s.config.EncryptionKeys, from)
				for i, v := range s.config.Connections {
					if v == from {
						s.config.Connections = append(s.config.Connections[:i], s.config.Connections[i+1:]...)
						s.log.Debugf("SUBSCRIBTION:: Revoked %s's key ", from)

					}
				}
			}

		}
	}()
}
