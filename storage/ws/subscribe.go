package ws

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/iryonetwork/network-poc/logger"
)

type subscribe struct {
	*Storage
}

func (s *Storage) Subscribe() {
	s.log.Debugf("WS::SubscribeDoctor called")
	sub := &subscribe{s}

	s.config.Subscribed = true
	defer func() {
		s.config.Subscribed = false
	}()
	go func() {
		for {
			// Decode the message
			r, err := sub.readMessage()
			if websocket.IsCloseError(err, 1000) {
				s.log.Printf("SUBSCRIBE:: Connection closed")
				break
			}
			if err != nil {
				s.log.Printf("SUBSCRIBE:: Read message error: %v", err)
				break
			}

			// Handle the request
			switch r.Name {
			case "ImportKey":
				sub.ImportKey(r)

			// Revoke key
			// Remove all entries connected to user
			case "RevokeKey":
				sub.revokeKey(r)

			// Data was reencrypted
			// make a new key request and delete old data
			case "Reencrypt":
				sub.subReencrypt(r)

			// User has granted access to doctor
			// Make a notification that access has been granted
			case "NotifyGranted":
				sub.accessWasGranted(r)

			// Key has beed request from another user
			// Notify me
			case "RequestKey":
				sub.notifyKeyRequested(r)

			case "NewUpload":
				sub.newUpload(r)

			default:
				s.log.Debugf("SUBSCRIPTION:: Got unknown request %v", r.Name)
			}
		}
	}()
}

func (s *subscribe) readMessage() (*request, error) {
	// Read the message
	_, message, err := s.conn.ReadMessage()
	if err != nil {
		if !websocket.IsUnexpectedCloseError(err, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015) {
			s.config.Connected = false
			s.log.Printf("SUBSCRIPTION:: Closing due to closed connection")
			s.log.Printf("SUBSCRIPTION:: Trying to reastablish connection")
			if err2 := retry(2*time.Second, 5, s.Reconnect); err2 != nil {
				return nil, err2
			}
		}
		return nil, err
	}
	return decode(message)
}

func (s *subscribe) ImportKey(r *request) {
	keyenc, err := base64.StdEncoding.DecodeString(subscribeGetStringDataFromRequest(r, "key", s.log))
	if err != nil {
		s.log.Debugf("Error decoding key from base64; %v", err)
	}
	from := subscribeGetStringDataFromRequest(r, "from", s.log)
	name := subscribeGetStringDataFromRequest(r, "name", s.log)

	rnd := rand.Reader
	key, err := rsa.DecryptOAEP(sha512.New(), rnd, s.config.RSAKey, keyenc, []byte{})
	if err != nil {
		s.log.Printf("Error decrypting key: %v", err)
		return
	}

	s.log.Debugf("SUBSCRIPTION:: Importing key from user %s", from)

	s.config.Directory[from] = name
	s.config.EncryptionKeys[from] = key

	exists := false
	for _, name := range s.config.Connections.WithKey {
		if name == from {
			exists = true
		}
	}
	if !exists {
		s.config.Connections.WithKey = append(s.config.Connections.WithKey, from)
	}

	s.log.Debugf("SUBSCRIPTION:: Imported key from %s ", from)
}

func (s *subscribe) revokeKey(r *request) {
	from := subscribeGetStringDataFromRequest(r, "from", s.log)

	s.log.Debugf("SUBSCRIPTION:: Revoking %s's key", from)

	s.ehr.RemoveUser(from)
	delete(s.config.EncryptionKeys, from)

	for i, v := range s.config.Connections.WithKey {
		if v == from {
			s.config.Connections.WithKey = append(s.config.Connections.WithKey[:i], s.config.Connections.WithKey[i+1:]...)
			s.log.Debugf("SUBSCRIPTION:: Revoked %s's key ", from)
		}
	}
}

func (s *subscribe) subReencrypt(r *request) {
	from := subscribeGetStringDataFromRequest(r, "from", s.log)
	s.ehr.RemoveUser(from)
	err := s.RequestsKey(from)
	if err != nil {
		s.log.Printf("Error creating RequestKey: %v", err)
	}
}

func (s *subscribe) accessWasGranted(r *request) {
	name := subscribeGetStringDataFromRequest(r, "name", s.log)
	from := subscribeGetStringDataFromRequest(r, "from", s.log)

	s.log.Debugf("Got notification 'accessGranted' from %s", from)
	s.config.Directory[from] = name

	// Check if we already have the user on the list
	onlist := false
	for _, v := range s.config.Connections.WithKey {
		if v == from {
			onlist = true
			return
		}
	}
	// if its not on list add it
	if !onlist {
		s.config.Connections.WithoutKey = append(s.config.Connections.WithoutKey, from)
	}
}

func (s *subscribe) notifyKeyRequested(r *request) {
	s.log.Debugf("SUBSCRIPTION:: Got RequestKey request")
	from := subscribeGetStringDataFromRequest(r, "from", s.log)
	name := subscribeGetStringDataFromRequest(r, "name", s.log)
	rsakey := subscribeGetDataFromRequest(r, "key", s.log)
	sign := subscribeGetStringDataFromRequest(r, "signature", s.log)

	// Check if account and key are connected
	valid, err := s.verifyRequestKeyRequest(sign, from, rsakey)
	if err != nil {
		s.log.Printf("Error checking valid account: %v", err)
		return
	}
	if !valid {
		s.log.Printf("SUBSCRIBE:: request could not be verified")
		return
	}

	// Save the request to storage for later usage
	s.config.Connections.Requested[from], err = rsaPEMKeyToRSAPublicKey(rsakey)
	if err != nil {
		s.log.Printf("SUBSCRIBE:: Error getting rsa public key; %v", err)
	}

	// Add user to directory
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
		for _, name := range s.config.Connections.GrantedTo {
			if name == from {
				add = false
			}
		}
		if add {
			s.config.Connections.GrantedTo = append(s.config.Connections.GrantedTo, from)
		}
		// Delete the user from requests
		delete(s.config.Connections.Requested, from)
	}
}

func (s *subscribe) newUpload(r *request) {
	account := subscribeGetStringDataFromRequest(r, "user", s.log)

	s.log.Debugf("New file for user: %s", account)
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

func rsaPEMKeyToRSAPublicKey(pubPEMData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pubPEMData)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	pubKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Failed to cast pub to rsa.PublicKey (got %T)", pub)
	}

	return pubKey, err
}

func retry(wait time.Duration, attempts int, f func() error) (err error) {
	for i := 0; i < attempts; i++ {
		if err = f(); err == nil {
			log.Printf("Function called successfully")
			return nil
		}

		time.Sleep(wait)

		log.Println("retrying after error:", err)
	}

	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
