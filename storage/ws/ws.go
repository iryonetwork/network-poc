package ws

import (
	"crypto/rsa"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"
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
	addr := "ws://" + config.IryoAddr + "/ws"
	log.Debugf("WS:: Connecting to %s", addr)

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

// crypto/x509 is not supported before go 1.10. These are the functions needed here
func parsePKCS1PublicKey(der []byte) (*rsa.PublicKey, error) {
	var pub pkcs1PublicKey
	rest, err := asn1.Unmarshal(der, &pub)
	if err != nil {
		return nil, err
	}
	if len(rest) > 0 {
		return nil, asn1.SyntaxError{Msg: "trailing data"}
	}

	if pub.N.Sign() <= 0 || pub.E <= 0 {
		return nil, errors.New("x509: public key contains zero or negative value")
	}
	if pub.E > 1<<31-1 {
		return nil, errors.New("x509: public key contains large public exponent")
	}

	return &rsa.PublicKey{
		E: pub.E,
		N: pub.N,
	}, nil
}

type pkcs1PublicKey struct {
	N *big.Int
	E int
}

func marshalPKCS1PublicKey(key *rsa.PublicKey) []byte {
	derBytes, _ := asn1.Marshal(pkcs1PublicKey{
		N: key.N,
		E: key.E,
	})
	return derBytes
}
