package main

import (
	"log"

	"google.golang.org/grpc"

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

	eth := eth.New(config)
	ehr := ehr.New()

	conn, err := grpc.Dial(config.IryoAddr)
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	_, err = newClient(config, specs.NewCloudClient(conn), eth, ehr)
	if err != nil {
		log.Fatalf("failed to initialize client: %v", err)
	}
}
