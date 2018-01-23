package config

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"reflect"

	"github.com/caarlos0/env"
	"github.com/ethereum/go-ethereum/crypto"
)

type Config struct {
	IryoAddr       string           `env:"IRYO_ADDR" envDefault:"iryo:80"`
	EthPublic      string           `env:"ETH_PUBLIC"`
	EthPrivate     ecdsa.PrivateKey `env:"ETH_PRIVATE,required"`
	EncryptionKeys map[string][]byte
	Connections    []string
	Tokens         map[string]string
}

func New() (*Config, error) {
	cfg := &Config{}
	parsers := map[reflect.Type]env.ParserFunc{
		reflect.TypeOf(cfg.EthPrivate): parseEthPrivateKey,
	}
	return cfg, env.ParseWithFuncs(cfg, parsers)
}

func parseEthPrivateKey(v string) (interface{}, error) {
	bd, err := hex.DecodeString(v)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode private key from HEX; %v", err)
	}
	k, err := crypto.ToECDSA(bd)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert to private key; %v", err)
	}

	return *k, nil
}

func (c *Config) GetEthPublicAddress() string {
	return crypto.PubkeyToAddress(c.EthPrivate.PublicKey).String()
}
