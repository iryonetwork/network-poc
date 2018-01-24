package client

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
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
	log      *logger.Log
}

func New(config *config.Config, client specs.CloudClient, eth *eth.Storage, ehr *ehr.Storage, log *logger.Log) (*RPCClient, error) {
	// generate random hash
	hash := make([]byte, 32)
	_, err := rand.Read(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random hash; %v", err)
	}

	// sign hash with private key
	sig, err := config.EthPrivate.Sign(rand.Reader, hash, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to sign the login request; %v", err)
	}

	// login with gRPC
	response, err := client.Login(context.Background(), &specs.LoginRequest{
		Public:    crypto.CompressPubkey(&config.EthPrivate.PublicKey),
		Signature: sig,
		Hash:      hash,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call login; %v", err)
	}

	// save login token to metadata for later gRPC calls
	md := metadata.Pairs(tokenKey, response.Token)
	config.EthContractAddr = response.ContractAddress

	return &RPCClient{
		client:   client,
		config:   config,
		metadata: md,
		eth:      eth,
		ehr:      ehr,
		log:      log,
	}, nil
}

func (c *RPCClient) Download(owner string) error {
	c.log.Debugf("RPCClient::Download(%s) called", owner)

	// check for permissions
	granted, err := c.eth.AccessGranted(owner, c.config.GetEthPublicAddress())
	if err != nil {
		return fmt.Errorf("failed to check accessGranted; %v", err)
	}

	if !granted {
		return fmt.Errorf("You do not have permission to download this file")
	}

	// download file from server
	response, err := c.client.Download(metadata.NewOutgoingContext(context.Background(), c.metadata), &specs.DownloadRequest{
		Owner: owner,
	})
	if err != nil {
		return fmt.Errorf("failed to call download; %v", err)
	}

	// save file to local storage
	c.ehr.Save(owner, response.Data)

	return nil
}

func (c *RPCClient) Upload(owner string) error {
	c.log.Debugf("RPCClient::upload(%s) called", owner)

	// check for permissions
	granted, err := c.eth.AccessGranted(owner, c.config.GetEthPublicAddress())
	if err != nil {
		return fmt.Errorf("failed to check accessGranted; %v", err)
	}

	if !granted {
		return fmt.Errorf("You do not have permission to upload this file")
	}

	// get data from local storage
	data := c.ehr.Get(owner)
	if data == nil {
		return fmt.Errorf("Document for %s does not exist", owner)
	}

	// upload data to server
	_, err = c.client.Upload(metadata.NewOutgoingContext(context.Background(), c.metadata), &specs.UploadRequest{
		Owner: owner,
		Data:  data,
	})

	if err != nil {
		return fmt.Errorf("failed to call Upload; %v", err)
	}

	return nil
}

func (c *RPCClient) GrantAccess(to string) error {
	c.log.Debugf("RPCClient::grantAccess(%s) called", to)

	// write access granted to blockchain
	err := c.eth.GrantAccess(to)
	if err != nil {
		return fmt.Errorf("failed to call grantAccess; %v", err)
	}

	// send key for storage encryption
	err = retry(10, 2*time.Second, func() error {
		_, err = c.client.SendKey(metadata.NewOutgoingContext(context.Background(), c.metadata), &specs.SendKeyRequest{
			To:  to,
			Key: c.config.EncryptionKeys[c.config.GetEthPublicAddress()],
		})

		if err == nil {
			found := false
			for _, connection := range c.config.Connections {
				if connection == to {
					found = true
					break
				}
			}
			if !found {
				c.config.Connections = append(c.config.Connections, to)
			}
		} else {
			err = fmt.Errorf("failed to call SendKey; %v", err)
		}

		return err
	})

	return err
}

func (c *RPCClient) RevokeAccess(to string) error {
	c.log.Debugf("RPCClient::revokeAccess(%s) called", to)

	// send empty key to doctor to revoke the access
	_, err := c.client.SendKey(metadata.NewOutgoingContext(context.Background(), c.metadata), &specs.SendKeyRequest{
		To:  to,
		Key: []byte{},
	})
	if err != nil {
		return fmt.Errorf("failed to call SendKey: %v", err)
	}

	// remove doctor from our connections
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

	// write revoke access to blockchain
	return c.eth.RevokeAccess(to)
}

func (c *RPCClient) Subscribe() error {
	c.log.Debugf("RPCClient::subscribe() called")

	// subscribe to key sent event
	stream, err := c.client.Subscribe(metadata.NewOutgoingContext(context.Background(), c.metadata), &specs.Empty{})
	if err != nil {
		return fmt.Errorf("failed to call subscribe; %v", err)
	}

	go func() {
		for {
			event, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				c.log.Fatalf("error receiving events: %v", err)
			}

			if event.Type == specs.Event_KeySent {
				c.config.EncryptionKeys[event.KeySentDetails.From] = event.KeySentDetails.Key
				found := false
				for i, connection := range c.config.Connections {
					if connection == event.KeySentDetails.From {
						if len(event.KeySentDetails.Key) == 0 {
							c.config.Connections = append(c.config.Connections[:i], c.config.Connections[i+1:]...)
						}
						found = true
						break
					}
				}
				if !found && len(event.KeySentDetails.Key) > 0 {
					c.config.Connections = append(c.config.Connections, event.KeySentDetails.From)
				}

				fmt.Printf("Received key for user %s", event.KeySentDetails.From)
			}
		}
	}()

	return nil
}

func retry(attempts int, sleep time.Duration, f func() error) (err error) {
	for i := 0; i < attempts; i++ {
		if err = f(); err == nil {
			return nil
		}

		time.Sleep(sleep)

		log.Println("retrying after error:", err)
	}

	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
