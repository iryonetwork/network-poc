# IRYO EOS CONTRACT

## Table
Each patient has its own table with data:
```
{
    "account_name" : "account1iryo"
    "account_name" : "account2iryo"
    "account_name" : "account3iryo"
    "account_name" : "account4iryo"

}
```
## Functions

`grantaccess(patient, account)` 
- add `account` to table
- signed by `patient`

`revokeaccess(patient, account)`
- remove `account` from table
- signed by `patient`

`revokeaccess2(patient, account)`
- remove `account` from table
- signed by `account`

## Get the table
`cleos -u $EOS_API get table <contract_account> <patient> person`

## Add to table
`cleos -u $EOS_API push action <contract_account> grantaccess '[<patient>, <account>]' -p <patient>@active`

## Remove from table
As patient  
`cleos -u $EOS_API push action <contract_account> revokeaccess '[<patient>, <account>]' -p <patient>@active`  

As doctor  
`cleos -u $EOS_API push action <contract_account> revokeaccess2 '[<patient>, <account>]' -p <account>@active`