package main

import (
	stdlog "log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/iryonetwork/network-poc/storage/token"
	"github.com/iryonetwork/network-poc/storage/ws"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/storage/eos"
	"github.com/rs/cors"
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
		log:    log,
		f: &storage{
			hub:    hub,
			token:  token.Init(log),
			eos:    eos,
			config: config,
			log:    log,
		},
	}
	router := mux.NewRouter()

	router.HandleFunc("/login", h.loginHandler).Methods("POST")
	router.HandleFunc("/ws", h.wsHandler)
	router.HandleFunc("/account", h.createaccHandler).Methods("GET")
	router.HandleFunc("/{account}", h.lsHandler).Methods("GET")
	router.HandleFunc("/{account}/{fid}", h.downloadHandler).Methods("GET")
	router.HandleFunc("/{account}/{fid}", func(w http.ResponseWriter, r *http.Request) {
		h.uploadHandler(w, r, mux.Vars(r)["fid"])
	}).Methods("PUT")
	router.HandleFunc("/{account}", func(w http.ResponseWriter, r *http.Request) {
		h.uploadHandler(w, r, "")
	}).Methods("POST")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "DELETE"},
		AllowedHeaders:   []string{"Authorization"},
		AllowCredentials: true,
	})

	log.Printf("starting HTTP server on http://%s", config.IryoAddr)

	if err := http.ListenAndServe(config.IryoAddr, c.Handler(router)); err != nil {
		log.Fatalf("error serving HTTP: %v", err)
	}
}
