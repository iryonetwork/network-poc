# network-poc
Data exchange PoC developed in https://futurehack.io hackaton.

In this project we would like to define and implement a system that allows user to own their healthcare data (EHR) and transparently manage who can have access to data.

## Main guidelines

* All data is encrypted and only visible to actors user shared the key with,
* all access grants and revocations are stored on a blockchain,
* user has a copy of the the data on own device, another (encrypted) copy is stored in the cloud or another globally distributed storage,
* keys to access to health record is not shared with the provider.

## About this PoC

In Iryo we planning to build an EHR system that will allow patients to take control of the data. They will be able to carry their EHR around with them as they will have an option to have a copy of it stored on their smart phone. As owners of data they will also have full control of who is able to read and modify data. By default any ungranted access to the data will not be possible. As can't afford to only have one copy of the patient's EHR, an encrypted copy will be uploaded to an external storage and only the patient will posses the key to decrypt this copy.

This PoC explores how we could utilize blockchain to transparently store and manage access control lists that will then control data flow and connections between different actors in the ecosystem. 

The setup consists of following components:

- prepopulated `geth` acting as a local testnet
  - smart contract keeping a list who is connected with who.
- `iryo` represents a cloud component that:
  -  allows anonymous entities to share data between each other
  -  recreates / deploys the smart contract on start
- [`patient1`](http://localhost:9001) and [`patient2`](http://localhost:9002) who represent a patient that:
  - is the initial owner of it's own EHR,
  - has an option to create a new connection with a doctor,
  - is the only entity that writes to blockchain
  - can send the an encrypted key to the doctor to enable the doctor to read and write to their EHR.
- [`doctor`](http://localhost:9003) that is able to:
  - receive new keys,
  - read and write to patient's EHR.
- [`myetherwallet`](http://localhost:8080) useful for inspecting current state of accounts and the contract.

Due to time constrains we had to skip using any actual medical data and more proper flows. Focus of this project was to setup a platform, that allows a secure and transparent sharing of data in which we as platform provider don't have access to patient data (only provide zero-knowledge storage).

### Smart contract

We opted to for now use a simple contract for managing ACL to be able to focus more on other components that have to be able to write to and read state from blockchain. For these reasons the smart contract represents a simple map that keeps track of connections between different entities.

```solidity
// ./contract/contract.sol

pragma solidity ^0.4.0;
contract PoC {
    mapping (address => mapping (address => bool)) connected;

    /* connects the msg.sender and to address as connected  */
    function grantAccess(address _to) public returns (bool success) {
        connected[msg.sender][_to] =true;          // Set the connection
        return true;
    }

    /* connects the msg.sender and to address as connected  */
    function revokeAccess(address _from) public returns (bool success) {
        connected[msg.sender][_from] =false;          // Remove the connection
        return true;
    }

    /* Get the amount of remaining tokens to spend */
    function accessGranted(address _from, address _to) public constant returns (bool is_connected) {
        return connected[_from][_to];
    }
}

```



## Setup

### Requirements

* go (current version 1.9.3, https://golang.org/dl)
* docker (current version 17.12.0-ce, https://www.docker.com/community-edition#/download)
* docker-compose (current version 1.18.0, https://docs.docker.com/compose/install/)
* make

#### Requirements for rebuilding the smart contract

- solc
- abigen (`go install github.com/ethereum/go-ethereum/cmd/abigen`)

### Setup the dev environment & run the code

The development environment is setup using docker-compose that allows us to have all dependencies running locally in an isolated environment.

```bash
# clone the repository
git clone git@github.com:iryonetwork/network-poc.git

# go to the folder
cd network-poc

# prapare the repository (this will initialize and start the testnet)
make

# check logs (it takes a while to prepare DAG, you can proceed once geth node starts mining)
make logs

# start rest of the services
make up

# once DAG is done, fire up browsers
open http://localhost:9001 # patient1
open http://localhost:9002 # patient2
open http://localhost:9003 # doctor
open http://localhost:8080 # local myetherwallet

# connect patient and doctor by copying doctor's address to input field on patient's website and click connect
```

## EOS Setup

### Requirements

* go (current version 1.9.3, https://golang.org/dl)
* docker (current version 17.12.0-ce, https://www.docker.com/community-edition#/download)
* docker-compose (current version 1.18.0, https://docs.docker.com/compose/install/)
* make

### Setup the dev environment & run the code

The development environment is setup using docker-compose that allows us to have all dependencies running locally in an isolated environment.

```bash
# clone the repository
git clone git@github.com:iryonetwork/network-poc.git

# go to the folder
cd network-poc

# prapare the repository (this will initialize and start the testnet, create accounts master, iryo and iryo.token, and create the patients)
# master is used to create both iryo accounts. iryo has iryo contract loaded. iryo.token has eosio.token contract loaded
make apiinit
make apiup
# check logs
make logs

# boot up the browsers
open http://localhost:9001 #patient1
open http://localhost:9002 #patient2
open http://localhost:9003 #doctor
```

## API
### Login
POST /login
```
IN
hash: "random hash"
sign: "signature of hash made with key"
key: "key"
account: "acount_name" optional

OUT:
token in uuid format
```
if account is not sent in the only endpoint accessable with token is create new account

### Create new account
GET /account/<Eos_Key>
```json
{"account":"account.iryo"}
```
### Upload
POST /<data_owner>
```json
In: "Content-type": multipart/form-data

    "key": EOS_Public_Key_used_to_sign_data,
    "sign": Signature_of_data's_sha256_hash,
    "data": file,
    "account": Name_of_account_signing,


Out:
{
    "fileID": "UUID",
    "createdAt": "YYYY-MM-DDTHH:MM:SS.MsMsMsZ"
}
OR
{
    "error": "error"
}
```

### List 
GET /<account_name>
```json
{
    "files":[
        {
            "fileID": "UUID1",
            "createdAt": "YYYY-MM-DDTHH:MM:SS.MsMsMsZ"
        },
        {
            "fileID": "UUID2",
            "createdAt": "YYYY-MM-DDTHH:MM:SS.MsMsMsZ"
        }
    ]
}
OR
{
    "error": "error"
}
```
### Download
GET /<account_name>/<file_id>
```
File's contents

OR

"ERROR: error"
```

## Lessons learned

#### github.com/ethereum/go-ethereum package

1. Does not allow cross-compilation (we work on macs) hence which is why we ended up with quite an awkward and slow go execution process. Subpackage `github.com/ethereum/go-ethereum/crypto/secp256k1` include C source files. Flags in this package running following from a mac console: `GOOS=linux go build ./cmd/iryo`. 

2. The same source files in the `crypto/secp256k1` package prevented us from using a vendor folder as the C source files are stripped out by `govendor`. We did not check if other dependency management tools (like `dep`) also have fail due to this issue.

3. There is a lot of development going on this package and the interfaces of public methods are constantly changing.

   Following error occurred two hours before the end of the hackaton:

   ```
   # github.com/iryonetwork/network-poc/contract
   ../../contract/contract.go:120:30: not enough arguments in call to bind.NewBoundContract
   have (common.Address, abi.ABI, bind.ContractCaller, bind.ContractTransactor)
   want (common.Address, abi.ABI, bind.ContractCaller, bind.ContractTransactor, bind.ContractFilterer)
   ```

## Notes

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

(5) 05342bde6dc4bc6f848dc1c79ea684daf4d496a317de1aa54c1570134a803f11 [free]
    0xC7b1447EE0BC47894e4bf67E6b31868463688cD3
```

* All accounts are have initial balance of 200 ETH.
* Keystore / JSON files are located in `./services/geth/keystore`. Password was set to `test12345`.


### Configuring my etherwallet

1. Select "Add custom network / Node" in the network dropdown in the top right corner
2. Enter following details:
   - name: `geth`
   - URL: `http://127.0.0.1`
   - Port: `8545`
   - Additionally you check "Support EIP-155" and enter `456719` under ChainID

This has to be done on the locally running instance of myetherwallet as the chain is not served with TLS.

### Contract ABI / JSON Interface

When testing the state of the contract from [myetherwallet](http://localhost:8080) following code has to be provided. Current address of the contract can be seen on any of the actor websites ([patient1](http://localhost:9001), [patient2](http://localhost:9002) or the [doctor](http://localhost:9003)) .

```json
[{"constant":false,"inputs":[{"name":"_to","type":"address"}],"name":"grantAccess","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"}],"name":"accessGranted","outputs":[{"name":"is_connected","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"}],"name":"revokeAccess","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]
```

