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
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/requests"
	"github.com/iryonetwork/network-poc/state"
	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eos"
)

type Client struct {
	config         *config.Config
	state          *state.State
	eos            *eos.Storage
	ehr            *ehr.Storage
	messageHandler MessageHandler
	log            *logger.Log
	ws             *Ws
	request        *requests.Requests
}

func New(config *config.Config, state *state.State, eos *eos.Storage, ehr *ehr.Storage, messageHandler MessageHandler, log *logger.Log) *Client {
	c := &Client{
		config:         config,
		state:          state,
		eos:            eos,
		ehr:            ehr,
		messageHandler: messageHandler,
		log:            log,
	}
	messageHandler.SetClient(c)

	return c
}

func (c *Client) ConnectWs() error {
	c.Login()
	wsStorage, err := ConnectWs(c.config, c.state, c.log, c.messageHandler, c.ehr, c.eos)
	if err != nil {
		return err
	}
	c.ws = wsStorage
	c.request = requests.NewRequests(c.log, c.config, c.state, wsStorage.Conn(), c.eos)
	c.messageHandler.SetRequests(c.request)
	return nil
}

func (c *Client) CloseWs() {
	c.ws.Close()
	c.state.Connected = false
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
	sig, err := c.eos.SignHash(hash)
	if err != nil {
		return fmt.Errorf("Failed to sign the login request; %v", err)
	}

	req := url.Values{"sign": {sig}, "key": {c.state.GetEosPublicKey()}, "hash": {string(hash)}}
	if account := c.state.EosAccount; account != "" {
		req["account"] = []string{account}
	}

	// send login request
	response, err := http.PostForm(fmt.Sprintf("%s/login", c.config.IryoAddr), req)
	if err != nil {
		return fmt.Errorf("failed to call login; %v", err)
	}
	if response.StatusCode != 201 {
		return fmt.Errorf("Code: %d", response.StatusCode)
	}
	data := make(map[string]string)
	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		return err
	}
	// Login again after token expires
	go c.loginWaiter(data["validUntil"])
	// save token to client
	c.state.Token = data["token"]
	return nil
}

func (c *Client) loginWaiter(str string) {
	i, err := strconv.ParseInt(str, 10, 64)
	// make request 5 seconds before token expires
	i -= 5
	if err != nil {
		c.log.Fatalf("Error getting validUntil")
	}
	time.Sleep(time.Until(time.Unix(i, 0)))
	if err := c.Login(); err != nil {
		log.Fatalf("Error logging in")
	}
}

func (c *Client) CreateAccount(key string) (string, error) {
	c.log.Debugf("Client::createaccount(%s) called", key)

	client := &http.Client{}

	data := url.Values{"name": {c.state.PersonalData.Name}}
	data.Add("name", c.state.PersonalData.Name)
	r, err := http.NewRequest("POST", fmt.Sprintf("%s/account", c.config.IryoAddr), strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Authorization", c.state.Token)
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

	// Check for errors
	if _, ok := a["error"]; ok {
		return "", fmt.Errorf(a["error"])
	}

	return a["account"], nil
}

func (c *Client) Ls(owner string) ([]map[string]string, error) {

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", c.config.IryoAddr, owner), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", c.state.Token)
	client := &http.Client{}
	res, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Code: %d", res.StatusCode)
	}
	var a map[string][]map[string]string
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
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s/%s", c.config.IryoAddr, owner, fileID), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", c.state.Token)
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
	if owner != c.state.EosAccount {
		granted, err := c.eos.AccessGranted(owner, c.state.EosAccount)
		if err != nil {
			return err
		}
		if !granted {
			c.ehr.RemoveUser(owner)
			return fmt.Errorf("You don't have permission granted to access this data")
		}
	}
	// List files
	list, err := c.Ls(owner)
	if err != nil {
		return err
	}
	// Download missing
	for _, f := range list {
		if !c.ehr.Exists(owner, f["fileID"]) {
			err = c.Download(owner, f["fileID"])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) Upload(owner, id string, reupload bool) error {
	c.log.Debugf("Client::upload(%s) called", owner)
	// get data from local storage
	data := c.ehr.Getid(owner, id)
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
	writer.WriteField("account", c.state.EosAccount)
	writer.WriteField("key", c.state.GetEosPublicKey())
	writer.WriteField("sign", sign)
	part, err := writer.CreateFormFile("data", id)
	if err != nil {
		return err
	}
	part.Write(data)

	err = writer.Close()
	if err != nil {
		return err
	}

	// upload data to server
	req := &http.Request{}
	if !reupload {
		req, err = http.NewRequest("POST", fmt.Sprintf("%s/%s", c.config.IryoAddr, owner), body)
	} else {
		req, err = http.NewRequest("PUT", fmt.Sprintf("%s/%s/%s", c.config.IryoAddr, owner, id), body)
	}

	if err != nil {
		return fmt.Errorf("failed to call Upload; %v", err)
	}
	req.Header.Add("Authorization", c.state.Token)
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
	a := make(map[string]string)
	err = json.Unmarshal(b, &a)
	if err != nil {
		return err
	}

	if !reupload {
		newid := a["fileID"]
		c.ehr.Rename(owner, id, newid)
	}

	return nil
}

func (c *Client) Reencrypt(key []byte) error {
	c.log.Debugf("Client:: Reencrypting")
	err := c.Update(c.state.EosAccount)
	if err != nil {
		return err
	}
	err = c.ehr.Reencrypt(c.state.EosAccount, c.state.EncryptionKeys[c.state.EosAccount], key)
	if err != nil {
		return err
	}

	// Save the new key
	c.state.EncryptionKeys[c.state.EosAccount] = key

	// Send notification about reencrypting to api
	err = c.request.ReencryptRequest()
	if err != nil {
		return err
	}

	// Take some time, to make sure that api folder no loger contains files
	time.Sleep(2 * time.Second)

	// Upload new files
	for id := range c.ehr.Get(c.state.EosAccount) {
		c.log.Debugf("Uploading %s", id)
		c.Upload(c.state.EosAccount, id, true)
	}
	return nil
}
func (c *Client) GrantAccess(to string) error {
	c.log.Debugf("Client::grantAccess(%s) called", to)

	// Make sure that reciever exists
	if !c.eos.CheckAccountExists(to) {
		c.log.Debugf("User %s does not exits", to)
		return fmt.Errorf("User does not exists")
	}

	// Check that users are not yet connected
	if ok, err := c.eos.AccessGranted(c.state.EosAccount, to); ok {
		c.log.Debugf("Access already granted to %s", to)
		if err != nil {
			return err
		}
		// make sure doctor is on list of connected
		conn := false
		for _, v := range c.state.Connections.GrantedTo {
			if v == to {
				conn = true
			}
		}
		if !conn {
			c.state.Connections.GrantedTo = append(c.state.Connections.GrantedTo, to)
		}
		return nil
	}

	// write access granted to blockchain
	c.log.Debugf("Granting access to %s", to)
	err := c.eos.GrantAccess(to)
	if err != nil {
		return fmt.Errorf("failed to call grantAccess; %v", err)
	}

	// if request for key was already made send the key
	// else notify the doctor that request was granted
	if _, ok := c.state.Connections.Requested[to]; ok {
		// send key for storage encryption
		err = c.request.SendKey(to)
		if err != nil {
			return fmt.Errorf("failed to send key; %v", err)
		}
	} else {
		err = c.request.NotifyGranted(to, "")
		if err != nil {
			return fmt.Errorf("Failed to notify access granted; %v", err)
		}
	}

	// add doctor to list of connected
	conn := false
	for _, v := range c.state.Connections.GrantedTo {
		if v == to {
			conn = true
		}
	}
	if !conn {
		c.state.Connections.GrantedTo = append(c.state.Connections.GrantedTo, to)
	}

	/// remove key from storage
	delete(c.state.Connections.Requested, to)

	return nil
}

func (c *Client) RevokeAccess(to string) error {
	c.log.Debugf("Client::revokeAccess(%s) called", to)

	// send empty key to doctor to revoke the access
	err := c.request.RevokeKey(to)
	if err != nil {
		return fmt.Errorf("Error revoking key: %v", err)
	}

	// write revoke access to blockchain
	err = c.eos.RevokeAccess(to)
	if err != nil {
		return err
	}
	// remove doctor from our connections
	found := false
	for i, v := range c.state.Connections.GrantedTo {
		if v == to {
			c.state.Connections.GrantedTo = append(c.state.Connections.GrantedTo[:i], c.state.Connections.GrantedTo[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("%s is not in connections", to)
	}
	return nil
}

// CheckGrantedStatus checks if access is granted for given account and if the encryption key is available.
func (c *Client) CheckGrantedStatus(eosAccountName string) (accessGranted bool, keyAvailable bool, err error) {
	c.log.Debugf("Client::CheckGrantedStatus(%s) called", eosAccountName)

	// Check if access was granted
	ok, err := c.eos.AccessGranted(c.state.EosAccount, eosAccountName)
	if err != nil {
		return false, false, err
	}

	if ok {
		// check if key is available and make sure connections lists are correct
		if _, ok := c.state.EncryptionKeys[eosAccountName]; !ok {
			for i, v := range c.state.Connections.WithKey {
				if v == eosAccountName {
					c.state.Connections.WithKey = append(c.state.Connections.WithKey[:i], c.state.Connections.WithKey[i+1:]...)
				}
			}
			onlist := false
			for _, v := range c.state.Connections.WithoutKey {
				if v == eosAccountName {
					onlist = true
				}
			}
			if !onlist {
				c.state.Connections.WithoutKey = append(c.state.Connections.WithoutKey, eosAccountName)
			}

			return true, false, nil
		}

		for i, v := range c.state.Connections.WithoutKey {
			if v == eosAccountName {
				c.state.Connections.WithoutKey = append(c.state.Connections.WithoutKey[:i], c.state.Connections.WithoutKey[i+1:]...)
			}
		}
		onlist := false
		for _, v := range c.state.Connections.WithKey {
			if v == eosAccountName {
				onlist = true
			}
		}
		if !onlist {
			c.state.Connections.WithKey = append(c.state.Connections.WithKey, eosAccountName)
		}

		return true, true, nil
	}

	// access is not granted, make sure to remove from all the list
	c.ehr.RemoveUser(eosAccountName)
	delete(c.state.EncryptionKeys, eosAccountName)
	for i, v := range c.state.Connections.WithKey {
		if v == eosAccountName {
			c.state.Connections.WithKey = append(c.state.Connections.WithKey[:i], c.state.Connections.WithKey[i+1:]...)
		}
	}
	for i, v := range c.state.Connections.WithoutKey {
		if v == eosAccountName {
			c.state.Connections.WithoutKey = append(c.state.Connections.WithoutKey[:i], c.state.Connections.WithoutKey[i+1:]...)
		}
	}

	return false, false, nil
}

func (c *Client) RequestAccess(to, customData string) error {
	err := c.request.RequestsKey(to, customData)
	return err
}

func (c *Client) NewRequestKeyQr(customData string) string {
	qrKey := fmt.Sprintf("%s\n%s", c.state.EosAccount, c.state.PersonalData.Name)
	if customData != "" {
		qrKey = fmt.Sprintf("%s\n%s", qrKey, customData)
	}

	return qrKey
}

func (c *Client) SaveAndUploadEhrData(user string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return c.SaveAndUploadData(user, jsonData)
}

func (c *Client) SaveAndUploadData(user string, data []byte) error {
	id, err := c.ehr.Encrypt(user, data, c.state.EncryptionKeys[user])
	if err != nil {
		return err
	}

	return c.Upload(user, id, false)
}

func (c *Client) AddFrontendWS(conn *websocket.Conn) {
	c.ws.frontendConn = append(c.ws.frontendConn, conn)
}

func (c *Client) RemoveFrontendWS(conn *websocket.Conn) error {
	deleted := false
	for i, v := range c.ws.frontendConn {
		if v == conn {
			c.ws.frontendConn = append(c.ws.frontendConn[:i], c.ws.frontendConn[i+1:]...)
			deleted = true
		}
	}
	if !deleted {
		return fmt.Errorf("Ws connection not found")
	}
	c.log.Debugf("Closing client's ws connection")
	return conn.Close()
}
