package eos

import (
	"crypto/sha256"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	"github.com/eoscanada/eos-go/system"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
)

type Storage struct {
	config *config.Config
	api    *eos.API
	log    *logger.Log
}

// New sets the connection to nodeos API up and adds keybag signer
func New(cfg *config.Config, log *logger.Log) (*Storage, error) {
	log.Debugf("Eos:: Adding EOS_API from %s", cfg.EosAPI)
	node := eos.New(cfg.EosAPI)
	node.SetSigner(eos.NewKeyBag())
	s := &Storage{log: log, config: cfg, api: node}
	s.api = node
	return s, nil
}

// AccessReq contains fields needed in sending access-contract related actions
type AccessReq struct {
	Patient eos.AccountName `json:"patient"`
	Account eos.AccountName `json:"account"`
}

// GrantAccess adds `to` field in contract table
func (s *Storage) GrantAccess(to string) error {
	s.log.Debugf("Eos::grantAccess(%s) called", to)

	// Give access action
	action := &eos.Action{
		Account: eos.AN(s.config.EosContractAccount),
		Name:    eos.ActN("grantaccess"),
		Authorization: []eos.PermissionLevel{
			{eos.AN(s.config.EosAccount), eos.PermissionName("active")},
		},
		ActionData: eos.NewActionData(AccessReq{eos.AN(s.config.EosAccount), eos.AN(to)}),
	}
	_, err := s.api.SignPushActions(action)
	s.log.Debugf("Granted access to user; %+v", to)
	return err
}

// RevokeAccess changes isEnabled field in table to 0
func (s *Storage) RevokeAccess(to string) error {
	s.log.Debugf("Eos::revokeAccess(%s) called", to)
	// Remove access action
	action := &eos.Action{
		Account: eos.AN(s.config.EosContractAccount),
		Name:    eos.ActN("revokeaccess"),
		Authorization: []eos.PermissionLevel{
			{eos.AN(s.config.EosAccount), eos.PermissionName("active")},
		},
		ActionData: eos.NewActionData(AccessReq{eos.AN(s.config.EosAccount), eos.AN(to)}),
	}
	_, err := s.api.SignPushActions(action)
	return err
}

// AccessGranted checks if connection between `patient` and `to` is establisehd
// return true if connection is established and false if it is not
// Due to uint32 limitations this functions allows connection for up to 4294967295 doctors to a single client
func (s *Storage) AccessGranted(patient, to string) (bool, error) {
	s.log.Debugf("Eos::accessGranted(%s, %s) called", patient, to)
	if patient == to {
		return true, nil
	}
	// Check if `patient` has its field in the table
	b := false
	list, err := s.ListConnected(patient)
	s.log.Debugf("Got connected users: %s", list)
	if err != nil {
		return b, nil
	}
	for _, entry := range list {
		if entry == to {
			b = true
			break
		}
	}
	return b, nil
}

type TableEntry struct {
	AccountName string `json:"account_name"`
}

func (s *Storage) ListConnected(patient string) ([]string, error) {
	s.log.Debugf("Eos::listAccountFromTable(%s, %v) called", patient)

	// Get the table
	r, err := s.api.GetTableRows(eos.GetTableRowsRequest{JSON: true, Scope: patient, Code: s.config.EosContractAccount, Table: "person", Limit: math.MaxUint32, TableKey: "account_name"})
	if err != nil {
		return nil, err
	}

	// Fill the map with data
	a := make([]TableEntry, 0)
	r.JSONToStructs(&a)

	// fill the list
	ret := []string{}
	for _, entry := range a {
		s.log.Debugf("Adding %s to list", entry.AccountName)
		ret = append(ret, entry.AccountName)
	}

	return ret, nil
}

// DeployContract pushes contract located in contract/eos to blockchain under name specified in config
func (s *Storage) DeployContract() error {
	s.log.Debugf("Eos::deployContract() called")

	if s.config.EosContractAccount == "" {
		return fmt.Errorf("No config.EosContractAccount specified, unable to deploy contract")
	}
	if s.config.EosTokenAccount == "" {
		return fmt.Errorf("No config.EosTokenAccount specified, unable to createa account to deploy contract to")
	}

	err := s.pushContract(s.config.EosContractAccount, s.config.EosContractName)
	if err != nil {
		return fmt.Errorf("Failed to deploy connections contract: %v", err)
	}

	if s.config.EosTokenName == "" {
		return fmt.Errorf("No config.EosTokenName specified, unable to deploy token")
	}
	if s.config.EosAccount == "" {
		return fmt.Errorf("No config.EosAccount specified, unable to createa account to deploy token to")
	}
	err = s.pushContract(s.config.EosTokenAccount, s.config.EosTokenName)
	if err != nil {
		return fmt.Errorf("Failed to deploy new token contract: %v", err)
	}

	return nil
}

func (s *Storage) pushContract(n, cn string) error {
	// import key
	key, err := s.ImportKey(s.config.EosPrivate)
	if err != nil {
		return err
	}
	// create account
	err = s.CreateAccount(n, key.PublicKey().String())
	if err != nil {
		return err
	}

	// Get newcontract actions
	contract, err := system.NewSetContract(eos.AN(n), "../../contract/"+cn+".wasm", "../../contract/"+cn+".abi")
	if err != nil {
		return err
	}

	_, err = s.api.SignPushActions(contract...)
	if err != nil {
		return err
	}

	return nil
}

// CreateAccount creates account named `account` using key `keyStr`
func (s *Storage) CreateAccount(account, keyStr string) error {
	s.log.Debugf("Eos::createAccount(%s) called", account)

	// Check if we have enough resources to create new account
	if err := s.checkBuyResources(s.config.EosAccount); err != nil {
		s.log.Debugf("Bought resources error: %+v", err)
	}

	key, err := ecc.NewPublicKey(keyStr)
	if err != nil {
		return err
	}

	actions := []*eos.Action{}

	// Create new account
	actions = append(actions, system.NewNewAccount(eos.AN(s.config.EosAccount), eos.AN(account), key))

	if s.config.EosRequiresRAM {
		actions = append(actions, system.NewBuyRAMBytes(eos.AccountName(s.config.EosAccount), eos.AccountName(account), 4096))
		actions = append(actions, s.buyCpuNetRequest(account, 10000, 10000))
	}

	err = retry(1*time.Second, 5, func() error {
		r, err := s.api.SignPushActions(actions...)
		if err != nil {
			return err
		}
		s.log.Debugf("Actions pushed. Transaction: %s", r.TransactionID)
		time.Sleep(2500 * time.Millisecond)

		if !s.CheckAccountExists(account) {
			return fmt.Errorf("Account does not exists")
		}

		return err
	})

	if err != nil {
		s.log.Debugf("Failed to create account after 5 attempts; %+v", err)
		return err
	}

	return nil
}

// NewKey creates new private key and saves it to config
func (s *Storage) NewKey() error {
	key, err := ecc.NewRandomPrivateKey()
	if err != nil {
		return err
	}
	s.ImportKey(key.String())
	s.config.EosPrivate = key.String()
	return nil
}

// ImportKey imports private key
// returns  privatekey struct or error
func (s *Storage) ImportKey(prkey string) (*ecc.PrivateKey, error) {
	key, err := ecc.NewPrivateKey(prkey)
	if err != nil {
		return key, err
	}
	s.api.Signer.ImportPrivateKey(prkey)
	return key, nil
}

// CheckAccountKey checks if the key is authority of account
func (s *Storage) CheckAccountKey(account, key string) (bool, error) {
	adata, err := s.api.GetAccount(eos.AN(account))
	if err != nil {
		return false, err
	}
	ret := false
	for _, p := range adata.Permissions {
		for _, keys := range p.RequiredAuth.Keys {
			if keys.PublicKey.String() == key {
				ret = true
			}
		}
	}
	return ret, nil
}

// Sign creates string signature of data
func (s *Storage) SignHash(data []byte) (string, error) {
	h := sha256.New()
	h.Write(data)
	return s.SignByte(h.Sum(nil))
}

func (s *Storage) SignByte(data []byte) (string, error) {
	sk, err := ecc.NewPrivateKey(s.config.EosPrivate)
	if err != nil {
		return "", err
	}
	sign, err := sk.Sign(data)
	return sign.String(), err
}

func (s *Storage) CheckAccountExists(account string) bool {
	_, err := s.api.GetAccount(eos.AN(account))
	if err != nil {
		s.log.Debugf("Error checking account: %v\nReturning false", err)
		return false
	}
	return true
}

func retry(wait time.Duration, attempts int, f func() error) (err error) {
	for i := 0; i < attempts; i++ {
		if err = f(); err == nil {
			log.Printf("Function called successfuly")
			return nil
		}

		time.Sleep(wait)

		log.Println("retrying after error:", err)
	}

	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
