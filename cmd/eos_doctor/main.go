package main

import (
	stdlog "log"
	"net/http"

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

	// if err := eos.SetupSession(); err != nil {
	// 	log.Fatalf("Failed to setup eth connection; %v", err)
	// }

	client, err := client.New(config, eos, ehr, log)
	if err != nil {
		log.Fatalf("Failed to setup client; %v", err)
	}
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
