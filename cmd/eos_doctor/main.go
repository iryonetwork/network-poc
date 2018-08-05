package main

import (
	stdlog "log"
	"net/http"

	"github.com/iryonetwork/network-poc/storage/ws"

	"github.com/iryonetwork/network-poc/config"
	client "github.com/iryonetwork/network-poc/eosclient"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eos"
)

func main() {
	config, err := config.New()
	if err != nil {
		stdlog.Fatalf("failed to get config: %v", err)
	}
	config.ClientType = "Doctor"

	log := logger.New(config)

	eos, err := eos.New(config, log)
	if err != nil {
		log.Fatalf("failed to setup eth storage; %v", err)
	}
	ehr := ehr.New()

	client, err := client.New(config, eos, ehr, log)
	if err != nil {
		log.Fatalf("Failed to setup client; %v", err)
	}
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

	ws, err := ws.Connect(config, log)
	if err != nil {
		log.Fatalf("Error connecting to websocket: %v", err)
	}
	defer ws.Close()

	client.AddWs(ws)
	client.Subscribe()

	h := &handlers{
		config: config,
		ehr:    ehr,
		client: client,
	}

	http.HandleFunc("/ehr/", h.ehrHandler)
	// http.HandleFunc("/save", h.saveEHRHandler)
	http.HandleFunc("/", h.indexHandler)

	log.Printf("starting HTTP server on http://%s", config.ClientAddr)

	if err := http.ListenAndServe(config.ClientAddr, nil); err != nil {
		log.Fatalf("error serving HTTP: %v", err)
	}
}
