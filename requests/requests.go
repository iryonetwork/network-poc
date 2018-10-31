package requests

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/state"
	"github.com/iryonetwork/network-poc/storage/eos"

	"github.com/gorilla/websocket"
	"github.com/iryonetwork/network-poc/logger"
)

type Requests struct {
	log    *logger.Log
	config *config.Config
	state  *state.State
	conn   *websocket.Conn
	eos    *eos.Storage
}

func NewRequests(log *logger.Log, cfg *config.Config, state *state.State, conn *websocket.Conn, eos *eos.Storage) *Requests {
	return &Requests{log, cfg, state, conn, eos}
}

func (s *Requests) SendKey(to string) error {
	s.log.Debugf("WS:: Sending encryption key to %s", to)

	r := NewReq("SendKey")
	r.Append("to", to)
	accessRequest, ok := s.state.Connections.Requested[to]
	if !ok {
		return fmt.Errorf("No key from user %s found", to)
	}

	// Encrypt key
	rnd := rand.Reader
	encKey, err := rsa.EncryptOAEP(sha512.New(), rnd, accessRequest.Key, s.state.EncryptionKeys[s.state.EosAccount], []byte{})

	if err != nil {
		return err
	}
	r.Append("key", base64.StdEncoding.EncodeToString(encKey))

	// append customData if set
	r.Append("customData", accessRequest.CustomData)

	req, err := r.Encode()
	if err != nil {
		return err
	}

	err = s.conn.WriteMessage(websocket.BinaryMessage, req)
	s.log.Debugf("Sent SendKey request")
	return err
}

func (s *Requests) RevokeKey(to string) error {
	s.log.Debugf("WS:: Revoking encryption key at %s", to)

	r := NewReq("RevokeKey")
	r.Append("to", to)
	req, err := r.Encode()
	if err != nil {
		return err
	}
	err = s.conn.WriteMessage(websocket.BinaryMessage, req)
	return err
}

func (s *Requests) RequestsKey(to, customData string) error {
	s.log.Debugf("WS:: Requesting encryption key from %s", to)

	r := NewReq("RequestKey")
	r.Append("to", to)

	// Generate public key
	key, err := rsaPublicToByte(&s.state.RSAKey.PublicKey)
	if err != nil {
		return err
	}

	r.Append("key", string(key))

	// Sign the key
	sign, err := s.eos.SignHash(key)
	if err != nil {
		return err
	}
	r.Append("signature", sign)

	// Add your EOS key
	r.Append("eoskey", s.state.GetEosPublicKey())

	// Add customData if set
	if customData != "" {
		r.Append("customData", customData)
	}

	req, err := r.Encode()
	err = s.conn.WriteMessage(websocket.BinaryMessage, req)

	// Check if user is on GrantedWithoutKeys list
	for i, v := range s.state.Connections.WithoutKey {
		if v == to {
			s.state.Connections.WithoutKey = append(s.state.Connections.WithoutKey[:i], s.state.Connections.WithoutKey[i+1:]...)
		}
	}

	return err
}

func rsaPublicToByte(public *rsa.PublicKey) ([]byte, error) {
	keyBytes, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		return nil, err
	}

	keyBlock := pem.Block{
		Type:    "PUBLIC KEY",
		Headers: nil,
		Bytes:   keyBytes,
	}
	return pem.EncodeToMemory(&keyBlock), nil
}

func (s *Requests) NotifyGranted(to string) error {
	s.log.Debugf("WS:: Notifying %s that access was granted", to)

	r := NewReq("NotifyGranted")
	r.Append("to", to)
	req, err := r.Encode()
	if err != nil {
		return err
	}
	err = s.conn.WriteMessage(websocket.BinaryMessage, req)
	return err
}

func (s *Requests) ReencryptRequest() error {
	s.log.Debugf("WS: Sending reeencrypted notification")

	r := NewReq("Reencrypt")
	req, err := r.Encode()
	if err != nil {
		return err
	}
	err = s.conn.WriteMessage(websocket.BinaryMessage, req)

	return err
}

type Request struct {
	Name   string
	Fields map[string]string
}

func NewReq(name string) *Request {
	return &Request{name, make(map[string]string)}
}

func (r *Request) Append(name string, data string) {
	r.Fields[name] = data
}

func (r *Request) Encode() ([]byte, error) {
	return json.Marshal(r)
}

func Decode(r []byte) (*Request, error) {
	req := &Request{}
	err := json.Unmarshal(r, req)
	if err != nil {
		return req, fmt.Errorf("Error decoding request: %v\nRequest sent in: %s", err, r)
	}
	return req, nil
}

func (r *Request) GetData(key string) ([]byte, error) {
	if d, ok := r.Fields[key]; ok {
		return []byte(d), nil
	}
	return []byte{}, fmt.Errorf("No data found for key %s", key)
}

func (r *Request) GetDataString(key string) (string, error) {
	if d, ok := r.Fields[key]; ok {
		return d, nil
	}
	return "", fmt.Errorf("No data found for key %s", key)
}
func (r *Request) Remove(key string) {
	if _, ok := r.Fields[key]; ok {
		delete(r.Fields, key)
	}
}
