package ws

import (
	"crypto/sha256"
	"fmt"

	"github.com/eoscanada/eos-go/ecc"
	"github.com/iryonetwork/network-poc/db"

	"github.com/gorilla/websocket"
)

func (s *Storage) HandleRequest(reqdata []byte, from string, db *db.Db) error {

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
		name, err := db.GetName(from)
		if err != nil {
			return err
		}

		r.append("name", []byte(name))
		r.append("key", key)
		r.append("from", []byte(from))

	case "RevokeKey":
		s.log.Debugf("WS_API:: Revoking key")
		r = newReq("RevokeKey")
		r.append("from", []byte(from))

	case "RequestKey":
		s.log.Debugf("WS_API:: Requesting key")
		err := s.requestKey(inReq, from, db)
		return err

	case "Reencrypt":
		s.log.Debugf("WS_API:: Got reencrypted notification")
		err := s.reencrypt(inReq, from)
		return err

	case "NotifyGranted":
		s.log.Debugf("WS_API:: Got access granted notification from %s", from)
		name, err := db.GetName(from)
		if err != nil {
			return err
		}
		r = newReq("NotifyGranted")
		r.append("from", []byte(from))
		r.append("name", []byte(name))
	}

	sendTo, err := inReq.getDataString("to")
	if err != nil {
		return err
	}
	// Check if reciever exists
	if !s.eos.CheckAccountExists(sendTo) {
		s.log.Debugf("User %s does not exists, trashing the request", sendTo)
		return nil
	}

	return s.sendRequest(r, sendTo)
}

func (s *Storage) sendRequest(r *request, to string) error {
	// Encode
	req, err := r.encode()
	if err != nil {
		return err
	}

	// handle sending
	// send if user is connected
	if s.hub.Connected(to) {
		// get connection
		conn, err := s.hub.GetConn(to)
		if err != nil {
			return err
		}
		// send
		conn.WriteMessage(websocket.BinaryMessage, req)
	} else {
		s.log.Debugf("User %s not connected, can't send message. Message will be sent when connented", to)
		// user is not connected, add the request to storage
		s.hub.AddRequest(to, req)
		return nil
	}
	return nil

}

func (s *Storage) reencrypt(r *request, from string) error {
	// Create list of doctors to send message to
	sendTo, err := s.eos.ListConnected(from)
	if err != nil {
		return err
	}
	// Construct request
	r = newReq("Reencrypt")
	r.append("from", []byte(from))
	req, err := r.encode()
	if err != nil {
		return err
	}

	// Send to all connected users
	for _, to := range sendTo {
		if s.hub.Connected(to) {
			conn, err := s.hub.GetConn(to)
			if err != nil {
				return err
			}
			err = conn.WriteMessage(websocket.BinaryMessage, req)
		} else {
			s.log.Debugf("User %s not connected, can't send message. Message will be sent when connented", to)
			s.hub.AddRequest(to, req)
		}
	}
	return err
}

func (s *Storage) requestKey(r *request, from string, db *db.Db) error {
	// Get the data in request
	eoskey, err := r.getDataString("eoskey")
	if err != nil {
		return err
	}
	rsakey, err := r.getData("key")
	if err != nil {
		return err
	}
	sign, err := r.getDataString("signature")
	if err != nil {
		return err
	}
	// Are key and account connected?
	if ok, err := s.eos.CheckAccountKey(from, eoskey); !ok || err != nil {
		if err != nil {
			return err
		}
		conn, err := s.hub.GetConn(from)
		if err != nil {
			return err
		}
		conn.WriteMessage(websocket.BinaryMessage, []byte("Problem verifying your request"))
	}
	// verify it
	valid, err := checkRequestKeySignature(eoskey, sign, rsakey)
	if err != nil {
		return err
	}

	// send if valid
	if valid {
		s.log.Debugf("Request verified")

		sendto, err := r.getDataString("to")
		if err != nil {
			return err
		}
		name, err := db.GetName(from)
		if err != nil {
			return err
		}

		r.remove("to")
		r.append("from", []byte(from))
		r.append("name", []byte(name))
		s.sendRequest(r, sendto)
	}
	return nil
}

func checkRequestKeySignature(eoskey, strsign string, rsakey []byte) (bool, error) {

	sign, err := ecc.NewSignature(strsign)
	if err != nil {
		return false, err
	}
	key, err := ecc.NewPublicKey(eoskey)
	if err != nil {
		return false, err
	}
	if !sign.Verify(getHash(rsakey), key) {
		return false, nil
	}
	return true, nil
}

func getHash(in []byte) []byte {
	sha := sha256.New()
	sha.Write(in)
	return sha.Sum(nil)
}
