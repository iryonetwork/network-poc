package main

import (
	stdlog "log"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/state"
	"github.com/iryonetwork/network-poc/storage/eos"
)

func main() {
	config, err := config.New()
	if err != nil {
		stdlog.Fatalf("failed to get config: %v", err)
	}

	log := logger.New(config)

	state, err := state.New(config, log)
	if err != nil {
		log.Fatalf("failed to initialize state: %v", err)
	}
	defer state.Close()

	eos, err := eos.New(config, state, log)
	if err != nil {
		log.Fatalf("failed to setup eos storage; %v", err)
	}
	err = eos.DeployContract()
	if err != nil {
		log.Fatalf("failed to deploy the contract; %v", err)
	}
}
