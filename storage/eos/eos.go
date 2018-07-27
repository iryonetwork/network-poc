package eos

import (
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
	log.Debugf("Adding API from %s", cfg.EosAPI)
	node := eos.New(cfg.EosAPI)
	node.SetSigner(eos.NewKeyBag())
	s := &Storage{config: cfg, api: node}
	s.api = node
	return s, nil
}

// AccessReq contains fields needed in sending access-contract related actions
type AccessReq struct {
	From eos.AccountName `json:"from"`
	To   eos.AccountName `json:"to"`
}

// GrantAccess adds `to` field in contract table
func (s *Storage) GrantAccess(to string) error {
	s.log.Printf("Eos::grantAccess(%s) called", to)
	// Give access action
	action := &eos.Action{
		Account: eos.AN(s.config.EosContractName),
		Name:    eos.ActN("give"),
		Authorization: []eos.PermissionLevel{
			{eos.AN(s.config.EosAccount), eos.PermissionName("active")},
		},
		ActionData: eos.NewActionData(AccessReq{From: eos.AN(s.config.EosAccount), To: eos.AN(to)}),
	}
	_, err := s.api.SignPushActions(action)
	return err
}

// RevokeAccess removes `to` field in contract table
func (s *Storage) RevokeAccess(to string) error {
	s.log.Printf("Eos::revokeAccess(%s) called", to)
	// Remove access action
	action := &eos.Action{
		Account: eos.AN(s.config.EosContractName),
		Name:    eos.ActN("premove"),
		Authorization: []eos.PermissionLevel{
			{eos.AN(s.config.EosAccount), eos.PermissionName("active")},
		},
		ActionData: eos.NewActionData(AccessReq{From: eos.AN(s.config.EosAccount), To: eos.AN(to)}),
	}
	_, err := s.api.SignPushActions(action)
	return err
}

// AccessGranted checks if connection between `from` and `to` is establisehd
// Due to uint32 limitations this functions allows connection for up to 4294967295 doctors to a single client
func (s *Storage) AccessGranted(from, to string) (bool, error) {
	s.log.Printf("Eos::accessGranted(%s, %s) called", from, to)
	// Get the table
	r, err := s.api.GetTableRows(eos.GetTableRowsRequest{JSON: true, Scope: to, Code: s.config.EosContractName, Table: "status", Limit: math.MaxUint32})

	s.log.Debugf("Got table: %s", r)
	a := make([]map[string]string, 0)
	r.JSONToStructs(&a)
	// Check if `to` has its field in the table
	b := false
	for _, st := range a {
		for _, n := range st {
			if n == to {
				b = true
			}
		}
	}
	return b, err
}

// DeployContract pushes contract located in contract/eos to blockchain under name specified in config
func (s *Storage) DeployContract() error {
	s.log.Printf("Eos::deployContract() called")

	if s.config.EosContractName == "" {
		return fmt.Errorf("No config.EosContractName specified, unable to deploy contract")
	}
	if s.config.EosTokenAccount == "" {
		return fmt.Errorf("No config.EosTokenAccount specified, unable to createa account to deploy contract to")
	}

	err := s.pushContract(s.config.EosAccount, s.config.EosContractName)
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
	s.CreateAccount(n)

	// Get newcontract actions
	contract, err := system.NewSetContract(eos.AN(n), "../../contract/eos/"+cn+".wasm", "../../contract/eos/"+cn+".abi")
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
func (s *Storage) SetupSession() error {
	err := s.CreateAccount(s.config.EosAccount)
	return err
}

// CreateAccount creates account using key specified in config.EosPrivate.
// The key is imported, then account is created
func (s *Storage) CreateAccount(account string) error {
	s.log.Printf("Eos::createAccount(%s) called", account)

	key, err := ecc.NewPrivateKey(s.config.EosPrivate)
	if err != nil {
		return err
	}
	s.api.Signer.ImportPrivateKey(s.config.EosPrivate)
	// Create new account
	action := system.NewNewAccount(eos.AN("master"), eos.AN(account), key.PublicKey())
	res, err := s.api.SignPushActions(action)
	s.log.Printf("newAccount action response: %#v", res)
	if err != nil {
		s.log.Printf("%s", err)
		return err
	}
	return nil
}
