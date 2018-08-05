package ws

import (
	"fmt"
)

func (s *Storage) HandleRequest(r []byte, from string) error {
	req, err := decode(r)
	if err != nil {
		return err
	}
	s.log.Debugf("Got string request: %v", string(r))
	s.log.Debugf("Got request: %v", req)

	switch req.Name {
	default:
		return fmt.Errorf("Request not valid")
	case "SendKey":
		s.log.Debugf("Sending key")
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
			s.log.Debugf("Sending request %v to %s", r, sendTo)
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
			s.log.Debugf("User %s is not connected, can't send request", sendTo)
			return fmt.Errorf("User %s is not connected, can't send request", sendTo)
		}
	case "RevokeKey":
		s.log.Debugf("Revoking key")
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
			s.log.Debugf("User %s is not connected, can't send request", sendTo)
			return fmt.Errorf("User %s is not connected, can't send request", sendTo)
		}

	}
	return nil
}
