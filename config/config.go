package config

import (
	"crypto/rsa"

	"github.com/caarlos0/env"
	"github.com/eoscanada/eos-go/ecc"
)

type Config struct {
	IryoAddr        string `env:"IRYO_ADDR" envDefault:"localhost:8000"`
	EosAPI          string `env:"EOS_API" envDefault:"http://localhost:8888"`
	EosPrivate      string `env:"EOS_PRIVATE,required"`
	EosAccount      string `env:"EOS_ACCOUNT"`
	EosContractName string `env:"EOS_CONTRACT_NAME"`
	EosTokenAccount string `env:"EOS_TOKEN_ACCOUNT"`
	EosTokenName    string `env:"EOS_TOKEN_NAME"`
	ClientType      string `env:"CLIENT_TYPE" envDefault:"Patient"`
	ClientAddr      string `env:"CLIENT_ADDR" envDefault:"localhost:9000"`
	Debug           bool   `env:"DEBUG" envDefault:"1"`
	EncryptionKeys  map[string][]byte
	RequestKeys     map[string]*rsa.PrivateKey
	Requested       map[string]*rsa.PublicKey
	Connections     []string
}

func New() (*Config, error) {
	cfg := &Config{
		Requested:      make(map[string]*rsa.PublicKey),
		RequestKeys:    make(map[string]*rsa.PrivateKey),
		EncryptionKeys: make(map[string][]byte),
		Connections:    []string{},
	}
	return cfg, env.Parse(cfg)
}

func (c *Config) GetEosPublicKey() string {
	key, err := ecc.NewPrivateKey(c.EosPrivate)
	if err != nil {
		panic(err)
	}
	return key.PublicKey().String()
}
