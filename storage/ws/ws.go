package ws

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/asn1"
	"errors"
	"math/big"

	"github.com/iryonetwork/network-poc/storage/ehr"

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
	ehr    *ehr.Storage
}

// NewStorage is APIside storage
func NewStorage(conn *websocket.Conn, config *config.Config, log *logger.Log, hub *Hub) *Storage {
	return &Storage{conn: conn, config: config, log: log, hub: hub}
}

// Connect connects client to api
func Connect(config *config.Config, log *logger.Log, ehr *ehr.Storage) (*Storage, error) {
	addr := "ws://" + config.IryoAddr + "/ws/"
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
	return &Storage{conn: c, config: config, log: log, ehr: ehr}, nil
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
