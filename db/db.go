package db

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
)

const namesBucket = "names"

type Db struct {
	db     *bolt.DB
	config *config.Config
	log    *logger.Log
}

func Init(config *config.Config, log *logger.Log) (*Db, error) {
	db, err := bolt.Open(getPath(config), 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(namesBucket))
		return err
	})

	return &Db{db: db, config: config, log: log}, err
}

func getPath(config *config.Config) string {
	return fmt.Sprintf("%s/db/names.db", config.StoragePath)
}

func (d *Db) GetName(account string) (name string, err error) {
	var out string
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(namesBucket))
		v := b.Get([]byte(account))
		out = string(v)
		return nil
	})

	return out, err
}

func (d *Db) AddName(account, name string) error {
	d.log.Debugf("Adding %s to db", account)
	err := d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(namesBucket))
		err := b.Put([]byte(account), []byte(name))
		return err
	})
	if err != nil {
		return err
	}

	d.log.Debugf("%s added to db", account)
	return nil
}

func (d *Db) Close() {
	d.db.Close()
}
