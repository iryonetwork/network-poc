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
- [`doctor1`](http://localhost:9003) and [`doctor2`](http://localhost:9003) that is able to:
  - receive new keys,
  - read and write to patient's EHR.

Due to time constrains we had to skip using any actual medical data and more proper flows. Focus of this project was to setup a platform, that allows a secure and transparent sharing of data in which we as platform provider don't have access to patient data (only provide zero-knowledge storage).

## Setup

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

# prapare the repository (this will initialize and start the testnet, create accounts master, iryo and iryo.token)
# master is used to create both iryo accounts. iryo has iryo contract loaded. iryo.token has eosio.token contract loaded
make

# reset testnet 
make clear init

# (re)start api, patients and doctors 
make up

# check logs
make logs

# boot up the browsers
open http://localhost:9001 #patient1
open http://localhost:9002 #patient2
open http://localhost:9003 #doctor1
open http://localhost:9004 #doctor2
```

## API
### WS
/ws
all requests are json websocket messages of type `websocket.BinaryMessage`  

When connecting to websocket endpoint send the token in `token` cookie field.
After connection is authorized a message will be sent back saying `Authorized`
```
Notify that access was granted
IN:
{
    "Name":"NotifyGranted"
    "Fields":{
        "to":"user recieving access"
    }
}

OUT:
{
    "Name":"NotifyGranted"
    "Fields":{
        "from":"Sender"
    }
}
```
```
Request Key
doctor sends request to patient using
IN:
{
    "Name":"RequestKey",
    "Fields":{
        "key":"RSA public key",
        "signature":"EOS's signature of sha256 hash of RSA public key"
        "eoskey":"eoskey used to sign the hash",
        "to":"Account name"
    }
}

OUT:
{
    "Name":"RequestKey",
    "Fields":{
        "key":"RSA public key",
        "signature":"EOS's signature of sha256 hash of RSA public key"
        "eoskey":"eoskey used to sign the hash",
        "from":"sender of request"
    }
}
```
```
Send Key
IN:
{
    "Name":"SendKey",
    "Fields":{
        "key":"encrypted ehr signing key",
        "to":"account which made RequestKey request"
    }
}

OUT:
{
    "Name":"ImportKey",
    "Fields":{
        "key":"encrypted ehr signing key"
        "from":"sender of request"
    }
}

```
```
Reencrypt
after patient reencrypts the data usign new key
IN:
{
    "Name":"Reencrypt"
}

OUT: - sent to all connected doctors
{
    "Name":"Reencrypt"
    "Fields":{
        "from":"sender of in request"
    }
}
```

```
Revoke Key
IN:
{
    "Name":"RevokeKey",
    "Fields":{
        "to":"account which's key must be revoked"
    }
}

OUT:
{
    "Name":"RevokeKey",
    "Fields":{
        "from":"sender of request"
    }
}

```

### Login
POST /login
```
IN
hash: "random hash"
sign: "signature of hash made with key"
key: "key"
account: "acount_name" optional

OUT:
{
    token: uuid    
    validUntil: unix format
}
```
if account is not sent in the only endpoint accessable with token is create new account

### Create new account
POST /account
```json
In: "name": "User's name"
Out:
{"account":"account.iryo"}
```
### Upload
POST /<data_owner>
```json
In: "Content-type": multipart/form-data

    "key": EOS_Public_Key_used_to_sign_data,
    "sign": Signature_of_data's_sha256_hash,
    "data": file,

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

### Reupload
PUT /<data_owner>/<file_id>
```json
In: "Content-type": multipart/form-data

    "key": EOS_Public_Key_used_to_sign_data,
    "sign": Signature_of_data's_sha256_hash,
    "data": file,

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

### Get name
GET /<account_name>/id
```
{
    "name":"Janez Primer"
}
OR
{
    "error":"error goes here"
}
```