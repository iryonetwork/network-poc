package ehr

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

type Storage struct {
	documents map[string][]byte
}

func New() *Storage {
	return &Storage{documents: make(map[string][]byte)}
}

func (s *Storage) Save(user string, document []byte) {
	s.documents[user] = document
}

func (s *Storage) Get(user string) []byte {
	if document, ok := s.documents[user]; ok {
		return document
	}
	return nil
}

const nonceLength = 12

func (s *Storage) Encrypt(user string, document, key []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	nonce := make([]byte, nonceLength)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	s.documents[user] = append(nonce, aesgcm.Seal(nil, nonce, document, nil)...)

	return nil
}

func (s *Storage) Decrypt(owner string, key []byte) ([]byte, error) {
	document, ok := s.documents[owner]
	if !ok {
		return nil, fmt.Errorf("Document for %s does not exist", owner)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return aesgcm.Open(nil, document[:nonceLength], document[nonceLength:], nil)
}
