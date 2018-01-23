package client

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

type RPCClient struct {
	client   specs.CloudClient
	config   *config.Config
	metadata metadata.MD
	eth      *eth.Storage
	ehr      *ehr.Storage
}

func New(config *config.Config, client specs.CloudClient, eth *eth.Storage, ehr *ehr.Storage) (*RPCClient, error) {
	response, err := client.Login(context.Background(), &specs.LoginRequest{
		Public:    config.GetEthPublicAddress(),
		Signature: []byte("TODO"),
	})
	if err != nil {
		return nil, err
	}

	md := metadata.Pairs(tokenKey, response.Token)

	return &RPCClient{
		client:   client,
		config:   config,
		metadata: md,
		eth:      eth,
		ehr:      ehr,
	}, nil
}

func (c *RPCClient) Download(owner string) error {
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

func (c *RPCClient) Upload(owner string) error {
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

func (c *RPCClient) GrantAccess(to string) error {
	err := c.eth.GrantAccess(to)
	if err != nil {
		return err
	}

	_, err = c.client.SendKey(metadata.NewOutgoingContext(context.Background(), c.metadata), &specs.SendKeyRequest{
		To:  to,
		Key: c.config.EncryptionKeys[c.config.GetEthPublicAddress()],
	})

	if err == nil {
		c.config.Connections = append(c.config.Connections, to)
	}

	return err
}

func (c *RPCClient) RevokeAccess(to string) error {
	found := false
	for i, v := range c.config.Connections {
		if v == to {
			c.config.Connections = append(c.config.Connections[:i], c.config.Connections[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("%s is not in connections", to)
	}

	return c.eth.RevokeAccess(to)
}

func (c *RPCClient) Subscribe() error {
	stream, err := c.client.Subscribe(metadata.NewOutgoingContext(context.Background(), c.metadata), &specs.Empty{})
	if err != nil {
		return err
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
				c.config.EncryptionKeys[event.KeySentDetails.From] = event.KeySentDetails.Key
				c.config.Connections = append(c.config.Connections, event.KeySentDetails.From)

				fmt.Printf("Received key for user %s", event.KeySentDetails.From)
			}
		}
	}()

	return nil
}
