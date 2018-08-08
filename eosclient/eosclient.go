package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eos"
	"github.com/iryonetwork/network-poc/storage/ws"
)

const tokenKey = "token"

type Client struct {
	config *config.Config
	eos    *eos.Storage
	ehr    *ehr.Storage
	log    *logger.Log
	ws     *ws.Storage
}

func New(config *config.Config, eos *eos.Storage, ehr *ehr.Storage, log *logger.Log) (*Client, error) {
	return &Client{
		config: config,
		eos:    eos,
		ehr:    ehr,
		log:    log,
	}, nil
}
func (c *Client) AddWs(ws *ws.Storage) {
	c.ws = ws
}
func (c *Client) CloseWs() {
	c.ws.Close()
	c.ws = nil
}
func (c *Client) CreateAccount(key string) (string, error) {
	c.log.Debugf("Client::createaccount(%s) called", key)
	r, err := http.Get(fmt.Sprintf("http://%s/account/%s", c.config.IryoAddr, key))
	if err != nil {
		return "", err
	}
	if r.StatusCode != 201 {
		return "", fmt.Errorf("Code: %d", r.StatusCode)
	}
	var a map[string]string
	err = json.NewDecoder(r.Body).Decode(&a)
	c.log.Debugf("Client:: createaccount returned: %v", a)
	if err != nil {
		return "", err
	}

	if _, ok := a["error"]; ok {
		return "", fmt.Errorf(a["error"])
	}
	return a["account"], nil

}

func (c *Client) Ls(owner string) ([]map[string]string, error) {
	c.log.Debugf("Client::Ls(%s) called", owner)

	r, err := http.Get(fmt.Sprintf("http://%s/%s", c.config.IryoAddr, owner))
	if err != nil {
		return nil, err
	}
	if r.StatusCode != 200 {
		return nil, fmt.Errorf("Code: %d", r.StatusCode)
	}
	var a map[string][]map[string]string
	c.log.Debugf("Client:: ls returned: %v", r.Body)
	err = json.NewDecoder(r.Body).Decode(&a)
	if err != nil {
		return nil, err
	}
	if _, ok := a["error"]; ok {
		return nil, fmt.Errorf("Could not list files")
	}
	return a["files"], nil

}
func (c *Client) Download(owner, fileID string) error {
	c.log.Debugf("Client::Download(%s, %s) called", owner, fileID)

	// download file from server
	r, err := http.Get(fmt.Sprintf("http://%s/%s/%s", c.config.IryoAddr, owner, fileID))
	if err != nil {
		return err
	}
	if r.StatusCode != 200 {
		return fmt.Errorf("Code: %d", r.StatusCode)
	}

	a, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	// save file to local storage
	c.ehr.Save(owner, []byte(a))

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
		c.log.Debugf("Client::Update: Checking file: %s ", f["fileID"])
		if !c.ehr.Exists(owner, f["ehrID"]) {
			c.log.Debugf("Client::Update: Trying to download file: %s ", f["fileID"])
			err = c.Download(owner, f["fileID"])
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

	// Get body for the request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("account", c.config.EosAccount)
	writer.WriteField("key", c.config.GetEosPublicKey())
	writer.WriteField("sign", sign)
	part, err := writer.CreateFormFile("data", ehrid)
	part.Write(data)

	err = writer.Close()
	if err != nil {
		return err
	}

	// upload data to server
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/%s", c.config.IryoAddr, owner), body)
	if err != nil {
		return fmt.Errorf("failed to call Upload; %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	r, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Upload; %v", err)
	}
	if r.StatusCode != 201 {
		return fmt.Errorf("Got code: %d", r.StatusCode)
	}
	ret := make(map[string]string)
	json.NewDecoder(r.Body).Decode(ret)
	c.log.Printf("Response: %s", ret)
	return nil
}

func (c *Client) GrantAccess(to string) error {
	c.log.Debugf("Client::grantAccess(%s) called", to)

	// Make sure that reciever exists
	if !c.eos.CheckAccountExists(to) {
		return fmt.Errorf("User does not exists")
	}

	// Check that users are not yet connected
	if ok, err := c.eos.AccessGranted(c.config.EosAccount, to); ok {
		if err != nil {
			return err
		}
		// make sure doctor is on list of connected
		conn := false
		for _, v := range c.config.Connections {
			if v == to {
				conn = true
			}
		}
		if !conn {
			c.config.Connections = append(c.config.Connections, to)
		}
		return nil
	}

	// write access granted to blockchain
	err := c.eos.GrantAccess(to)
	if err != nil {
		return fmt.Errorf("failed to call grantAccess; %v", err)
	}

	// send key for storage encryption
	err = c.ws.SendKey(to)
	if err != nil {
		return fmt.Errorf("failed to send key; %v", err)
	}

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

	return err
}

func (c *Client) RevokeAccess(to string) error {
	c.log.Debugf("Client::revokeAccess(%s) called", to)

	// send empty key to doctor to revoke the access
	err := c.ws.RevokeKey(to)
	if err != nil {
		return fmt.Errorf("Error revoking key: %v", err)
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
	return c.eos.RevokeAccess(to)
}

func (c *Client) Subscribe() {
	c.log.Debugf("Client::subscribe() called")

	//subscribe to key sent event
	c.ws.Subscribe()
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
