package config

import (
	"crypto/rsa"

	"github.com/caarlos0/env"
	"github.com/eoscanada/eos-go/ecc"
)

type Config struct {
	IryoAddr           string `env:"IRYO_ADDR" envDefault:"localhost:8000"`
	EosAPI             string `env:"EOS_API" envDefault:"http://localhost:8888"`
	EosPrivate         string `env:"EOS_PRIVATE"`
	EosAccount         string `env:"EOS_ACCOUNT"`
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
	StoragePath        string `env:"DATA_PATH" envDefault:"/data/ehr"` // Where to store uploaded ehr data
	EncryptionKeys     map[string][]byte
	RequestKeys        map[string]*rsa.PrivateKey
	Requested          map[string]*rsa.PublicKey
	Connections        []string
	GrantedWithoutKeys []string
}

func New() (*Config, error) {
	cfg := &Config{
		Requested:          make(map[string]*rsa.PublicKey),
		RequestKeys:        make(map[string]*rsa.PrivateKey),
		EncryptionKeys:     make(map[string][]byte),
		Connections:        []string{},
		GrantedWithoutKeys: []string{},
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
