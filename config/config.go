package config

import (
	"crypto/rsa"

	"github.com/caarlos0/env"
	"github.com/eoscanada/eos-go/ecc"
	"github.com/iryonetwork/network-poc/openEHR"
)

type Config struct {
	IryoAddr           string `env:"IRYO_ADDR" envDefault:"localhost:8000"`
	EosAPI             string `env:"EOS_API" envDefault:"http://localhost:8888"`
	EosPrivate         string `env:"EOS_PRIVATE"`
	EosAccount         string `env:"EOS_ACCOUNT"`
	EosContractAccount string `env:"EOS_CONTRACT_ACCOUNT"`
	EosContractName    string `env:"EOS_CONTRACT_NAME"`
	EosTokenAccount    string `env:"EOS_TOKEN_ACCOUNT"`
	EosTokenName       string `env:"EOS_TOKEN_NAME"`
	EosAccountFormat   string `env:"EOS_ACCOUNT_FORMAT" envDefault:"[a-z1-5]{7}\\.iryo"`
	EosRequiresRAM     bool   `env:"EOS_REQUIRES_RAM" envDefault:"0"`
	EosMinimumRAM      int    `env:"EOS_MINIMUM_RAM" envDefault:"10000"`
	EosStakeRAM        int    `env:"EOS_STAKE_RAM" envDefault:"4096"`
	ClientType         string `env:"CLIENT_TYPE" envDefault:"Patient"`
	ClientAddr         string `env:"CLIENT_ADDR" envDefault:"localhost:9000"`
	Debug              bool   `env:"DEBUG" envDefault:"1"`
	StoragePath        string `env:"DATA_PATH" envDefault:"/data"`
	Token              string
	Connected          bool
	Subscribed         bool
	IsDoctor           bool
	PersonalData       *openEHR.PersonalData
	EncryptionKeys     map[string][]byte
	RSAKey             *rsa.PrivateKey
	Connections        Connections
	Directory          map[string]string
}

type Connections struct {
	WithKey    []string                  // Access is written on the blockchain and we have the key
	WithoutKey []string                  // Access has been written on the blockchain, but we do not have the key
	GrantedTo  []string                  // We have granted the access to our data to these users. Its on them to make key request
	Requested  map[string]*rsa.PublicKey // They are not connected to us, but request for key has been made
}

func New() (*Config, error) {
	cfg := &Config{
		Connected:      false,
		Subscribed:     false,
		PersonalData:   &openEHR.PersonalData{},
		EncryptionKeys: make(map[string][]byte),
		Connections: Connections{
			WithKey:    []string{},
			WithoutKey: []string{},
			GrantedTo:  []string{},
			Requested:  make(map[string]*rsa.PublicKey),
		},
		Directory: make(map[string]string),
		IsDoctor:  false,
	}
	err := env.Parse(cfg)

	if cfg.ClientType == "Doctor" {
		cfg.IsDoctor = true
	}

	return cfg, err
}

func (c *Config) GetEosPublicKey() string {
	key, err := ecc.NewPrivateKey(c.EosPrivate)
	if err != nil {
		panic(err)
	}
	return key.PublicKey().String()
}

func (c *Config) GetNames(list []string) map[string]string {
	out := make(map[string]string)
	for _, username := range list {
		if _, ok := c.Directory[username]; !ok {
			out[username] = username
			continue
		}
		out[username] = c.Directory[username]
	}
	return out
}
