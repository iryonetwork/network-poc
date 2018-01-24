package eth

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/contract"
	"github.com/iryonetwork/network-poc/logger"
)

type Storage struct {
	config  *config.Config
	client  *ethclient.Client
	auth    *bind.TransactOpts
	session *contract.PoCSession
	log     *logger.Log
}

func New(cfg *config.Config, log *logger.Log) (*Storage, error) {
	client, err := ethclient.Dial(cfg.EthAddr)
	if err != nil {
		return nil, err
	}
	auth := bind.NewKeyedTransactor(&cfg.EthPrivate)

	s := &Storage{config: cfg, client: client, auth: auth, log: log}

	return s, nil
}

func (s *Storage) GrantAccess(to string) error {
	s.log.Debugf("Eth::grantAccess(%s) called", to)
	_, err := s.session.GrantAccess(common.HexToAddress(to))
	return err
}

func (s *Storage) RevokeAccess(from string) error {
	s.log.Debugf("Eth::revokeAccess(%s) called", from)
	_, err := s.session.RevokeAccess(common.HexToAddress(from))
	return err
}

func (s *Storage) AccessGranted(from, to string) (bool, error) {
	s.log.Debugf("Eth::AccessGranted(%s, %s) called", from, to)
	if from == to {
		return true, nil
	}

	b, err := s.session.AccessGranted(common.HexToAddress(from), common.HexToAddress(to))
	s.log.Debugf("Eth::AccessGranted will return %v, %v\n%+v", b, err, s.session)
	return true, nil
}

func (s *Storage) DeployContract() error {
	s.log.Debugf("Eth::deployContract() called")

	if s.config.EthContractAddr != "" {
		return nil
	}

	address, _, _, err := contract.DeployPoC(s.auth, s.client)
	if err != nil {
		return fmt.Errorf("Failed to deploy new token contract: %v", err)
	}
	s.config.EthContractAddr = address.String()

	s.log.Debugf("Successfully deployed contract to address %s", address.String())

	return s.SetupSession()
}

func (s *Storage) SetupSession() error {
	s.log.Debugf("Setting up eth session for address %s", s.config.EthContractAddr)

	addr := common.HexToAddress(s.config.EthContractAddr)

	poc, err := contract.NewPoC(addr, s.client)
	if err != nil {
		return fmt.Errorf("Failed to initialize PoC contract; %v", err)
	}

	sender := crypto.PubkeyToAddress(s.config.EthPrivate.PublicKey)

	s.session = &contract.PoCSession{
		Contract: poc,
		CallOpts: bind.CallOpts{
			Pending: true,
			From:    sender,
		},
		TransactOpts: *s.auth,
	}

	return nil
}
