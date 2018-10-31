package main

import (
	"crypto/rand"
	"crypto/rsa"
	stdlog "log"
	"net/http"

	"github.com/iryonetwork/network-poc/client"
	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/openEHR/personaldata"
	"github.com/iryonetwork/network-poc/state"
	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eos"
)

func main() {
	config, err := config.New()
	if err != nil {
		stdlog.Fatalf("failed to get config: %v", err)
	}

	log := logger.New(config)

	state, err := state.New(config, log)
	if err != nil {
		log.Fatalf("failed to initialize state: %v", err)
	}
	defer state.Close()

	personaldata.New(state)

	eos, err := eos.New(config, state, log)
	if err != nil {
		log.Fatalf("failed to setup eth storage; %v", err)
	}
	ehr := ehr.New()

	if eos.NewKey() != nil {
		log.Fatalf("Failed to create new key; %v", err)
	}

	state.RSAKey, err = rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Fatalf("Failed generating rsa key")
	}

	client := client.New(config, state, eos, ehr, log)

	if err = client.Login(); err != nil {
		log.Fatalf("Failed to login; %v", err)
	}

	state.EosAccount, err = client.CreateAccount(state.GetEosPublicKey())
	if err != nil {
		log.Fatalf("Failed to create account: %v", err)
	}

	err = client.ConnectWs()
	if err != nil {
		log.Fatalf("ws problem: %v", err.Error())
	}
	defer client.CloseWs()

	// Create key
	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		log.Fatalf("failed to generate random key: %v", err)
	}
	state.EncryptionKeys[state.EosAccount] = key

	if err := personaldata.Upload(state, ehr, client); err != nil {
		log.Fatalf("Error uploading personal data: %v", err)
	}

	h := &handlers{
		config: config,
		state:  state,
		ehr:    ehr,
		client: client,
		log:    log,
	}

	http.HandleFunc("/", h.indexHandler)
	http.HandleFunc("/ehr/", h.ehrHandler)
	http.HandleFunc("/save", h.saveEHRHandler)
	http.HandleFunc("/close", h.closeHandler)
	http.HandleFunc("/connect", h.connectHandler)
	http.HandleFunc("/request", h.requestHandler)
	http.HandleFunc("/ignore", h.ignoreHandler)
	http.HandleFunc("/reencrypt", h.reencryptHandler)
	http.HandleFunc("/grant", h.grantAccessHandler)
	http.HandleFunc("/deny", h.denyAccessHandler)
	http.HandleFunc("/revoke", h.revokeAccessHandler)
	http.HandleFunc("/config", h.configHandler)
	http.HandleFunc("/ws", h.wsHandler)
	if config.ClientType == "Doctor" {
		http.HandleFunc("/switchMode", h.switchModeHandler)
	}

	log.Printf("starting HTTP server on http://%s", config.ClientAddr)

	if err := http.ListenAndServe(config.ClientAddr, nil); err != nil {
		log.Fatalf("error serving HTTP: %v", err)
	}
}
