# network-poc
Data exchange PoC developed in https://futurehack.io hackaton.

## Basic idea

In this project we would like to define and implement a system that allows user to own their healthcare data (EHR) and transparently manage who can have access to data.

## Main guidelines

* All data is encrypted and only visible to actors user shared the key with,
* all access grants and revocations are stored on a blockchain,
* user has a copy of the the data on own device, another (encrypted) copy is stored in the cloud or another globally distributed storage,
* keys to access to health record is not shared with the provider.

## Requirements

* go (current version 1.9.3, https://golang.org/dl)
* docker (current version 17.12.0-ce, https://www.docker.com/community-edition#/download)
* docker-compose (current version 1.18.0, https://docs.docker.com/compose/install/)
* govendor (`go get -u github.com/kardianos/govendor`)
* make

### Requirements for rebuilding the smart contract

- solc
- abigen (`go install github.com/ethereum/go-ethereum/cmd/abigen`)

## Setup

```bash
# clone the repository
go get -u github.com/iryonetwork/network-poc

# go to the folder (assuming $GOPATH is set to ~)
cd ~/go/src/github.com/iryonetwork/network-poc

# prapare the repository
make

# run tests
make test

# start it up
make up

# TBD ...

# rebuild the smart contract (requires abigen and solc)
make buildContract
```

