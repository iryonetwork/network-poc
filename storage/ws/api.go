package ws

import (
	"crypto/sha256"
	"fmt"

	"github.com/eoscanada/eos-go/ecc"

	"github.com/iryonetwork/network-poc/storage/eos"
)

func (s *Storage) HandleRequest(r []byte, from string) error {
	req, err := decode(r)
	if err != nil {
		return err
	}
	s.log.Debugf("WS_API:: Got request: %s", req)

	switch req.Name {
	default:
		return fmt.Errorf("Request not valid")
	case "SendKey":
		s.log.Debugf("WS_API:: Sending key")
		r := newReq("ImportKey")
		key, err := req.getData("key")
		if err != nil {
			return err
		}
		r.append("key", key)
		r.append("from", []byte(from))
		sendTo, err := req.getDataString("to")
		if err != nil {
			return err
		}
		if s.hub.Connected(sendTo) {
			s.log.Debugf("WS_API:: Sending key to %s", sendTo)
			conn, err := s.hub.GetConn(sendTo)
			if err != nil {
				return err
			}

			req, err := r.encode()
			if err != nil {
				return err
			}
			conn.WriteMessage(2, req)
		} else {
			s.log.Debugf("WS_API:: User %s is not connected, can't send request", sendTo)
			return nil
		}
	case "RevokeKey":
		s.log.Debugf("WS_API:: Revoking key")
		r := newReq("RevokeKey")
		r.append("from", []byte(from))
		sendTo, err := req.getDataString("to")
		if err != nil {
			return err
		}
		if s.hub.Connected(sendTo) {
			conn, err := s.hub.GetConn(sendTo)
			if err != nil {
				return err
			}

			req, err := r.encode()
			if err != nil {
				return err
			}
			conn.WriteMessage(2, req)
		} else {
			s.log.Debugf("WS_API:: User %s is not connected, can't send request", sendTo)
			return nil
		}

	}
	return nil
}

// Authenticate takes connection token, authentication message and eos package storage
// Returns true and account name if user used his key to sign and the signature is correct
// Returns false if signature could not be verifyed
func Authenticate(token, msg []byte, eos *eos.Storage) (bool, string, error) {
	req, err := decode(msg)

	if err != nil {
		return false, "", err
	}
	if req.Name != "Authenticate" {
		return false, "", fmt.Errorf("Request is not authenticate")
	}
	user, err := req.getDataString("user")
	if err != nil {
		return false, "", err
	}
	key, err := req.getDataString("key")
	if err != nil {
		return false, "", err
	}

	correctKey, err := eos.CheckAccountKey(user, key)
	if err != nil {
		return false, "", err
	}
	if !correctKey {
		return false, "", fmt.Errorf("Provided key is not connected to provided account")
	}
	// Get signature and check it
	signature, err := req.getDataString("signature")
	if err != nil {
		return false, "", err
	}
	valid, err := checkSignature(token, signature, key)
	return valid, user, nil
}

func checkSignature(data []byte, sig, key string) (bool, error) {
	signature, err := ecc.NewSignature(sig)
	if err != nil {
		return false, err
	}
	pk, err := ecc.NewPublicKey(key)
	if err != nil {
		return false, err
	}
	h := sha256.New()
	h.Write(data)
	return signature.Verify(h.Sum(nil), pk), nil
}
