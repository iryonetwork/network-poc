package main

import (
	"fmt"
	"time"
)

func (s *storage) tokenValidateGetName(token string) (string, int, error) {
	if id, valid := s.token.ValidateGetInfo(token); valid {
		return id, 200, nil
	}
	s.log.Printf("Token %s is not valid", token)
	return "", 401, fmt.Errorf("Unknown token")
}

func (s *storage) newToken(id string, exists bool) (string, time.Time, int, error) {
	token, validUntil, err := s.token.NewToken(id, exists)
	if err != nil {
		s.log.Debugf("Error generating token: %+v", err)
		return "", time.Time{}, 500, fmt.Errorf("Error generating token")
	}
	s.log.Debugf("Token %s created", token)
	return token, validUntil, 200, nil
}

func (s *storage) tokenAccountExists(token string) (string, bool, int, error) {
	id, code, err := s.tokenValidateGetName(token)
	if s.token.IsAccount(token) {
		return id, true, code, err
	}
	return id, false, code, err
}

func (s *storage) tokenAccountCreation(token string) (string, int, error) {
	id, exists, code, err := s.tokenAccountExists(token)
	if err != nil {
		return "", code, err
	}
	if exists {
		return "", 403, fmt.Errorf("There is already an account tied to provided token")
	}
	return id, 200, nil
}
