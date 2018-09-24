package main

import (
	"fmt"

	"github.com/iryonetwork/network-poc/requests"

	"github.com/eoscanada/eos-go/ecc"
	"github.com/iryonetwork/network-poc/db"

	"github.com/gorilla/websocket"
)

func (s *wsStruct) HandleRequest(reqdata []byte, from string, db *db.Db) error {

	inReq, err := requests.Decode(reqdata)
	if err != nil {
		return err
	}

	s.log.Debugf("WS_API:: Got request: %s", inReq.Name)
	var r *requests.Request

	switch inReq.Name {
	case "SendKey":
		s.log.Debugf("WS_API:: Sending key")
		r = requests.NewReq("ImportKey")
		key, err := inReq.GetDataString("key")
		if err != nil {
			return err
		}
		name, err := db.GetName(from)
		if err != nil {
			return err
		}
		r.Append("name", (name))
		r.Append("key", key)
		r.Append("from", (from))

	case "RevokeKey":
		s.log.Debugf("WS_API:: Revoking key")
		r = requests.NewReq("RevokeKey")
		r.Append("from", (from))

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
		r = requests.NewReq("NotifyGranted")
		r.Append("from", (from))
		r.Append("name", (name))
	default:
		s.log.Debugf("Recieved an invalid request")
		return nil

	}

	sendTo, err := inReq.GetDataString("to")
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

func (s *wsStruct) sendRequest(r *requests.Request, to string) error {
	// Encode
	req, err := r.Encode()
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

func (s *wsStruct) reencrypt(r *requests.Request, from string) error {
	// Create list of doctors to send message to
	sendTo, err := s.eos.ListConnected(from)
	if err != nil {
		return err
	}
	// Construct request
	r = requests.NewReq("Reencrypt")
	r.Append("from", (from))
	req, err := r.Encode()
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

func (s *wsStruct) requestKey(r *requests.Request, from string, db *db.Db) error {
	// Get the data in request
	rsakey, err := r.GetData("key")
	if err != nil {
		return err
	}
	sign, err := r.GetDataString("signature")
	if err != nil {
		return err
	}

	// verify it
	if valid, err := s.verifyRequestKeyRequest(sign, from, rsakey); !valid || err != nil {
		conn, err2 := s.hub.GetConn(from)
		if err2 != nil {
			return err2
		}

		conn.WriteMessage(websocket.BinaryMessage, []byte("Problem verifying your request"))
		return err
	}

	s.log.Debugf("Request verified")

	sendto, err := r.GetDataString("to")
	if err != nil {
		return err
	}
	name, err := db.GetName(from)
	if err != nil {
		return err
	}

	r.Remove("to")
	r.Append("from", from)
	r.Append("name", name)
	s.sendRequest(r, sendto)

	return nil
}

func (s *wsStruct) verifyRequestKeyRequest(signature, from string, rsakey []byte) (bool, error) {
	eoskey, err := requestGetKeyFromSignature(signature, rsakey)
	if err != nil {
		return false, err
	}

	return s.eos.CheckAccountKey(from, eoskey.String())
}

func requestGetKeyFromSignature(strsign string, rsakey []byte) (ecc.PublicKey, error) {

	sign, err := ecc.NewSignature(strsign)
	if err != nil {
		return ecc.PublicKey{}, err
	}

	key, err := sign.PublicKey(getHash(rsakey))
	if err != nil {
		return ecc.PublicKey{}, fmt.Errorf("Signture could not be verified; %v", err)
	}
	return key, nil
}
