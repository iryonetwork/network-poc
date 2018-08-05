package main

import (
	stdlog "log"
	"net/http"

	"github.com/iryonetwork/network-poc/storage/ws"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/storage/eos"
)

func main() {
	config, err := config.New()

	if err != nil {
		stdlog.Fatalf("failed to get config: %v", err)
	}
	config.ClientType = "Iryo"

	log := logger.New(config)
	eos, err := eos.New(config, log)
	if err != nil {
		panic(err)
	}
	hub := ws.NewHub(log)
	go hub.Run()
	eos.ImportKey(config.EosPrivate)
	h := &handlers{
		config: config,
		eos:    eos,
		log:    log,
		hub:    hub,
	}

	http.HandleFunc("/upload", h.UploadHandler)
	http.HandleFunc("/ls", h.lsHandler)
	http.HandleFunc("/download", h.downloadHandler)
	http.HandleFunc("/createaccount", h.createaccHandler)
	http.HandleFunc("/ws", h.wsHandler)

	log.Printf("starting HTTP server on http://%s", config.IryoAddr)

	if err := http.ListenAndServe(config.IryoAddr, nil); err != nil {
		log.Fatalf("error serving HTTP: %v", err)
	}
}
