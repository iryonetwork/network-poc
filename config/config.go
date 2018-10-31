package config

import (
	"github.com/caarlos0/env"
)

type Config struct {
	IryoAddr                     string `env:"IRYO_ADDR" envDefault:"localhost:8000"`
	EosAPI                       string `env:"EOS_API" envDefault:"http://localhost:8888"`
	EosPrivate                   string `env:"EOS_PRIVATE"`
	EosAccount                   string `env:"EOS_ACCOUNT"`
	EosContractAccount           string `env:"EOS_CONTRACT_ACCOUNT"`
	EosContractName              string `env:"EOS_CONTRACT_NAME"`
	EosTokenAccount              string `env:"EOS_TOKEN_ACCOUNT"`
	EosTokenName                 string `env:"EOS_TOKEN_NAME"`
	EosAccountFormat             string `env:"EOS_ACCOUNT_FORMAT" envDefault:"[a-z1-5]{7}\\.iryo"`
	EosRequiresRAM               bool   `env:"EOS_REQUIRES_RAM" envDefault:"0"`
	EosMinimumRAM                int    `env:"EOS_MINIMUM_RAM" envDefault:"10000"`
	EosStakeRAM                  int    `env:"EOS_STAKE_RAM" envDefault:"4096"`
	ClientType                   string `env:"CLIENT_TYPE" envDefault:"Patient"`
	ClientAddr                   string `env:"CLIENT_ADDR" envDefault:"localhost:9000"`
	Debug                        bool   `env:"DEBUG" envDefault:"1"`
	StoragePath                  string `env:"DATA_PATH" envDefault:"/data"`
	PersistentState              bool   `env:"PERSISTENT_STATE" envDefault:"0"`
	PersistentStateEncryptionKey string `env:"PERSISTENT_STATE_ENCRYPTION_KEY"`
	PersistentStateStoragePath   string `env:"PERSISTENT_STATE_STORAGE_PATH"`
}

func New() (*Config, error) {
	cfg := &Config{}
	err := env.Parse(cfg)

	return cfg, err
}
