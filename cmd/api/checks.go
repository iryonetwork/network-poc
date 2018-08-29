package main

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"

	"github.com/eoscanada/eos-go/ecc"
)

func (s *storage) checkSignature(keystr, signature string, data []byte) (int, error) {
	// Check if signature is correct
	key, err := ecc.NewPublicKey(keystr)
	if err != nil {
		s.log.Printf("Error creating key: %+v", err)
		return 500, fmt.Errorf("Key provided could not be reconstructed")
	}

	// reconstruct signature
	sign, err := ecc.NewSignature(signature)
	if err != nil {
		s.log.Printf("Error creating signature")
		return 500, fmt.Errorf("Error creating signature")
	}

	hash := getHash(data)

	// verify signature
	if !sign.Verify(hash, key) {
		s.log.Printf("Error verifying signature")
		return 403, fmt.Errorf("Signature could not be verified")
	}
	s.log.Debugf("Signature verified")
	return 200, nil
}

func getHash(in []byte) []byte {
	sha := sha256.New()
	sha.Write(in)
	return sha.Sum(nil)
}

// Verify that accounts have all the permissions needed to work with eachother
func (s *storage) checkEOSAccountConnections(account, owner, key string) (int, error) {
	if code, err := s.checkKeyAndID(account, key); err != nil {
		return code, err
	}

	return s.checkAccessGranted(owner, account)
}

// Verifies that account and key are connected
func (s *storage) checkKeyAndID(id, key string) (int, error) {
	if code, err := s.checkAccountExists(id); err != nil {
		return code, err
	}
	if ok, err := s.eos.CheckAccountKey(id, key); !ok {
		if err != nil {
			s.log.Printf("Provided key could not be connected to provided account; %+v", err)
			return 500, fmt.Errorf("Server side error: Provided key could not be connected to provided account")
		}
		s.log.Printf("Key provided is not assigned to provided account")
		return 401, fmt.Errorf("Key provided is not assigned to provided account")
	}

	return 200, nil
}

func (s *storage) checkAccessGranted(owner, account string) (int, error) {
	if code, err := s.checkAccountExists(owner); err != nil {
		return code, err
	}
	if code, err := s.checkAccountExists(account); err != nil {
		return code, err
	}

	if ok, err := s.eos.AccessGranted(owner, account); err != nil {
		s.log.Printf("Error checking connection; %+v", err)
		return 500, fmt.Errorf("Internal server error")
	} else if !ok {
		return 403, fmt.Errorf("You don't have patient's permission to access the data")
	}
	return 200, nil
}

func (s *storage) checkAccountExists(id string) (int, error) {
	if s.eos.CheckAccountExists(id) {
		return 200, nil
	}
	return 404, fmt.Errorf("Account %s not found on the blockchain", id)
}

func isMultipart(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data")
}

func isReupload(r *http.Request) bool {
	reupload := false
	if v, ok := r.Form["reupload"]; ok {
		if v[0] == "true" {
			reupload = true
		}
	}
	return reupload
}
