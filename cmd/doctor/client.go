package main

import (
	"context"
	"fmt"
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
	config   *config.Config
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

	c := &rpcClient{
		client:   client,
		metadata: md,
		eth:      eth,
		ehr:      ehr,
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
				config.Connections = append(config.Connections, event.KeySentDetails.From)
				c.Download(event.KeySentDetails.From)
			}
		}
	}()

	return c, nil
}

func (c *rpcClient) Download(owner string) error {
	granted, err := c.eth.AccessGranted(owner, c.config.EthPublic)
	if err != nil {
		return err
	}

	if !granted {
		return fmt.Errorf("You do not have permission to download this file")
	}

	response, err := c.client.Download(metadata.NewOutgoingContext(context.Background(), c.metadata), &specs.DownloadRequest{
		Owner: owner,
	})
	if err != nil {
		return err
	}

	c.ehr.Save(owner, response.Data)

	return nil
}

func (c *rpcClient) Upload(owner string) error {
	granted, err := c.eth.AccessGranted(owner, c.config.EthPublic)
	if err != nil {
		return err
	}

	if !granted {
		return fmt.Errorf("You do not have permission to upload this file")
	}

	data := c.ehr.Get(owner)
	if data == nil {
		return fmt.Errorf("Document for %s does not exist", owner)
	}

	_, err = c.client.Upload(metadata.NewOutgoingContext(context.Background(), c.metadata), &specs.UploadRequest{
		Owner: owner,
		Data:  data,
	})

	return err
}
