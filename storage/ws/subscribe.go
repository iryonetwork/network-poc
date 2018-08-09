package ws

import (
	"crypto/rsa"
	"crypto/sha256"

	"github.com/gorilla/websocket"
)

func (s *Storage) SubscribeDoctor() {
	s.log.Debugf("WS::SubscribeDoctor called")

	go func() {
		for {
			_, message, err := s.conn.ReadMessage()
			if err != nil {
				if !websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
					s.log.Debugf("SUBSCRIBTION:: Closing due to closed connection")
					break
				} else {
					s.log.Fatalf("Error while subscribing: %v", err)
				}
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
				keyenc, err := r.getData("key")
				if err != nil {
					s.log.Fatalf("Error getting `key`: %v", err)
				}
				from, err := r.getDataString("from")
				if err != nil {
					s.log.Fatalf("Error getting `from`: %v", err)
				}
				key, err := rsa.DecryptOAEP(sha256.New(), nil, s.config.RequestKeys[from], keyenc, nil)
				if err != nil {
					s.log.Fatalf("Error decrypting key: %v", err)
				}

				s.log.Debugf("SUBSCRIBTION:: Improting key from user %s", from)
				s.config.EncryptionKeys[from] = key
				s.config.Connections = append(s.config.Connections, from)
				s.log.Debugf("SUBSCRIBTION:: Improted key from %s ", from)

			case "RevokeKey":
				from, err := r.getDataString("from")
				if err != nil {
					s.log.Fatalf("Error getting `from`: %v", err)
				}
				s.log.Debugf("SUBSCRIBTION:: Revoking %s's key", from)
				s.ehr.Remove(from)
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

func (s *Storage) SubscribePatient() {
	s.log.Debugf("WS::SubscribePatient called")

	go func() {
		for {
			_, message, err := s.conn.ReadMessage()
			if err != nil {
				if !websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
					s.log.Debugf("SUBSCRIBTION:: Closing due to closed connection")
					break
				} else {
					s.log.Fatalf("Error while subscribing: %v", err)
				}
			}
			r, err := decode(message)
			if err != nil {
				s.log.Fatalf("Error decoding: %v", err)
			}
			// Import key
			switch r.Name {
			default:
				s.log.Debugf("SUBSCRIBTION:: Got unknown request %v", r.Name)
			case "RequestKey":
				from, err := r.getDataString("from")
				if err != nil {
					s.log.Fatalf("Error getting `from`: %v", err)
				}
				key, err := r.getData("key")
				if err != nil {
					s.log.Fatalf("Error getting `key`: %v", err)
				}
				s.config.Requested[from], err = parsePKCS1PublicKey(key)
				if err != nil {
					s.log.Fatalf("Error getting key: %v", err)
				}
			}
		}
	}()
}
