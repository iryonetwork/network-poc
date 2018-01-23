package main

import (
	"context"
	"io"
	"log"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/specs"
	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eth"
	"google.golang.org/grpc/metadata"
)

const tokenKey = "token"

type rpcClient struct {
	client   specs.CloudClient
	metadata metadata.MD
	eth      *eth.Storage
	ehr      *ehr.Storage
}

func newClient(config *config.Config, client specs.CloudClient, eth *eth.Storage, ehr *ehr.Storage) (*rpcClient, error) {
	response, err := client.Login(context.Background(), &specs.LoginRequest{
		Public:    config.EthPublic,
		Signature: []byte("TODO"),
	})
	if err != nil {
		return nil, err
	}

	md := metadata.Pairs(tokenKey, response.Token)

	stream, err := client.Subscribe(metadata.NewOutgoingContext(context.Background(), md), nil)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			event, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("error receiving events: %v", err)
			}

			if event.Type == specs.Event_KeySent {
				config.EncryptionKeys[event.KeySentDetails.From] = event.KeySentDetails.Key
			}
		}
	}()

	return &rpcClient{
		client:   client,
		metadata: md,
		eth:      eth,
		ehr:      ehr,
	}, nil
}
