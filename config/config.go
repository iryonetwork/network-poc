package config

import (
	"github.com/caarlos0/env"
)

type Config struct {
	IryoAddr       string `env:"IRYO_ADDR" envDefault:"iryo:80"`
	EthPublic      string `env:"ETH_PUBLIC"`
	EthPrivate     string `env:"ETH_PRIVATE"`
	EncryptionKeys map[string][]byte
	Connections    []string
	Tokens         map[string]string
}

func New() (*Config, error) {
	cfg := &Config{}
	return cfg, env.Parse(cfg)
}
