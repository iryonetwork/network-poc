package ws

import (
	"crypto/rand"
	"crypto/rsa"

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
				key, err := rsa.DecryptPKCS1v15(rand.Reader, s.config.RequestKeys[from], keyenc)
				if err != nil {
					s.log.Fatalf("Error decrypting key: %v", err)
				}

				s.log.Debugf("SUBSCRIBTION:: Improting key from user %s", from)
				s.config.EncryptionKeys[from] = key
				exists := false
				for _, name := range s.config.Connections {
					if name == from {
						exists = true
					}
				}
				if !exists {
					s.config.Connections = append(s.config.Connections, from)
				}
				s.log.Debugf("SUBSCRIBTION:: Improted key from %s ", from)

			case "RevokeKey":
				from, err := r.getDataString("from")
				if err != nil {
					s.log.Fatalf("Error getting `from`: %v", err)
				}
				s.log.Debugf("SUBSCRIBTION:: Revoking %s's key", from)
				s.ehr.RemoveUser(from)
				delete(s.config.EncryptionKeys, from)
				for i, v := range s.config.Connections {
					if v == from {
						s.config.Connections = append(s.config.Connections[:i], s.config.Connections[i+1:]...)
						s.log.Debugf("SUBSCRIBTION:: Revoked %s's key ", from)
					}
				}
			case "Reencrypt":
				from, err := r.getDataString("from")
				if err != nil {
					s.log.Fatalf("Error getting `from`: %v", err)
				}
				s.ehr.RemoveUser(from)
				err = s.RequestsKey(from)
				if err != nil {
					s.log.Fatalf("Error creating RequestKey: %v", err)
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
				// Check if access is already granted
				// if it is, send the key without prompting the user for confirmation
				granted, err := s.eos.AccessGranted(s.config.EosAccount, from)
				if err != nil {
					s.log.Fatalf("Error getting key: %v", err)
				}
				if granted {
					s.SendKey(from)
					// make sure they are on the list
					add := false
					for _, name := range s.config.Connections {
						if name == from {
							add = false
						}
					}
					if add {
						s.config.Connections = append(s.config.Connections, from)
					}
					delete(s.config.Requested, from)
				}
			}
		}
	}()
}
