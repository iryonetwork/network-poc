package eth

import (
	"github.com/iryonetwork/network-poc/config"
)

type Storage struct {
	config *config.Config
}

func New(cfg *config.Config) *Storage {
	return &Storage{config: cfg}
}

func (s *Storage) GrantAccess(to string) error {
	return nil
}

func (s *Storage) RevokeAccess(from string) error {
	return nil
}

func (s *Storage) AccessGranted(from, to string) (bool, error) {
	return true, nil
}
