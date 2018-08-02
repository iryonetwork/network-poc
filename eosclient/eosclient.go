package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eos"
)

const tokenKey = "token"

type Client struct {
	config *config.Config
	eos    *eos.Storage
	ehr    *ehr.Storage
	log    *logger.Log
}

func New(config *config.Config, eos *eos.Storage, ehr *ehr.Storage, log *logger.Log) (*Client, error) {
	return &Client{
		config: config,
		eos:    eos,
		ehr:    ehr,
		log:    log,
	}, nil
}
func (c *Client) CreateAccount(key string) (string, error) {
	c.log.Debugf("Client::createaccount(%s) called", key)

	r, err := http.PostForm("http://"+c.config.IryoAddr+"/createaccount",
		url.Values{"key": {key}})
	var a map[string]string
	c.log.Debugf("createaccount returned: %v", r.Body)
	err = json.NewDecoder(r.Body).Decode(&a)
	if err != nil {
		return "", err
	}
	if _, ok := a["error"]; !ok {
		return a["account"], nil
	} else {
		return "", fmt.Errorf(a["error"])
	}

}

func (c *Client) Ls(owner string) ([]map[string]string, error) {
	c.log.Debugf("Client::Ls(%s) called", owner)

	r, err := http.PostForm("http://"+c.config.IryoAddr+"/ls",
		url.Values{"account": {owner}})
	var a map[string][]map[string]string
	c.log.Debugf("ls returned: %v", r.Body)
	err = json.NewDecoder(r.Body).Decode(&a)
	if err != nil {
		return nil, err
	}
	if _, ok := a["error"]; !ok {
		return a["files"], nil
	} else {
		return nil, fmt.Errorf("Could not list files")
	}

}
func (c *Client) Download(owner, fileID, ehrID string) error {
	c.log.Debugf("Client::Download(%s, %s,%s) called", owner, fileID, ehrID)

	// check for permissions
	granted, err := c.eos.AccessGranted(owner, c.config.EosAccount)
	if err != nil {
		return fmt.Errorf("failed to check accessGranted; %v", err)
	}

	if !granted {
		return fmt.Errorf("You do not have permission to download this file")
	}

	// download file from server
	r, err := http.PostForm("http://"+c.config.IryoAddr+"/download",
		url.Values{"ehrID": {ehrID}, "fileID": {fileID}, "account": {owner}})
	if err != nil {
		return err
	}
	var a []byte
	err = json.NewDecoder(r.Body).Decode(&a)

	// save file to local storage
	c.log.Debugf("Calling EHR::SaveID(%s, %s, <Data>)", owner, ehrID)
	c.ehr.Saveid(owner, ehrID, []byte(a))

	return nil
}

// Update downloads files for user, if they do not exist. Remove them if access was removed
func (c *Client) Update(owner string) error {
	// First check if access is granted
	granted, err := c.eos.AccessGranted(owner, c.config.EosAccount)
	if err != nil {
		return err
	}
	if !granted {
		c.ehr.Remove(owner)
		return fmt.Errorf("You don't have permission granted to access this data")
	}

	// List files
	c.log.Debugf("Client::Update(%s) called", owner)
	list, err := c.Ls(owner)
	if err != nil {
		return err
	}
	// Download missing
	for _, f := range list {
		c.log.Debugf("Client::Fill: Checking file: %s ", f["fileID"])
		if !c.ehr.Exists(owner, f["ehrID"]) {
			c.log.Debugf("Client::Fill: Trying to download file: %s ", f["fileID"])
			err = c.Download(owner, f["fileID"], f["ehrID"])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) Upload(owner, ehrid string) error {
	c.log.Debugf("Client::upload(%s) called", owner)

	// check for permissions
	if owner != c.config.EosAccount {
		granted, err := c.eos.AccessGranted(owner, c.config.EosAccount)
		if err != nil {
			return fmt.Errorf("failed to check accessGranted; %v", err)
		}

		if !granted {
			return fmt.Errorf("You do not have permission to upload this file")
		}
	}
	// get data from local storage
	data := c.ehr.Getid(owner, ehrid)
	if data == nil {
		return fmt.Errorf("Document for %s does not exist", owner)
	}
	// lets get a signature
	sign, err := c.eos.Sign(data)
	if err != nil {
		return err
	}
	// upload data to server
	_, err = http.PostForm("http://"+c.config.IryoAddr+"/upload",
		url.Values{"ehrID": {ehrid}, "data": {string(data)}, "account": {c.config.EosAccount},
			"sign": {sign}, "key": {c.config.GetEosPublicKey()}, "owner": {owner}})

	if err != nil {
		return fmt.Errorf("failed to call Upload; %v", err)
	}

	return nil
}

func (c *Client) GrantAccess(to string) error {
	c.log.Debugf("Client::grantAccess(%s) called", to)

	// write access granted to blockchain
	err := c.eos.GrantAccess(to)
	if err != nil {
		return fmt.Errorf("failed to call grantAccess; %v", err)
	}

	// // send key for storage encryption
	// err = retry(10, 2*time.Second, func() error {
	// 	_, err = c.client.SendKey(metadata.NewOutgoingContext(context.Background(), c.metadata), &specs.SendKeyRequest{
	// 		To:  to,
	// 		Key: c.config.EncryptionKeys[c.config.GetEthPublicAddress()],
	// 	})

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

	// 	return err
	// })

	return err
}

func (c *Client) RevokeAccess(to string) error {
	c.log.Debugf("Client::revokeAccess(%s) called", to)

	// // send empty key to doctor to revoke the access
	// _, err := c.client.SendKey(metadata.NewOutgoingContext(context.Background(), c.metadata), &specs.SendKeyRequest{
	// 	To:  to,
	// 	Key: []byte{},
	// })
	// if err != nil {
	// 	return fmt.Errorf("failed to call SendKey: %v", err)
	// }

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
	return c.eos.RevokeAccess(to)
}

func (c *Client) Subscribe() error {
	c.log.Debugf("Client::subscribe() called")

	// // subscribe to key sent event
	// stream, err := c.client.Subscribe(metadata.NewOutgoingContext(context.Background(), c.metadata), &specs.Empty{})
	// if err != nil {
	// 	return fmt.Errorf("failed to call subscribe; %v", err)
	// }

	// go func() {
	// 	for {
	// 		event, err := stream.Recv()
	// 		if err == io.EOF {
	// 			break
	// 		}
	// 		if err != nil {
	// 			c.log.Fatalf("error receiving events: %v", err)
	// 		}

	// 		if event.Type == specs.Event_KeySent {
	// 			c.config.EncryptionKeys[event.KeySentDetails.From] = event.KeySentDetails.Key
	// 			found := false
	// 			for i, connection := range c.config.Connections {
	// 				if connection == event.KeySentDetails.From {
	// 					if len(event.KeySentDetails.Key) == 0 {
	// 						c.config.Connections = append(c.config.Connections[:i], c.config.Connections[i+1:]...)
	// 					}
	// 					found = true
	// 					break
	// 				}
	// 			}
	// 			if !found && len(event.KeySentDetails.Key) > 0 {
	// 				c.config.Connections = append(c.config.Connections, event.KeySentDetails.From)
	// 			}

	// 			fmt.Printf("Received key for user %s", event.KeySentDetails.From)
	// 		}
	// 	}
	// }()

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
