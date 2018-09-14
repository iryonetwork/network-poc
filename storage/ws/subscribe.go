package ws

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/iryonetwork/network-poc/logger"
)

func (s *Storage) SubscribeDoctor() {
	s.log.Debugf("WS::SubscribeDoctor called")

	s.config.Subscribed = true
	defer func() {
		s.config.Subscribed = false
	}()
	go func() {
		for {
			message, err := s.readMessage()
			if websocket.IsCloseError(err, 1000) {
				s.log.Printf("SUBSCRIBE:: Connection closed")
				break
			}
			if err != nil {
				s.log.Printf("SUBSCRIBE:: readMessageError: %v", err)
				break
			}
			// Decode the message
			r, err := decode(message)
			if err != nil {
				s.log.Printf("Error decoding: %v", err)
			}

			// Handle the request
			switch r.Name {
			default:
				s.log.Debugf("SUBSCRIBTION:: Got unknown request %v", r.Name)

			// Import key
			// Get data from request, decrypt the key with RSA key used when request was sent
			// add the key to storage
			case "ImportKey":
				keyenc, err := base64.StdEncoding.DecodeString(subscribeGetStringDataFromRequest(r, "key", s.log))
				if err != nil {
					s.log.Debugf("Error decoding key from base64; %v", err)
				}
				from := subscribeGetStringDataFromRequest(r, "from", s.log)
				name := subscribeGetStringDataFromRequest(r, "name", s.log)

				key, err := rsa.DecryptPKCS1v15(nil, s.config.RSAKey, keyenc)
				if err != nil {
					s.log.Printf("Error decrypting key: %v", err)
					break
				}

				s.log.Debugf("SUBSCRIBTION:: Improting key from user %s", from)

				s.config.Directory[from] = name
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

			// Revoke key
			// Remove all entries connected to user
			case "RevokeKey":
				from := subscribeGetStringDataFromRequest(r, "from", s.log)
				s.log.Debugf("SUBSCRIBTION:: Revoking %s's key", from)
				s.ehr.RemoveUser(from)
				delete(s.config.EncryptionKeys, from)
				for i, v := range s.config.Connections {
					if v == from {
						s.config.Connections = append(s.config.Connections[:i], s.config.Connections[i+1:]...)
						s.log.Debugf("SUBSCRIBTION:: Revoked %s's key ", from)
					}
				}

			// Data was reencrypted
			// make a new key request and delete old data
			case "Reencrypt":
				from := subscribeGetStringDataFromRequest(r, "from", s.log)
				s.ehr.RemoveUser(from)
				err = s.RequestsKey(from)
				if err != nil {
					s.log.Printf("Error creating RequestKey: %v", err)
				}

			// User has granted access to doctor
			// Make a notification that acess has been granted
			case "NotifyGranted":
				name := subscribeGetStringDataFromRequest(r, "name", s.log)
				from := subscribeGetStringDataFromRequest(r, "from", s.log)

				s.log.Debugf("Got notification 'accessGranted' from %s", from)
				s.config.Directory[from] = name
				// Check if we already have the user on the list
				onlist := false
				for _, v := range s.config.Connections {
					if v == from {
						onlist = true
						break
					}
				}
				// if its not on list add it
				if !onlist {
					s.config.GrantedWithoutKeys = append(s.config.GrantedWithoutKeys, from)
				}
			}
		}
	}()
}

func (s *Storage) SubscribePatient() {
	s.log.Debugf("WS::SubscribePatient called")

	s.config.Subscribed = true
	defer func() {
		s.config.Subscribed = false
	}()
	go func() {
		for {
			message, err := s.readMessage()
			if websocket.IsCloseError(err, 1000) {
				s.log.Printf("SUBSCRIBE:: Connection closed")
				break
			}
			if err != nil {
				s.log.Printf("SUBSCRIBE:: readMessageError: %v", err)
				break
			}

			// Decode message into request
			r, err := decode(message)
			if err != nil {
				s.log.Printf("Error decoding: %v", err)
			}

			// Handle the message
			switch r.Name {
			default:
				s.log.Debugf("SUBSCRIBTION:: Got unknown request %v", r.Name)
			case "RequestKey":
				s.log.Debugf("SUBSCRIBTION:: Got RequestKey request")
				from := subscribeGetStringDataFromRequest(r, "from", s.log)
				name := subscribeGetStringDataFromRequest(r, "name", s.log)
				rsakey := subscribeGetDataFromRequest(r, "key", s.log)
				eoskey := subscribeGetStringDataFromRequest(r, "eoskey", s.log)
				sign := subscribeGetStringDataFromRequest(r, "signature", s.log)

				// Check if account and key are connected
				valid, err := s.eos.CheckAccountKey(from, eoskey)
				if err != nil {
					s.log.Printf("Error checking valid account: %v", err)
					break
				}
				if !valid {
					s.log.Debugf("SUBSCRIBE:: Account is not linked to eos account ")
					break
				}

				// check if signature is correct
				valid, err = checkRequestKeySignature(eoskey, sign, rsakey)
				if err != nil {
					s.log.Printf("Error checking valid signature: %v", err)
					break
				}
				if !valid {
					s.log.Debugf("SUBSCRIBE:: signature not valid")
					break
				}

				// Save the request to storage for later usage
				s.config.Requested[from], err = rsaPEMKeyT(rsakey)
				if err != nil {
					s.log.Printf("SUBSCRIBE:: Error getting rsa public key; %v", err)
				}
				s.log.Debugf("%s", s.config.Requested[from])
				s.config.Directory[from] = name

				// Check if access is already granted
				// if it is, send the key without prompting the user for confirmation
				granted, err := s.eos.AccessGranted(s.config.EosAccount, from)
				if err != nil {
					s.log.Printf("Error getting key: %v", err)
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
					// Delete the user from requests
					delete(s.config.Requested, from)
				}
			}
		}
	}()
}

func (s *Storage) readMessage() ([]byte, error) {
	// Read the message
	_, message, err := s.conn.ReadMessage()
	if err != nil {
		if !websocket.IsUnexpectedCloseError(err, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015) {
			s.config.Connceted = false
			s.log.Printf("SUBSCRIBTION:: Closing due to closed connection")
			s.log.Printf("SUBSCRIBTION:: Trying to reastablish connection")
			if err2 := retry(2*time.Second, 5, s.Reconnect); err2 != nil {
				return nil, err2
			}
		}
		return nil, err
	}
	return message, nil
}

func subscribeGetStringDataFromRequest(r *request, key string, log *logger.Log) string {
	out, err := r.getDataString(key)
	if err != nil {
		log.Printf("Error getting `%s`: %v", key, err)
	}
	return out
}

func subscribeGetDataFromRequest(r *request, key string, log *logger.Log) []byte {
	out, err := r.getData(key)
	if err != nil {
		log.Printf("Error getting `%s`: %v", key, err)
	}
	return out
}

func rsaPEMKeyT(pubPEMData []byte) (*rsa.PublicKey, error) {

	block, _ := pem.Decode(pubPEMData)
	if block == nil || block.Type != "RSA PUBLIC KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing public key")
	}

	return x509.ParsePKCS1PublicKey(block.Bytes)
}

func retry(wait time.Duration, attempts int, f func() error) (err error) {
	for i := 0; i < attempts; i++ {
		if err = f(); err == nil {
			log.Printf("Function called successfuly")
			return nil
		}

		time.Sleep(wait)

		log.Println("retrying after error:", err)
	}

	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
