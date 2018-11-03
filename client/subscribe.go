package client

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/eoscanada/eos-go/ecc"

	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/openEHR/ehrdata"
	"github.com/iryonetwork/network-poc/requests"
	"github.com/iryonetwork/network-poc/state"
	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eos"
)

type subscribe struct {
	eos      *eos.Storage
	ehr      *ehr.Storage
	requests *requests.Requests
	ws       *Ws
	client   *Client
	state    *state.State
	log      *logger.Log
}

func NewMessageHandler(state *state.State, ehr *ehr.Storage, eos *eos.Storage, log *logger.Log) *subscribe {
	return &subscribe{eos: eos, ehr: ehr, state: state, log: log}
}

func (s *subscribe) SetClient(client *Client) MessageHandler {
	s.client = client

	return s
}

func (s *subscribe) SetWs(ws *Ws) MessageHandler {
	s.ws = ws

	return s
}

func (s *subscribe) SetRequests(requests *requests.Requests) MessageHandler {
	s.requests = requests

	return s
}

func (s *subscribe) ImportKey(r *requests.Request) {
	keyenc, err := base64.StdEncoding.DecodeString(subscribeGetStringDataFromRequest(r, "key", s.log))
	if err != nil {
		s.log.Debugf("Error decoding key from base64; %v", err)
	}
	from := subscribeGetStringDataFromRequest(r, "from", s.log)
	name := subscribeGetStringDataFromRequest(r, "name", s.log)
	customData := subscribeGetStringDataFromRequest(r, "customData", s.log)

	rnd := rand.Reader
	key, err := rsa.DecryptOAEP(sha512.New(), rnd, s.state.RSAKey, keyenc, []byte{})
	if err != nil {
		s.log.Printf("Error decrypting key: %v", err)
		return
	}

	s.log.Debugf("SUBSCRIPTION:: Importing key from user %s (%s)", from, customData)

	s.state.Directory[from] = name
	s.state.EncryptionKeys[from] = key

	exists := false
	for _, name := range s.state.Connections.WithKey {
		if name == from {
			exists = true
		}
	}
	if !exists {
		s.state.Connections.WithKey = append(s.state.Connections.WithKey, from)
	}

	s.log.Debugf("SUBSCRIPTION:: Imported key from %s ", from)
}

func (s *subscribe) RevokeKey(r *requests.Request) {
	from := subscribeGetStringDataFromRequest(r, "from", s.log)

	s.log.Debugf("SUBSCRIPTION:: Revoking %s's key", from)

	s.ehr.RemoveUser(from)
	delete(s.state.EncryptionKeys, from)

	for i, v := range s.state.Connections.WithKey {
		if v == from {
			s.state.Connections.WithKey = append(s.state.Connections.WithKey[:i], s.state.Connections.WithKey[i+1:]...)
			s.log.Debugf("SUBSCRIPTION:: Revoked %s's key ", from)
		}
	}
}

func (s *subscribe) SubReencrypt(r *requests.Request) {
	from := subscribeGetStringDataFromRequest(r, "from", s.log)
	s.ehr.RemoveUser(from)
	err := s.requests.RequestsKey(from, "")
	if err != nil {
		s.log.Printf("Error creating RequestKey: %v", err)
	}
}

func (s *subscribe) AccessWasGranted(r *requests.Request) {
	name := subscribeGetStringDataFromRequest(r, "name", s.log)
	from := subscribeGetStringDataFromRequest(r, "from", s.log)

	s.log.Debugf("Got notification 'accessGranted' from %s", from)
	s.state.Directory[from] = name

	// Check if we already have the user on the list
	onlist := false
	for _, v := range s.state.Connections.WithKey {
		if v == from {
			onlist = true
			return
		}
	}
	// if its not on list add it
	if !onlist {
		s.state.Connections.WithoutKey = append(s.state.Connections.WithoutKey, from)
	}
}

func (s *subscribe) NotifyKeyRequested(r *requests.Request) {
	s.log.Debugf("SUBSCRIPTION:: Got RequestKey request")
	from := subscribeGetStringDataFromRequest(r, "from", s.log)
	name := subscribeGetStringDataFromRequest(r, "name", s.log)
	rsakey := subscribeGetDataFromRequest(r, "key", s.log)
	sign := subscribeGetStringDataFromRequest(r, "signature", s.log)
	customData := subscribeGetStringDataFromRequest(r, "customData", s.log)

	// Check if account and key are connected
	valid, err := s.verifyRequestKeyRequest(sign, from, rsakey)
	if err != nil {
		s.log.Printf("Error checking valid account: %v", err)
		return
	}
	if !valid {
		s.log.Printf("SUBSCRIBE:: request could not be verified")
		return
	}

	// Save the request to storage for later usage
	pubKey, err := rsaPEMKeyToRSAPublicKey(rsakey)
	if err != nil {
		s.log.Printf("SUBSCRIBE:: Error getting rsa public key; %v", err)
	}
	s.state.Connections.Requested[from] = state.Request{Key: pubKey, CustomData: customData}

	// Add user to directory
	s.state.Directory[from] = name

	// Check if access is already granted
	// if it is, send the key without prompting the user for confirmation
	granted, err := s.eos.AccessGranted(s.state.EosAccount, from)
	if err != nil {
		s.log.Printf("Error getting key: %v", err)
	}
	if granted {
		s.requests.SendKey(from)

		// make sure they are on the list
		add := false
		for _, name := range s.state.Connections.GrantedTo {
			if name == from {
				add = false
			}
		}
		if add {
			s.state.Connections.GrantedTo = append(s.state.Connections.GrantedTo, from)
		}
		// Delete the user from requests
		delete(s.state.Connections.Requested, from)
	}
}

func (s *subscribe) NewUpload(r *requests.Request) {
	account := subscribeGetStringDataFromRequest(r, "user", s.log)

	s.log.Debugf("New file for user: %s", account)

	err := s.client.Update(account)
	if err != nil {
		s.log.Debugf("error updating: %v", err)
	}

	dataMap, err := ehrdata.ExtractEhrData(account, s.ehr, s.state)
	if err != nil {
		s.log.Debugf("Error getting ehrdata: %v", err)
	}

	data, err := json.Marshal(dataMap)
	if err != nil {
		s.log.Debugf("Error marshaling json: %v", err)
	}
	userdata := []byte(fmt.Sprintf(`{"account":"%s"}`, account))
	for _, conn := range s.ws.frontendConn {
		err = conn.WriteMessage(1, append(userdata, data...))
		if err != nil {
			s.log.Debugf("Error writing message: %v", err)
		}
	}
}

func subscribeGetStringDataFromRequest(r *requests.Request, key string, log *logger.Log) string {
	out, err := r.GetDataString(key)
	if err != nil {
		log.Printf("Error getting `%s`: %v", key, err)
	}
	return out
}

func subscribeGetDataFromRequest(r *requests.Request, key string, log *logger.Log) []byte {
	out, err := r.GetData(key)
	if err != nil {
		log.Printf("Error getting `%s`: %v", key, err)
	}
	return out
}

func rsaPEMKeyToRSAPublicKey(pubPEMData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pubPEMData)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	pubKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Failed to cast pub to rsa.PublicKey (got %T)", pub)
	}

	return pubKey, err
}

func (s *subscribe) verifyRequestKeyRequest(signature, from string, rsakey []byte) (bool, error) {
	eoskey, err := requestGetKeyFromSignature(signature, rsakey)
	if err != nil {
		return false, err
	}

	return s.eos.CheckAccountKey(from, eoskey.String())
}

func requestGetKeyFromSignature(strsign string, rsakey []byte) (ecc.PublicKey, error) {
	sign, err := ecc.NewSignature(strsign)
	if err != nil {
		return ecc.PublicKey{}, err
	}

	key, err := sign.PublicKey(getHash(rsakey))
	if err != nil {
		return ecc.PublicKey{}, fmt.Errorf("Signture could not be verified; %v", err)
	}
	return key, nil
}

func getHash(in []byte) []byte {
	sha := sha256.New()
	sha.Write(in)
	return sha.Sum(nil)
}
