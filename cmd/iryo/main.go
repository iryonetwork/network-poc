package main

import (
	"log"
	"net"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/specs"
	"github.com/iryonetwork/network-poc/storage/eth"
	"google.golang.org/grpc"
)

func main() {
	config, err := config.New()
	if err != nil {
		log.Fatalf("failed to get config: %v", err)
	}

	eth := eth.New(config)

	server := &rpcServer{
		config:  config,
		keySent: make(map[string]chan specs.Event_KeySentDetails),
		eth:     eth,
	}

	lis, err := net.Listen("tcp", config.IryoAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Creates a new gRPC server
	s := grpc.NewServer()
	specs.RegisterCloudServer(s, server)
	s.Serve(lis)
}
