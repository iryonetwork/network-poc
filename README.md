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

### Available accounts (private + public)

```
(0) 896772721092967eb44d6c72e3ebdd97babbada1c29eace2d8c1dd7bd1f9ee99 [IRYO]
    0xF5CB47467d3da58e456Bd62Fc1B3e4FB74f8E64C
    
(1) 30fa8b6d85a5aa024cd3286907c96318b27217546d28232373f94cc76bb74b76 [PATIENT1]
    0x740f35190F298c0675d100aFaf9dEe2f75Fa7BE2
    
(2) d43cd3f2c6e210fc263e6cf6ad5db8cd9eb746e206df43f5f07eaf2d2e72bac0 [PATIENT2]
    0xf69aEB8E48C98771e48e38C5709C6f7EC3afC77b
    
(3) 188a9155c8588b67e3083834e970e8fcb40ad9fd069eae0f247d16cb8c0a8173 [DOCTOR]
    0x691447E125D7ab2C1b12D10AA5e97D2696603714
    
(4) d6579f267149ea4a7c0787c21b72727683da8bc83f73c94bbc8c8378bf351467 [MINER]
    0xe93247262a6737B0af86c647c665513e8323AD96

(5) 05342bde6dc4bc6f848dc1c79ea684daf4d496a317de1aa54c1570134a803f11 [CONTRACT DEPLOY]
    0xC7b1447EE0BC47894e4bf67E6b31868463688cD3
```

* All accounts are have initial balance of 200 ETH.
* Keystore / JSON files are located in `./services/geth/keystore`. Password was set to `test12345`.

