package main

import (
	"crypto/rand"
	"log"
	"net/http"

	"github.com/iryonetwork/network-poc/client"
	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eos"
)

func main() {
	// config
	config, err := config.New()
	if err != nil {
		log.Fatalf("failed to get config: %v", err)
	}
	config.ClientType = "Patient"
	// log
	log := logger.New(config)
	// eos
	eos, err := eos.New(config, log)
	if err != nil {
		log.Fatalf("failed to setup eth storage; %v", err)
	}
	// ehr
	ehr := ehr.New()

	// Client
	client, err := client.New(config, eos, ehr, log)
	if err != nil {
		log.Fatalf("Failed to get client: %v", err)
	}
	if err := client.Login(); err != nil {
		log.Fatalf("Failed to log in: %v", err)
	}

	// Create account
	_, err = eos.ImportKey(config.EosPrivate)
	log.Debugf("Imported key: %v", config.GetEosPublicKey())
	if err != nil {
		log.Fatalf("Failed to import key: %v", err)
	}

	acc, err := client.CreateAccount(config.GetEosPublicKey())
	if err != nil {
		log.Fatalf("Failed to create account: %v", err)
	}
	config.EosAccount = acc

	// Create key
	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		log.Fatalf("failed to generate random key: %v", err)
	}
	config.EncryptionKeys[config.EosAccount] = key

	err = client.ConnectWs()
	if err != nil {
		log.Fatalf("ws problem: %v", err.Error())
	}
	defer client.CloseWs()

	h := &handlers{
		config: config,
		client: client,
		ehr:    ehr,
	}

	http.HandleFunc("/", h.indexHandler)
	http.HandleFunc("/grant", h.grantAccessHandler)
	http.HandleFunc("/deny", h.denyAccessHandler)
	http.HandleFunc("/revoke", h.revokeAccessHandler)
	http.HandleFunc("/save", h.saveEHRHandler)
	http.HandleFunc("/reencrypt", h.reencryptHandler)

	log.Printf("starting HTTP server on http://%s", config.ClientAddr)

	if err := http.ListenAndServe(config.ClientAddr, nil); err != nil {
		log.Fatalf("error serving HTTP: %v", err)
	}
}
