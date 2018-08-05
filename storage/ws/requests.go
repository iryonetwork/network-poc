package ws

import (
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

type Requests interface {
	SendKey(to string)
	RevokeKey(to string)
}

func (s *Storage) SendKey(to string) error {
	s.log.Debugf("WS:: Sending encryption key to %s", to)

	r := newReq("SendKey")
	r.append("to", []byte(to))
	r.append("key", s.config.EncryptionKeys[s.config.EosAccount])
	req, err := r.encode()
	if err != nil {
		return err
	}
	s.conn.WriteMessage(websocket.BinaryMessage, req)
	s.log.Debugf("Sent request: %v", r)
	return nil
}

func (s *Storage) RevokeKey(to string) error {
	s.log.Debugf("WS:: Revoking encryption key at %s", to)

	r := newReq("RevokeKey")
	r.append("to", []byte(to))
	req, err := r.encode()
	if err != nil {
		return err
	}
	s.conn.WriteMessage(websocket.BinaryMessage, req)
	return nil
}

type request struct {
	Name   string
	Fields map[string][]byte
}
type field struct {
	Name string
	Data []byte
}

func newReq(name string) *request {
	return &request{name, make(map[string][]byte)}
}

func (r *request) append(name string, data []byte) {
	r.Fields[name] = data
}

func (r *request) encode() ([]byte, error) {
	return json.Marshal(r)
}

func decode(r []byte) (request, error) {
	req := request{}
	err := json.Unmarshal(r, &req)
	if err != nil {
		return req, err
	}
	return req, nil
}

func (r *request) getData(key string) ([]byte, error) {
	if d, ok := r.Fields[key]; ok {
		return d, nil
	}
	return []byte{}, fmt.Errorf("No data found for key %s", key)
}

func (r *request) getDataString(key string) (string, error) {
	if d, ok := r.Fields[key]; ok {
		return string(d), nil
	}
	return "", fmt.Errorf("No data found for key %s", key)
}
