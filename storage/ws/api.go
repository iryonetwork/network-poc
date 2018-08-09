package ws

import (
	"crypto/sha256"
	"fmt"

	"github.com/eoscanada/eos-go/ecc"
	"github.com/gorilla/websocket"

	"github.com/iryonetwork/network-poc/storage/eos"
)

func (s *Storage) HandleRequest(reqdata []byte, from string) error {

	inReq, err := decode(reqdata)
	if err != nil {
		return err
	}
	s.log.Debugf("WS_API:: Got request: %s", inReq.Name)
	var r *request
	switch inReq.Name {
	default:
		return fmt.Errorf("Request not valid")
	case "SendKey":
		s.log.Debugf("WS_API:: Sending key")
		r = newReq("ImportKey")
		key, err := inReq.getData("key")
		if err != nil {
			return err
		}
		r.append("key", key)
		r.append("from", []byte(from))

	case "RevokeKey":
		s.log.Debugf("WS_API:: Revoking key")
		r = newReq("RevokeKey")
		r.append("from", []byte(from))

	case "RequestKey":
		s.log.Debugf("WS_API:: Requesting key")
		r = newReq("RequestKey")
		key, err := inReq.getData("key")
		if err != nil {
			return err
		}
		r.append("from", []byte(from))
		r.append("key", key)
	}

	sendTo, err := inReq.getDataString("to")
	if err != nil {
		return err
	}
	// Encode
	req, err := r.encode()
	if err != nil {
		return err
	}

	// handle sending
	// send if user is connected
	if s.hub.Connected(sendTo) {
		// get connection
		conn, err := s.hub.GetConn(sendTo)
		if err != nil {
			return err
		}
		// send
		conn.WriteMessage(websocket.BinaryMessage, req)
	} else {
		s.log.Debugf("WS_API:: User %s is not connected, can't send request", sendTo)
		// user is not connected, add the request to storage
		s.hub.AddRequest(sendTo, req)
		return nil
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
