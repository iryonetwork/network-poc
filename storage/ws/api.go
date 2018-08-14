package ws

import (
	"fmt"
	"os"

	"github.com/gorilla/websocket"
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
	case "Reencrypt":
		s.log.Debugf("WS_API:: Got reencrypted notification")
		err := s.reencrypt(&inReq, from)
		return err
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
		s.log.Debugf("User %s not connected, can't send message. Message will be sent when connented", sendTo)
		// user is not connected, add the request to storage
		s.hub.AddRequest(sendTo, req)
		return nil
	}
	return nil
}

func (s *Storage) reencrypt(r *request, from string) error {
	// Backup current data
	os.Rename(fmt.Sprintf("ehr/%s", from), fmt.Sprintf("ehr/%s_backup", from))

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
