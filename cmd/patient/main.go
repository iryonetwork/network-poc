package main

import (
	"crypto/rand"
	"log"
	"net/http"

	"google.golang.org/grpc"

	"github.com/iryonetwork/network-poc/client"
	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/specs"
	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eth"
)

func main() {
	config, err := config.New()
	if err != nil {
		log.Fatalf("failed to get config: %v", err)
	}
	config.ClientType = "Patient"

	eth := eth.New(config)
	ehr := ehr.New()

	conn, err := grpc.Dial(config.IryoAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	client, err := client.New(config, specs.NewCloudClient(conn), eth, ehr)
	if err != nil {
		log.Fatalf("failed to initialize client: %v", err)
	}

	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		log.Fatalf("failed to generate random key: %v", err)
	}
	config.EncryptionKeys[config.GetEthPublicAddress()] = key
	ehr.Encrypt(config.GetEthPublicAddress(), []byte{}, key)

	h := &handlers{
		config: config,
		client: client,
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
