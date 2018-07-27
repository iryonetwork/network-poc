package main

import (
	"crypto/rand"
	"log"
	"net/http"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eos"
)

func main() {
	config, err := config.New()
	if err != nil {
		log.Fatalf("failed to get config: %v", err)
	}
	config.ClientType = "Patient"

	log := logger.New(config)

	eos, err := eos.New(config, log)
	if err != nil {
		log.Fatalf("failed to setup eth storage; %v", err)
	}
	ehr := ehr.New()

	err = eos.SetupSession()
	if err != nil {
		log.Fatalf("Failed to create new account; %v", err)
	}

	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		log.Fatalf("failed to generate random key: %v", err)
	}
	config.EncryptionKeys[config.GetEosPublicKey()] = key
	err = ehr.Encrypt(config.GetEosPublicKey(), []byte{}, key)
	if err != nil {
		log.Fatalf("failed to encrypt data: %v", err)
	}

	h := &handlers{
		config: config,
		eos:    eos,
		ehr:    ehr,
	}

	http.HandleFunc("/", h.indexHandler)
	http.HandleFunc("/grant", h.grantAccessHandler)
	http.HandleFunc("/revoke", h.revokeAccessHandler)
	http.HandleFunc("/save", h.saveEHRHandler)

	log.Printf("starting HTTP server on http://%s", config.ClientAddr)

	if err := http.ListenAndServe(config.ClientAddr, nil); err != nil {
		log.Fatalf("error serving HTTP: %v", err)
	}
}
