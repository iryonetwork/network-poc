package main

import (
	"log"
	"net"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/specs"
	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eth"
	"google.golang.org/grpc"
)

func main() {
	config, err := config.New()
	if err != nil {
		log.Fatalf("failed to get config: %v", err)
	}

	eth := eth.New(config)
	ehr := ehr.New()

	server := &rpcServer{
		config:  config,
		keySent: make(map[string]chan specs.Event_KeySentDetails),
		eth:     eth,
		ehr:     ehr,
	}

	lis, err := net.Listen("tcp", config.IryoAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Printf("starting gRPC server on %s", config.IryoAddr)

	// Creates a new gRPC server
	s := grpc.NewServer()
	specs.RegisterCloudServer(s, server)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("error serving gRPC: %v", err)
	}
}
