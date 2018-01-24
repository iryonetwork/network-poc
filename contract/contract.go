// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contract

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// PoCABI is the input ABI used to generate the binding from.
const PoCABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"}],\"name\":\"grantAccess\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"}],\"name\":\"accessGranted\",\"outputs\":[{\"name\":\"is_connected\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"}],\"name\":\"revokeAccess\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// PoCBin is the compiled bytecode used for deploying new contracts.
const PoCBin = `0x6060604052341561000f57600080fd5b6101928061001e6000396000f3006060604052600436106100565763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416630ae5e739811461005b578063705e06e21461008e57806385e68531146100b3575b600080fd5b341561006657600080fd5b61007a600160a060020a03600435166100d2565b604051901515815260200160405180910390f35b341561009957600080fd5b61007a600160a060020a0360043581169060243516610108565b34156100be57600080fd5b61007a600160a060020a0360043516610134565b600160a060020a03338116600090815260208181526040808320938516835292905220805460ff19166001908117909155919050565b600160a060020a0391821660009081526020818152604080832093909416825291909152205460ff1690565b600160a060020a03338116600090815260208181526040808320938516835292905220805460ff1916905560019190505600a165627a7a72305820b2cc37e5a4b000ecb3f439f2e16b183ebf7eb20c4564a85bb0ed47ff5d668c690029`

// DeployPoC deploys a new Ethereum contract, binding an instance of PoC to it.
func DeployPoC(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *PoC, error) {
	parsed, err := abi.JSON(strings.NewReader(PoCABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(PoCBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &PoC{PoCCaller: PoCCaller{contract: contract}, PoCTransactor: PoCTransactor{contract: contract}}, nil
}

// PoC is an auto generated Go binding around an Ethereum contract.
type PoC struct {
	PoCCaller     // Read-only binding to the contract
	PoCTransactor // Write-only binding to the contract
}

// PoCCaller is an auto generated read-only Go binding around an Ethereum contract.
type PoCCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PoCTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PoCTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PoCSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PoCSession struct {
	Contract     *PoC              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PoCCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PoCCallerSession struct {
	Contract *PoCCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// PoCTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PoCTransactorSession struct {
	Contract     *PoCTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PoCRaw is an auto generated low-level Go binding around an Ethereum contract.
type PoCRaw struct {
	Contract *PoC // Generic contract binding to access the raw methods on
}

// PoCCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PoCCallerRaw struct {
	Contract *PoCCaller // Generic read-only contract binding to access the raw methods on
}

// PoCTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PoCTransactorRaw struct {
	Contract *PoCTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPoC creates a new instance of PoC, bound to a specific deployed contract.
func NewPoC(address common.Address, backend bind.ContractBackend) (*PoC, error) {
	contract, err := bindPoC(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PoC{PoCCaller: PoCCaller{contract: contract}, PoCTransactor: PoCTransactor{contract: contract}}, nil
}

// NewPoCCaller creates a new read-only instance of PoC, bound to a specific deployed contract.
func NewPoCCaller(address common.Address, caller bind.ContractCaller) (*PoCCaller, error) {
	contract, err := bindPoC(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &PoCCaller{contract: contract}, nil
}

// NewPoCTransactor creates a new write-only instance of PoC, bound to a specific deployed contract.
func NewPoCTransactor(address common.Address, transactor bind.ContractTransactor) (*PoCTransactor, error) {
	contract, err := bindPoC(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &PoCTransactor{contract: contract}, nil
}

// bindPoC binds a generic wrapper to an already deployed contract.
func bindPoC(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(PoCABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PoC *PoCRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _PoC.Contract.PoCCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PoC *PoCRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PoC.Contract.PoCTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PoC *PoCRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PoC.Contract.PoCTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PoC *PoCCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _PoC.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PoC *PoCTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PoC.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PoC *PoCTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PoC.Contract.contract.Transact(opts, method, params...)
}

// AccessGranted is a free data retrieval call binding the contract method 0x705e06e2.
//
// Solidity: function accessGranted(_from address, _to address) constant returns(is_connected bool)
func (_PoC *PoCCaller) AccessGranted(opts *bind.CallOpts, _from common.Address, _to common.Address) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _PoC.contract.Call(opts, out, "accessGranted", _from, _to)
	return *ret0, err
}

// AccessGranted is a free data retrieval call binding the contract method 0x705e06e2.
//
// Solidity: function accessGranted(_from address, _to address) constant returns(is_connected bool)
func (_PoC *PoCSession) AccessGranted(_from common.Address, _to common.Address) (bool, error) {
	return _PoC.Contract.AccessGranted(&_PoC.CallOpts, _from, _to)
}

// AccessGranted is a free data retrieval call binding the contract method 0x705e06e2.
//
// Solidity: function accessGranted(_from address, _to address) constant returns(is_connected bool)
func (_PoC *PoCCallerSession) AccessGranted(_from common.Address, _to common.Address) (bool, error) {
	return _PoC.Contract.AccessGranted(&_PoC.CallOpts, _from, _to)
}

// GrantAccess is a paid mutator transaction binding the contract method 0x0ae5e739.
//
// Solidity: function grantAccess(_to address) returns(success bool)
func (_PoC *PoCTransactor) GrantAccess(opts *bind.TransactOpts, _to common.Address) (*types.Transaction, error) {
	return _PoC.contract.Transact(opts, "grantAccess", _to)
}

// GrantAccess is a paid mutator transaction binding the contract method 0x0ae5e739.
//
// Solidity: function grantAccess(_to address) returns(success bool)
func (_PoC *PoCSession) GrantAccess(_to common.Address) (*types.Transaction, error) {
	return _PoC.Contract.GrantAccess(&_PoC.TransactOpts, _to)
}

// GrantAccess is a paid mutator transaction binding the contract method 0x0ae5e739.
//
// Solidity: function grantAccess(_to address) returns(success bool)
func (_PoC *PoCTransactorSession) GrantAccess(_to common.Address) (*types.Transaction, error) {
	return _PoC.Contract.GrantAccess(&_PoC.TransactOpts, _to)
}

// RevokeAccess is a paid mutator transaction binding the contract method 0x85e68531.
//
// Solidity: function revokeAccess(_from address) returns(success bool)
func (_PoC *PoCTransactor) RevokeAccess(opts *bind.TransactOpts, _from common.Address) (*types.Transaction, error) {
	return _PoC.contract.Transact(opts, "revokeAccess", _from)
}

// RevokeAccess is a paid mutator transaction binding the contract method 0x85e68531.
//
// Solidity: function revokeAccess(_from address) returns(success bool)
func (_PoC *PoCSession) RevokeAccess(_from common.Address) (*types.Transaction, error) {
	return _PoC.Contract.RevokeAccess(&_PoC.TransactOpts, _from)
}

// RevokeAccess is a paid mutator transaction binding the contract method 0x85e68531.
//
// Solidity: function revokeAccess(_from address) returns(success bool)
func (_PoC *PoCTransactorSession) RevokeAccess(_from common.Address) (*types.Transaction, error) {
	return _PoC.Contract.RevokeAccess(&_PoC.TransactOpts, _from)
}
