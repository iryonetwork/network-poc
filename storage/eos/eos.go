package eos

import (
	"crypto/sha256"
	"fmt"
	"math"

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

type AddToTableReq struct {
	Patient        eos.AccountName `json:"patient"`
	Account        eos.AccountName `json:"account"`
	IsDoctorValue  uint64          `json:"isDoctorValue"`
	IsEnabledValue uint64          `json:"isenabledValue"`
}

// GrantAccess adds `to` field in contract table
func (s *Storage) GrantAccess(to string) error {
	s.log.Debugf("Eos::grantAccess(%s) called", to)
	// Check if user is alreday on the table
	onTable, err := s.isOnTable(to)
	if err != nil {
		return err
	}
	if !onTable {
		s.log.Debugf("EOS::grantAccess(%s) User is not on table. Adding")
		action := &eos.Action{
			Account: eos.AN(s.config.EosContractAccount),
			Name:    eos.ActN("add"),
			Authorization: []eos.PermissionLevel{
				{eos.AN(s.config.EosAccount), eos.PermissionName("active")},
			},
			ActionData: eos.NewActionData(AddToTableReq{eos.AN(s.config.EosAccount), eos.AN(to), 1, 1}),
		}
		_, err := s.api.SignPushActions(action)
		s.log.Debugf("Added user to the table; %+v", err)
		return err
	}

	// Give access action
	action := &eos.Action{
		Account: eos.AN(s.config.EosContractAccount),
		Name:    eos.ActN("grantaccess"),
		Authorization: []eos.PermissionLevel{
			{eos.AN(s.config.EosAccount), eos.PermissionName("active")},
		},
		ActionData: eos.NewActionData(AccessReq{eos.AN(s.config.EosAccount), eos.AN(to)}),
	}
	_, err = s.api.SignPushActions(action)
	s.log.Debugf("Granted access to user; %+v", err)
	return err
}

func (s *Storage) isOnTable(user string) (bool, error) {
	ls, err := s.listAccountFromTable(s.config.EosAccount, false)
	if err != nil {
		return false, err
	}

	for _, v := range ls {
		if v == user {
			s.log.Debugf("User is on table")
			return true, nil
		}
	}
	return false, nil
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

func (s *Storage) ListConnected(to string) ([]string, error) {
	return s.listAccountFromTable(to, true)
}

type TableEntry struct {
	AccountName string `json:"account_name"`
	IsDoctor    int    `json:"isDoctor"`
	IsEnabled   int    `json:"isEnabled"`
}

func (s *Storage) listAccountFromTable(patient string, onlyConnected bool) ([]string, error) {
	s.log.Debugf("Eos::listAccountFromTable(%s, %v) called", patient, onlyConnected)

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
		if entry.IsEnabled == 1 || !onlyConnected {
			s.log.Debugf("Adding %s to list", entry.AccountName)
			ret = append(ret, entry.AccountName)
		}
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
	for _, a := range contract {
		_, err := s.api.SignPushActions(a)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateAccount creates account named `account` using key `key_str`
func (s *Storage) CreateAccount(account, key_str string) error {
	s.log.Debugf("Eos::createAccount(%s) called", account)
	key, err := ecc.NewPublicKey(key_str)

	actions := []*eos.Action{}

	// Create new account
	actions = append(actions, system.NewNewAccount(eos.AN(s.config.EosAccount), eos.AN(account), key))

	if s.config.EosRequiresRAM {
		actions = append(actions, system.NewBuyRAMBytes(eos.AccountName(s.config.EosAccount), eos.AccountName(account), 4096))
		actions = append(actions, system.NewDelegateBW(eos.AccountName(s.config.EosAccount), eos.AccountName(account), eos.NewEOSAsset(int64(10000)), eos.NewEOSAsset(int64(10000)), true))
	}

	_, err = s.api.SignPushActions(actions...)
	if err != nil {
		s.log.Debugf("Failed to create account; %+v", err)
		return err
	}

	return nil
}

// NewKey create new private key
func (s *Storage) NewKey() error {
	key, err := ecc.NewRandomPrivateKey()
	if err != nil {
		return err
	}
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
		s.log.Debugf("Error checking account: %v", err)
		return false
	}
	return true
}

// func (s *Storage) verifyTransaction(id string) (bool, error) {
// 	transaction, err := s.api.GetTransaction(id)
// 	if err != nil {
// 		return false, err
// 	}
// }
