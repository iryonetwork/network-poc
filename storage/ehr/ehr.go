package ehr

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/segmentio/ksuid"
)

type Storage struct {
	documents map[string]map[string][]byte
}

func New() *Storage {
	return &Storage{documents: make(map[string]map[string][]byte)}
}

func (s *Storage) Saveid(user, id string, document []byte) {
	s.check(user)

	s.documents[user][id] = document
}

func (s *Storage) Save(user string, document []byte) string {
	s.check(user)

	id := newID()
	s.documents[user][id] = document
	return id
}

func (s *Storage) Getid(user, id string) []byte {
	if document, ok := s.documents[user][id]; ok {
		return document
	}
	return nil
}
func (s *Storage) Get(user string) map[string][]byte {
	if document, ok := s.documents[user]; ok {
		return document
	}
	return nil
}

// Exists checks if file exists in docs[user][id]
func (s *Storage) Exists(user, id string) bool {
	_, ok := s.documents[user][id]
	return ok
}

func (s *Storage) Remove(user string) {
	s.documents[user] = make(map[string][]byte)
}

func (s *Storage) Rename(user, id, newid string) {
	s.documents[user][newid] = s.documents[user][id]
	delete(s.documents[user], id)
}

const nonceLength = 12

// Encrypt encrypts and saves document to user's storage using key
// returns ehrID if there is no error. Otherwise returns error
func (s *Storage) Encrypt(user string, document, key []byte) (string, error) {
	s.check(user)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, nonceLength)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	id := newID()
	s.documents[user][id] = append(nonce, aesgcm.Seal(nil, nonce, document, nil)...)

	return id, nil
}

func (s *Storage) Decrypt(owner, id string, key []byte) ([]byte, error) {
	document, ok := s.documents[owner][id]
	if !ok {
		return nil, fmt.Errorf("Document for %s_%s does not exist", owner, id)
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

func newID() string {
	return ksuid.New().String()
}

func (s *Storage) check(user string) {
	if _, ok := s.documents[user]; !ok {
		s.documents[user] = make(map[string][]byte)
	}
}

func (s *Storage) ListIds(user string) []string {
	var ret []string
	for k := range s.documents[user] {
		ret = append(ret, k)
	}
	return ret
}
