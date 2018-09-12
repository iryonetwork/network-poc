package main

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/iryonetwork/network-poc/config"
)

const namesBucket = "names"

func dbInit(config *config.Config) (*bolt.DB, error) {
	db, err := bolt.Open(dbPath(config), 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(namesBucket))
		return err
	})

	return db, err
}

func (s *storage) dbGetName(account string) (name string, code int, err error) {
	var out string
	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(namesBucket))
		v := b.Get([]byte(account))
		out = string(v)
		return nil
	})

	return out, 200, nil
}

func (s *storage) dbAddName(account, name string) (int, error) {
	s.log.Debugf("Adding %s to db", account)
	s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(namesBucket))
		err := b.Put([]byte(account), []byte(name))
		return err
	})

	s.log.Debugf("%s added db", account)
	return 200, nil
}

func dbPath(config *config.Config) string {
	return fmt.Sprintf("%s/db/names.db", config.StoragePath)
}
