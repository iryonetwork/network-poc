package state

import (
	"fmt"
	"os"

	"github.com/go-openapi/swag"
	"github.com/iryonetwork/encrypted-bolt"

	"github.com/iryonetwork/network-poc/logger"
)

type (
	persistentStorage struct {
		db  *bolt.DB
		log *logger.Log
	}
)

var bucket []byte = []byte("state")
var dbPermissions os.FileMode = 0666

func (s *persistentStorage) Get(key string, value interface{}) (bool, error) {
	ok := false
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("state bucket does not exist")
		}

		currentData := b.Get([]byte(key))
		if currentData == nil {
			return nil
		}

		err := swag.ReadJSON(currentData, value)
		if err != nil {
			return err
		}

		ok = true
		return nil
	})

	if err != nil {
		return false, err
	}

	return ok, nil
}

func (s *persistentStorage) Set(key string, value interface{}) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("state bucket does not exist")
		}

		data, err := swag.WriteJSON(value)
		if err != nil {
			return err
		}

		return b.Put([]byte(key), data)
	})
}

// Close closes the database
func (s *persistentStorage) Close() error {
	return s.db.Close()
}

func NewPersitentStorage(path string, key []byte, log *logger.Log) (*persistentStorage, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("Encryption key must be 32 bytes long")
	}

	db, err := bolt.Open(key, path, dbPermissions, nil)
	if err != nil {
		return nil, err
	}

	s := &persistentStorage{
		db:  db,
		log: log,
	}

	// add bucket if does not exist
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	})
	if err != nil {
		return nil, err
	}

	return s, nil
}
