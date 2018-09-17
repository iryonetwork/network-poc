package ws

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/gorilla/websocket"
)

type Requests interface {
	SendKey(to string)
	RevokeKey(to string)
	RequestKey(to string)
	NotifyGranted(to string)
	ReencryptRequest()
}

func (s *Storage) SendKey(to string) error {
	s.log.Debugf("WS:: Sending encryption key to %s", to)

	r := newReq("SendKey")
	r.append("to", to)
	// Encrypt key
	if _, ok := s.config.Connections.Requested[to]; !ok {
		return fmt.Errorf("No key from user %s found", to)
	}
	encKey, err := rsa.EncryptPKCS1v15(rand.Reader, s.config.Connections.Requested[to], s.config.EncryptionKeys[s.config.EosAccount])
	if err != nil {
		return err
	}
	r.append("key", base64.StdEncoding.EncodeToString(encKey))
	req, err := r.encode()
	if err != nil {
		return err
	}
	err = s.conn.WriteMessage(websocket.BinaryMessage, req)
	s.log.Debugf("Sent SendKey request")
	return err
}

func (s *Storage) RevokeKey(to string) error {
	s.log.Debugf("WS:: Revoking encryption key at %s", to)

	r := newReq("RevokeKey")
	r.append("to", to)
	req, err := r.encode()
	if err != nil {
		return err
	}
	err = s.conn.WriteMessage(websocket.BinaryMessage, req)
	return err
}

func (s *Storage) RequestsKey(to string) error {
	s.log.Debugf("WS:: Requesting encryption key from %s", to)

	r := newReq("RequestKey")
	r.append("to", to)

	// Generate public key
	key := rsaPublicToByte(&s.config.RSAKey.PublicKey)

	r.append("key", string(key))

	// Sign the key
	sign, err := s.eos.SignHash(key)
	if err != nil {
		return err
	}
	r.append("signature", sign)

	// And add your EOS key
	r.append("eoskey", s.config.GetEosPublicKey())

	req, err := r.encode()
	err = s.conn.WriteMessage(websocket.BinaryMessage, req)

	// Check if user is on GrantedWithoutKeys list
	for i, v := range s.config.Connections.WithoutKey {
		if v == to {
			s.config.Connections.WithoutKey = append(s.config.Connections.WithoutKey[:i], s.config.Connections.WithoutKey[i+1:]...)
		}
	}

	return err
}

func rsaPublicToByte(public *rsa.PublicKey) []byte {
	keyBytes := x509.MarshalPKCS1PublicKey(public)
	keyBlock := pem.Block{
		Type:    "RSA PUBLIC KEY",
		Headers: nil,
		Bytes:   keyBytes,
	}
	return pem.EncodeToMemory(&keyBlock)
}

func (s *Storage) NotifyGranted(to string) error {
	s.log.Debugf("WS:: Notifying %s that access was granted", to)

	r := newReq("NotifyGranted")
	r.append("to", to)
	req, err := r.encode()
	if err != nil {
		return err
	}
	err = s.conn.WriteMessage(websocket.BinaryMessage, req)
	return err
}

func (s *Storage) ReencryptRequest() error {
	s.log.Debugf("WS: Sending reeencrypted notification")

	r := newReq("Reencrypt")
	req, err := r.encode()
	if err != nil {
		return err
	}
	err = s.conn.WriteMessage(websocket.BinaryMessage, req)

	return err
}

type request struct {
	Name   string
	Fields map[string]string
}

func newReq(name string) *request {
	return &request{name, make(map[string]string)}
}

func (r *request) append(name string, data string) {
	r.Fields[name] = data
}

func (r *request) encode() ([]byte, error) {
	return json.Marshal(r)
}

func decode(r []byte) (*request, error) {
	req := &request{}
	err := json.Unmarshal(r, req)
	if err != nil {
		return req, fmt.Errorf("Error decoding request: %v\nRequest sent in: %s", err, r)
	}
	return req, nil
}

func (r *request) getData(key string) ([]byte, error) {
	if d, ok := r.Fields[key]; ok {
		return []byte(d), nil
	}
	return []byte{}, fmt.Errorf("No data found for key %s", key)
}

func (r *request) getDataString(key string) (string, error) {
	if d, ok := r.Fields[key]; ok {
		return d, nil
	}
	return "", fmt.Errorf("No data found for key %s", key)
}
func (r *request) remove(key string) {
	if _, ok := r.Fields[key]; ok {
		delete(r.Fields, key)
	}
}
