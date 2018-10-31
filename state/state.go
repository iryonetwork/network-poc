package state

import (
	"crypto/rsa"
	"encoding/base64"

	"github.com/eoscanada/eos-go/ecc"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/openEHR"
)

type (
	State struct {
		Token          string
		Connected      bool
		Subscribed     bool
		IsDoctor       bool
		EosPrivate     string `env:"EOS_PRIVATE"`
		EosAccount     string `env:"EOS_ACCOUNT"`
		PersonalData   *openEHR.PersonalData
		EncryptionKeys map[string][]byte
		RSAKey         *rsa.PrivateKey
		Connections    Connections
		Directory      map[string]string
		log            *logger.Log
		persistent     *persistentStorage
	}
)

type Connections struct {
	WithKey    []string           // Access is written on the blockchain and we have the key
	WithoutKey []string           // Access has been written on the blockchain, but we do not have the key
	GrantedTo  []string           // We have granted the access to our data to these users. Its on them to make key request
	Requested  map[string]Request // They are not connected to us, but request for key has been made
}

type Request struct {
	Key        *rsa.PublicKey
	CustomData string
}

const (
	StorageKeyIsDoctor       = "IS_DOCTOR"
	StorageKeyEosPrivate     = "EOS_PRIVATE"
	StorageKeyEosAccount     = "EOS_ACCOUNT"
	StorageKeyPersonalData   = "PERSONAL_DATA"
	StorageKeyEncryptionKeys = "ENCRYPTION_KEYS"
	StorageKeyRSAKey         = "RSA_KEY"
	StorageKeyConnections    = "CONNECTIONS"
	StorageKeyDirectory      = "DIRECTORY"
)

func (s *State) GetEosPublicKey() string {
	key, err := ecc.NewPrivateKey(s.EosPrivate)
	if err != nil {
		panic(err)
	}
	return key.PublicKey().String()
}

func (s *State) GetNames(list []string) map[string]string {
	out := make(map[string]string)
	for _, username := range list {
		if _, ok := s.Directory[username]; !ok {
			out[username] = username
			continue
		}
		out[username] = s.Directory[username]
	}
	return out
}

func (s *State) Close() error {
	if s.persistent != nil {
		err := s.persistent.Set(StorageKeyIsDoctor, s.IsDoctor)
		if err != nil {
			return err
		}

		if s.EosPrivate != "" {
			err = s.persistent.Set(StorageKeyEosPrivate, s.EosPrivate)
			if err != nil {
				return err
			}
		}

		if s.EosAccount != "" {
			err = s.persistent.Set(StorageKeyEosAccount, s.EosAccount)
			if err != nil {
				return err
			}
		}

		if s.PersonalData != nil {
			err = s.persistent.Set(StorageKeyPersonalData, s.PersonalData)
			if err != nil {
				return err
			}
		}

		if s.EncryptionKeys != nil {
			err = s.persistent.Set(StorageKeyEncryptionKeys, s.EncryptionKeys)
			if err != nil {
				return err
			}
		}

		if s.RSAKey != nil {
			err = s.persistent.Set(StorageKeyRSAKey, s.RSAKey)
			if err != nil {
				return err
			}
		}

		err = s.persistent.Set(StorageKeyConnections, s.Connections)
		if err != nil {
			return err
		}

		if s.Directory != nil {
			err = s.persistent.Set(StorageKeyDirectory, s.Directory)
			if err != nil {
				return err
			}
		}

		return s.persistent.Close()
	}

	return nil
}

func New(cfg *config.Config, log *logger.Log) (*State, error) {
	s := State{
		Connected:      false,
		Subscribed:     false,
		PersonalData:   &openEHR.PersonalData{},
		EncryptionKeys: make(map[string][]byte),
		Connections: Connections{
			WithKey:    []string{},
			WithoutKey: []string{},
			GrantedTo:  []string{},
			Requested:  make(map[string]Request),
		},
		Directory: make(map[string]string),
		IsDoctor:  false,
		log:       log,
	}

	if cfg.EosPrivate != "" {
		s.EosPrivate = cfg.EosPrivate
	}
	if cfg.EosAccount != "" {
		s.EosAccount = cfg.EosAccount
	}
	if cfg.ClientType == "Doctor" {
		s.IsDoctor = true
	}

	if cfg.PersistentState {
		err := s.loadPersistentStorage(cfg.PersistentStateEncryptionKey, cfg.PersistentStateStoragePath)
		if err != nil {
			return nil, err
		}
	}

	return &s, nil
}

func (s *State) loadPersistentStorage(base64EncodedKey, storagePath string) error {
	key, err := base64.StdEncoding.DecodeString(base64EncodedKey)
	if err != nil {
		s.log.Fatalf("failed to decode persistent state encryption key")
		return err
	}

	persistent, err := NewPersitentStorage(storagePath, key, s.log)
	if err != nil {
		return err
	}

	s.persistent = persistent

	var isDoctor bool
	ok, err := s.persistent.Get(StorageKeyIsDoctor, &isDoctor)
	if err != nil {
		return err
	}
	if ok {
		s.IsDoctor = isDoctor
	}

	var eosPrivate string
	ok, err = s.persistent.Get(StorageKeyEosPrivate, &eosPrivate)
	if err != nil {
		return err
	}
	if ok {
		s.EosPrivate = eosPrivate
	}

	var eosAccount string
	ok, err = s.persistent.Get(StorageKeyEosAccount, &eosAccount)
	if err != nil {
		return err
	}
	if ok {
		s.EosAccount = eosAccount
	}

	var personalData openEHR.PersonalData
	ok, err = s.persistent.Get(StorageKeyPersonalData, &personalData)
	if err != nil {
		return err
	}
	if ok {
		s.PersonalData = &personalData
	}

	var encryptionKeys map[string][]byte
	ok, err = s.persistent.Get(StorageKeyEncryptionKeys, &encryptionKeys)
	if err != nil {
		return err
	}
	if ok {
		s.EncryptionKeys = encryptionKeys
	}

	var rsaKey rsa.PrivateKey
	ok, err = s.persistent.Get(StorageKeyRSAKey, &rsaKey)
	if err != nil {
		return err
	}
	if ok {
		s.RSAKey = &rsaKey
	}

	var connections Connections
	ok, err = s.persistent.Get(StorageKeyConnections, &connections)
	if err != nil {
		return err
	}
	if ok {
		s.Connections = connections
	}

	var directory map[string]string
	ok, err = s.persistent.Get(StorageKeyDirectory, &directory)
	if err != nil {
		return err
	}
	if ok {
		s.Directory = directory
	}

	return nil
}
