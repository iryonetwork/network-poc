package client

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
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
	token  string
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

func (c *Client) Login() error {
	// generate random hash
	hash := make([]byte, 32)
	_, err := rand.Read(hash)
	if err != nil {
		return fmt.Errorf("failed to generate random hash; %v", err)
	}

	// sign hash with private key
	sig, err := c.eos.SignByte(hash)
	if err != nil {
		return fmt.Errorf("Failed to sign the login request; %v", err)
	}
	req := url.Values{"sign": {sig}, "key": {c.config.GetEosPublicKey()}, "hash": {string(hash)}}
	if account := c.config.EosAccount; account != "" {
		req["account"] = []string{account}
	}
	// send login request
	response, err := http.PostForm(fmt.Sprintf("http://%s/login", c.config.IryoAddr), req)
	if err != nil {
		return fmt.Errorf("failed to call login; %v", err)
	}
	if response.StatusCode != 201 {
		return fmt.Errorf("Code: %d", response.StatusCode)
	}
	// save token to client
	token, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	c.token = string(token)
	return nil
}

func (c *Client) CreateAccount(key string) (string, error) {
	c.log.Debugf("Client::createaccount(%s) called", key)
	r, err := http.NewRequest("GET", fmt.Sprintf("http://%s/account/%s", c.config.IryoAddr, key), nil)
	if err != nil {
		return "", err
	}
	r.Header.Add("Authorization", c.token)
	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		return "", err
	}
	if res.StatusCode != 201 {
		return "", fmt.Errorf("Code: %d", res.StatusCode)
	}
	var a map[string]string
	err = json.NewDecoder(res.Body).Decode(&a)
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

	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/%s", c.config.IryoAddr, owner), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", c.token)
	client := &http.Client{}
	res, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Code: %d", res.StatusCode)
	}
	var a map[string][]map[string]string
	c.log.Debugf("Client:: ls returned: %v", res.Body)
	err = json.NewDecoder(res.Body).Decode(&a)
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
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/%s/%s", c.config.IryoAddr, owner, fileID), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", c.token)
	client := &http.Client{}
	res, err := client.Do(req)

	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("Code: %d", res.StatusCode)
	}

	a, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	// save file to local storage
	c.ehr.Saveid(owner, fileID, []byte(a))

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
		if !c.ehr.Exists(owner, f["fileID"]) {
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
	// get data from local storage
	data := c.ehr.Getid(owner, ehrid)
	if data == nil {
		return fmt.Errorf("Document for %s does not exist", owner)
	}
	// lets get a signature
	sign, err := c.eos.SignHash(data)
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
	req.Header.Add("Authorization", c.token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Upload; %v", err)
	}
	b, _ := ioutil.ReadAll(res.Body)
	if string(b) == "unknown token" {
		return fmt.Errorf(string(b))
	}
	if res.StatusCode != 201 {
		return fmt.Errorf("Got code: %d", res.StatusCode)
	}
	ret := make(map[string]string)
	json.NewDecoder(res.Body).Decode(ret)
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
