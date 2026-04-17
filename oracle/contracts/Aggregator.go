// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contracts

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// AggregatorMetaData contains all meta data concerning the Aggregator contract.
var AggregatorMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_queryFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_oracleReward\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_modelCreatorReward\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_verifierAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_queueAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_managerAddress\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"jobId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"submitter\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"vectorLength\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"JobCompleted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"requestId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"ipfsCid\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"requester\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"payment\",\"type\":\"uint256\"}],\"name\":\"LogNewCustomerRequest\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"jobId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"ipfsCid\",\"type\":\"string\"}],\"name\":\"LogNewJobForOracles\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_requestId\",\"type\":\"uint256\"}],\"name\":\"approveJob\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"_oracle\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_jobId\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_vectorLength\",\"type\":\"uint256\"}],\"name\":\"distributeRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"filterThreshold\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"filterType\",\"outputs\":[{\"internalType\":\"enumIAggregator.FilterType\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getFilterPolicy\",\"outputs\":[{\"internalType\":\"enumIAggregator.FilterType\",\"name\":\"\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_jobId\",\"type\":\"uint256\"}],\"name\":\"getResult\",\"outputs\":[{\"internalType\":\"int128[]\",\"name\":\"\",\"type\":\"int128[]\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_jobId\",\"type\":\"uint256\"}],\"name\":\"isCompleted\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"manager\",\"outputs\":[{\"internalType\":\"contractIRoyaltyManager\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"modelCreatorReward\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"oracleReward\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"queryFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"queue\",\"outputs\":[{\"internalType\":\"contractIOracleQueue\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_ipfsCid\",\"type\":\"string\"}],\"name\":\"requestAttribution\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"enumIAggregator.FilterType\",\"name\":\"_filterType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"_threshold\",\"type\":\"uint256\"}],\"name\":\"setFilterPolicy\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"configDigest\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"seqNr\",\"type\":\"uint64\"},{\"internalType\":\"bytes\",\"name\":\"report\",\"type\":\"bytes\"},{\"internalType\":\"bytes32[]\",\"name\":\"rs\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32[]\",\"name\":\"ss\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32\",\"name\":\"rawVs\",\"type\":\"bytes32\"}],\"name\":\"transmit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"verifier\",\"outputs\":[{\"internalType\":\"contractIOracleVerifier\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// AggregatorABI is the input ABI used to generate the binding from.
// Deprecated: Use AggregatorMetaData.ABI instead.
var AggregatorABI = AggregatorMetaData.ABI

// Aggregator is an auto generated Go binding around an Ethereum contract.
type Aggregator struct {
	AggregatorCaller     // Read-only binding to the contract
	AggregatorTransactor // Write-only binding to the contract
	AggregatorFilterer   // Log filterer for contract events
}

// AggregatorCaller is an auto generated read-only Go binding around an Ethereum contract.
type AggregatorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AggregatorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AggregatorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AggregatorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AggregatorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AggregatorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AggregatorSession struct {
	Contract     *Aggregator       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AggregatorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AggregatorCallerSession struct {
	Contract *AggregatorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// AggregatorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AggregatorTransactorSession struct {
	Contract     *AggregatorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// AggregatorRaw is an auto generated low-level Go binding around an Ethereum contract.
type AggregatorRaw struct {
	Contract *Aggregator // Generic contract binding to access the raw methods on
}

// AggregatorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AggregatorCallerRaw struct {
	Contract *AggregatorCaller // Generic read-only contract binding to access the raw methods on
}

// AggregatorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AggregatorTransactorRaw struct {
	Contract *AggregatorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAggregator creates a new instance of Aggregator, bound to a specific deployed contract.
func NewAggregator(address common.Address, backend bind.ContractBackend) (*Aggregator, error) {
	contract, err := bindAggregator(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Aggregator{AggregatorCaller: AggregatorCaller{contract: contract}, AggregatorTransactor: AggregatorTransactor{contract: contract}, AggregatorFilterer: AggregatorFilterer{contract: contract}}, nil
}

// NewAggregatorCaller creates a new read-only instance of Aggregator, bound to a specific deployed contract.
func NewAggregatorCaller(address common.Address, caller bind.ContractCaller) (*AggregatorCaller, error) {
	contract, err := bindAggregator(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AggregatorCaller{contract: contract}, nil
}

// NewAggregatorTransactor creates a new write-only instance of Aggregator, bound to a specific deployed contract.
func NewAggregatorTransactor(address common.Address, transactor bind.ContractTransactor) (*AggregatorTransactor, error) {
	contract, err := bindAggregator(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AggregatorTransactor{contract: contract}, nil
}

// NewAggregatorFilterer creates a new log filterer instance of Aggregator, bound to a specific deployed contract.
func NewAggregatorFilterer(address common.Address, filterer bind.ContractFilterer) (*AggregatorFilterer, error) {
	contract, err := bindAggregator(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AggregatorFilterer{contract: contract}, nil
}

// bindAggregator binds a generic wrapper to an already deployed contract.
func bindAggregator(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AggregatorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Aggregator *AggregatorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Aggregator.Contract.AggregatorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Aggregator *AggregatorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Aggregator.Contract.AggregatorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Aggregator *AggregatorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Aggregator.Contract.AggregatorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Aggregator *AggregatorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Aggregator.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Aggregator *AggregatorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Aggregator.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Aggregator *AggregatorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Aggregator.Contract.contract.Transact(opts, method, params...)
}

// FilterThreshold is a free data retrieval call binding the contract method 0x717b90b4.
//
// Solidity: function filterThreshold() view returns(uint256)
func (_Aggregator *AggregatorCaller) FilterThreshold(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "filterThreshold")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FilterThreshold is a free data retrieval call binding the contract method 0x717b90b4.
//
// Solidity: function filterThreshold() view returns(uint256)
func (_Aggregator *AggregatorSession) FilterThreshold() (*big.Int, error) {
	return _Aggregator.Contract.FilterThreshold(&_Aggregator.CallOpts)
}

// FilterThreshold is a free data retrieval call binding the contract method 0x717b90b4.
//
// Solidity: function filterThreshold() view returns(uint256)
func (_Aggregator *AggregatorCallerSession) FilterThreshold() (*big.Int, error) {
	return _Aggregator.Contract.FilterThreshold(&_Aggregator.CallOpts)
}

// FilterType is a free data retrieval call binding the contract method 0x83dc9373.
//
// Solidity: function filterType() view returns(uint8)
func (_Aggregator *AggregatorCaller) FilterType(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "filterType")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// FilterType is a free data retrieval call binding the contract method 0x83dc9373.
//
// Solidity: function filterType() view returns(uint8)
func (_Aggregator *AggregatorSession) FilterType() (uint8, error) {
	return _Aggregator.Contract.FilterType(&_Aggregator.CallOpts)
}

// FilterType is a free data retrieval call binding the contract method 0x83dc9373.
//
// Solidity: function filterType() view returns(uint8)
func (_Aggregator *AggregatorCallerSession) FilterType() (uint8, error) {
	return _Aggregator.Contract.FilterType(&_Aggregator.CallOpts)
}

// GetFilterPolicy is a free data retrieval call binding the contract method 0xe41a9bff.
//
// Solidity: function getFilterPolicy() view returns(uint8, uint256)
func (_Aggregator *AggregatorCaller) GetFilterPolicy(opts *bind.CallOpts) (uint8, *big.Int, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "getFilterPolicy")

	if err != nil {
		return *new(uint8), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return out0, out1, err

}

// GetFilterPolicy is a free data retrieval call binding the contract method 0xe41a9bff.
//
// Solidity: function getFilterPolicy() view returns(uint8, uint256)
func (_Aggregator *AggregatorSession) GetFilterPolicy() (uint8, *big.Int, error) {
	return _Aggregator.Contract.GetFilterPolicy(&_Aggregator.CallOpts)
}

// GetFilterPolicy is a free data retrieval call binding the contract method 0xe41a9bff.
//
// Solidity: function getFilterPolicy() view returns(uint8, uint256)
func (_Aggregator *AggregatorCallerSession) GetFilterPolicy() (uint8, *big.Int, error) {
	return _Aggregator.Contract.GetFilterPolicy(&_Aggregator.CallOpts)
}

// GetResult is a free data retrieval call binding the contract method 0x995e4339.
//
// Solidity: function getResult(uint256 _jobId) view returns(int128[], address, uint256)
func (_Aggregator *AggregatorCaller) GetResult(opts *bind.CallOpts, _jobId *big.Int) ([]*big.Int, common.Address, *big.Int, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "getResult", _jobId)

	if err != nil {
		return *new([]*big.Int), *new(common.Address), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)
	out1 := *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	out2 := *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return out0, out1, out2, err

}

// GetResult is a free data retrieval call binding the contract method 0x995e4339.
//
// Solidity: function getResult(uint256 _jobId) view returns(int128[], address, uint256)
func (_Aggregator *AggregatorSession) GetResult(_jobId *big.Int) ([]*big.Int, common.Address, *big.Int, error) {
	return _Aggregator.Contract.GetResult(&_Aggregator.CallOpts, _jobId)
}

// GetResult is a free data retrieval call binding the contract method 0x995e4339.
//
// Solidity: function getResult(uint256 _jobId) view returns(int128[], address, uint256)
func (_Aggregator *AggregatorCallerSession) GetResult(_jobId *big.Int) ([]*big.Int, common.Address, *big.Int, error) {
	return _Aggregator.Contract.GetResult(&_Aggregator.CallOpts, _jobId)
}

// IsCompleted is a free data retrieval call binding the contract method 0x7a41984b.
//
// Solidity: function isCompleted(uint256 _jobId) view returns(bool)
func (_Aggregator *AggregatorCaller) IsCompleted(opts *bind.CallOpts, _jobId *big.Int) (bool, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "isCompleted", _jobId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsCompleted is a free data retrieval call binding the contract method 0x7a41984b.
//
// Solidity: function isCompleted(uint256 _jobId) view returns(bool)
func (_Aggregator *AggregatorSession) IsCompleted(_jobId *big.Int) (bool, error) {
	return _Aggregator.Contract.IsCompleted(&_Aggregator.CallOpts, _jobId)
}

// IsCompleted is a free data retrieval call binding the contract method 0x7a41984b.
//
// Solidity: function isCompleted(uint256 _jobId) view returns(bool)
func (_Aggregator *AggregatorCallerSession) IsCompleted(_jobId *big.Int) (bool, error) {
	return _Aggregator.Contract.IsCompleted(&_Aggregator.CallOpts, _jobId)
}

// Manager is a free data retrieval call binding the contract method 0x481c6a75.
//
// Solidity: function manager() view returns(address)
func (_Aggregator *AggregatorCaller) Manager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "manager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Manager is a free data retrieval call binding the contract method 0x481c6a75.
//
// Solidity: function manager() view returns(address)
func (_Aggregator *AggregatorSession) Manager() (common.Address, error) {
	return _Aggregator.Contract.Manager(&_Aggregator.CallOpts)
}

// Manager is a free data retrieval call binding the contract method 0x481c6a75.
//
// Solidity: function manager() view returns(address)
func (_Aggregator *AggregatorCallerSession) Manager() (common.Address, error) {
	return _Aggregator.Contract.Manager(&_Aggregator.CallOpts)
}

// ModelCreatorReward is a free data retrieval call binding the contract method 0xa7878be6.
//
// Solidity: function modelCreatorReward() view returns(uint256)
func (_Aggregator *AggregatorCaller) ModelCreatorReward(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "modelCreatorReward")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ModelCreatorReward is a free data retrieval call binding the contract method 0xa7878be6.
//
// Solidity: function modelCreatorReward() view returns(uint256)
func (_Aggregator *AggregatorSession) ModelCreatorReward() (*big.Int, error) {
	return _Aggregator.Contract.ModelCreatorReward(&_Aggregator.CallOpts)
}

// ModelCreatorReward is a free data retrieval call binding the contract method 0xa7878be6.
//
// Solidity: function modelCreatorReward() view returns(uint256)
func (_Aggregator *AggregatorCallerSession) ModelCreatorReward() (*big.Int, error) {
	return _Aggregator.Contract.ModelCreatorReward(&_Aggregator.CallOpts)
}

// OracleReward is a free data retrieval call binding the contract method 0x21873631.
//
// Solidity: function oracleReward() view returns(uint256)
func (_Aggregator *AggregatorCaller) OracleReward(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "oracleReward")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// OracleReward is a free data retrieval call binding the contract method 0x21873631.
//
// Solidity: function oracleReward() view returns(uint256)
func (_Aggregator *AggregatorSession) OracleReward() (*big.Int, error) {
	return _Aggregator.Contract.OracleReward(&_Aggregator.CallOpts)
}

// OracleReward is a free data retrieval call binding the contract method 0x21873631.
//
// Solidity: function oracleReward() view returns(uint256)
func (_Aggregator *AggregatorCallerSession) OracleReward() (*big.Int, error) {
	return _Aggregator.Contract.OracleReward(&_Aggregator.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Aggregator *AggregatorCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Aggregator *AggregatorSession) Owner() (common.Address, error) {
	return _Aggregator.Contract.Owner(&_Aggregator.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Aggregator *AggregatorCallerSession) Owner() (common.Address, error) {
	return _Aggregator.Contract.Owner(&_Aggregator.CallOpts)
}

// QueryFee is a free data retrieval call binding the contract method 0xfdd26881.
//
// Solidity: function queryFee() view returns(uint256)
func (_Aggregator *AggregatorCaller) QueryFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "queryFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// QueryFee is a free data retrieval call binding the contract method 0xfdd26881.
//
// Solidity: function queryFee() view returns(uint256)
func (_Aggregator *AggregatorSession) QueryFee() (*big.Int, error) {
	return _Aggregator.Contract.QueryFee(&_Aggregator.CallOpts)
}

// QueryFee is a free data retrieval call binding the contract method 0xfdd26881.
//
// Solidity: function queryFee() view returns(uint256)
func (_Aggregator *AggregatorCallerSession) QueryFee() (*big.Int, error) {
	return _Aggregator.Contract.QueryFee(&_Aggregator.CallOpts)
}

// Queue is a free data retrieval call binding the contract method 0xe10d29ee.
//
// Solidity: function queue() view returns(address)
func (_Aggregator *AggregatorCaller) Queue(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "queue")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Queue is a free data retrieval call binding the contract method 0xe10d29ee.
//
// Solidity: function queue() view returns(address)
func (_Aggregator *AggregatorSession) Queue() (common.Address, error) {
	return _Aggregator.Contract.Queue(&_Aggregator.CallOpts)
}

// Queue is a free data retrieval call binding the contract method 0xe10d29ee.
//
// Solidity: function queue() view returns(address)
func (_Aggregator *AggregatorCallerSession) Queue() (common.Address, error) {
	return _Aggregator.Contract.Queue(&_Aggregator.CallOpts)
}

// Verifier is a free data retrieval call binding the contract method 0x2b7ac3f3.
//
// Solidity: function verifier() view returns(address)
func (_Aggregator *AggregatorCaller) Verifier(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Aggregator.contract.Call(opts, &out, "verifier")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Verifier is a free data retrieval call binding the contract method 0x2b7ac3f3.
//
// Solidity: function verifier() view returns(address)
func (_Aggregator *AggregatorSession) Verifier() (common.Address, error) {
	return _Aggregator.Contract.Verifier(&_Aggregator.CallOpts)
}

// Verifier is a free data retrieval call binding the contract method 0x2b7ac3f3.
//
// Solidity: function verifier() view returns(address)
func (_Aggregator *AggregatorCallerSession) Verifier() (common.Address, error) {
	return _Aggregator.Contract.Verifier(&_Aggregator.CallOpts)
}

// ApproveJob is a paid mutator transaction binding the contract method 0x4bd23b9e.
//
// Solidity: function approveJob(uint256 _requestId) returns(uint256)
func (_Aggregator *AggregatorTransactor) ApproveJob(opts *bind.TransactOpts, _requestId *big.Int) (*types.Transaction, error) {
	return _Aggregator.contract.Transact(opts, "approveJob", _requestId)
}

// ApproveJob is a paid mutator transaction binding the contract method 0x4bd23b9e.
//
// Solidity: function approveJob(uint256 _requestId) returns(uint256)
func (_Aggregator *AggregatorSession) ApproveJob(_requestId *big.Int) (*types.Transaction, error) {
	return _Aggregator.Contract.ApproveJob(&_Aggregator.TransactOpts, _requestId)
}

// ApproveJob is a paid mutator transaction binding the contract method 0x4bd23b9e.
//
// Solidity: function approveJob(uint256 _requestId) returns(uint256)
func (_Aggregator *AggregatorTransactorSession) ApproveJob(_requestId *big.Int) (*types.Transaction, error) {
	return _Aggregator.Contract.ApproveJob(&_Aggregator.TransactOpts, _requestId)
}

// DistributeRewards is a paid mutator transaction binding the contract method 0x8dc3ec4d.
//
// Solidity: function distributeRewards(address _oracle, uint256 _jobId, uint256 _vectorLength) returns()
func (_Aggregator *AggregatorTransactor) DistributeRewards(opts *bind.TransactOpts, _oracle common.Address, _jobId *big.Int, _vectorLength *big.Int) (*types.Transaction, error) {
	return _Aggregator.contract.Transact(opts, "distributeRewards", _oracle, _jobId, _vectorLength)
}

// DistributeRewards is a paid mutator transaction binding the contract method 0x8dc3ec4d.
//
// Solidity: function distributeRewards(address _oracle, uint256 _jobId, uint256 _vectorLength) returns()
func (_Aggregator *AggregatorSession) DistributeRewards(_oracle common.Address, _jobId *big.Int, _vectorLength *big.Int) (*types.Transaction, error) {
	return _Aggregator.Contract.DistributeRewards(&_Aggregator.TransactOpts, _oracle, _jobId, _vectorLength)
}

// DistributeRewards is a paid mutator transaction binding the contract method 0x8dc3ec4d.
//
// Solidity: function distributeRewards(address _oracle, uint256 _jobId, uint256 _vectorLength) returns()
func (_Aggregator *AggregatorTransactorSession) DistributeRewards(_oracle common.Address, _jobId *big.Int, _vectorLength *big.Int) (*types.Transaction, error) {
	return _Aggregator.Contract.DistributeRewards(&_Aggregator.TransactOpts, _oracle, _jobId, _vectorLength)
}

// RequestAttribution is a paid mutator transaction binding the contract method 0x853edc03.
//
// Solidity: function requestAttribution(string _ipfsCid) payable returns(uint256)
func (_Aggregator *AggregatorTransactor) RequestAttribution(opts *bind.TransactOpts, _ipfsCid string) (*types.Transaction, error) {
	return _Aggregator.contract.Transact(opts, "requestAttribution", _ipfsCid)
}

// RequestAttribution is a paid mutator transaction binding the contract method 0x853edc03.
//
// Solidity: function requestAttribution(string _ipfsCid) payable returns(uint256)
func (_Aggregator *AggregatorSession) RequestAttribution(_ipfsCid string) (*types.Transaction, error) {
	return _Aggregator.Contract.RequestAttribution(&_Aggregator.TransactOpts, _ipfsCid)
}

// RequestAttribution is a paid mutator transaction binding the contract method 0x853edc03.
//
// Solidity: function requestAttribution(string _ipfsCid) payable returns(uint256)
func (_Aggregator *AggregatorTransactorSession) RequestAttribution(_ipfsCid string) (*types.Transaction, error) {
	return _Aggregator.Contract.RequestAttribution(&_Aggregator.TransactOpts, _ipfsCid)
}

// SetFilterPolicy is a paid mutator transaction binding the contract method 0x51d2a7d4.
//
// Solidity: function setFilterPolicy(uint8 _filterType, uint256 _threshold) returns()
func (_Aggregator *AggregatorTransactor) SetFilterPolicy(opts *bind.TransactOpts, _filterType uint8, _threshold *big.Int) (*types.Transaction, error) {
	return _Aggregator.contract.Transact(opts, "setFilterPolicy", _filterType, _threshold)
}

// SetFilterPolicy is a paid mutator transaction binding the contract method 0x51d2a7d4.
//
// Solidity: function setFilterPolicy(uint8 _filterType, uint256 _threshold) returns()
func (_Aggregator *AggregatorSession) SetFilterPolicy(_filterType uint8, _threshold *big.Int) (*types.Transaction, error) {
	return _Aggregator.Contract.SetFilterPolicy(&_Aggregator.TransactOpts, _filterType, _threshold)
}

// SetFilterPolicy is a paid mutator transaction binding the contract method 0x51d2a7d4.
//
// Solidity: function setFilterPolicy(uint8 _filterType, uint256 _threshold) returns()
func (_Aggregator *AggregatorTransactorSession) SetFilterPolicy(_filterType uint8, _threshold *big.Int) (*types.Transaction, error) {
	return _Aggregator.Contract.SetFilterPolicy(&_Aggregator.TransactOpts, _filterType, _threshold)
}

// Transmit is a paid mutator transaction binding the contract method 0xf957c546.
//
// Solidity: function transmit(bytes32 configDigest, uint64 seqNr, bytes report, bytes32[] rs, bytes32[] ss, bytes32 rawVs) returns()
func (_Aggregator *AggregatorTransactor) Transmit(opts *bind.TransactOpts, configDigest [32]byte, seqNr uint64, report []byte, rs [][32]byte, ss [][32]byte, rawVs [32]byte) (*types.Transaction, error) {
	return _Aggregator.contract.Transact(opts, "transmit", configDigest, seqNr, report, rs, ss, rawVs)
}

// Transmit is a paid mutator transaction binding the contract method 0xf957c546.
//
// Solidity: function transmit(bytes32 configDigest, uint64 seqNr, bytes report, bytes32[] rs, bytes32[] ss, bytes32 rawVs) returns()
func (_Aggregator *AggregatorSession) Transmit(configDigest [32]byte, seqNr uint64, report []byte, rs [][32]byte, ss [][32]byte, rawVs [32]byte) (*types.Transaction, error) {
	return _Aggregator.Contract.Transmit(&_Aggregator.TransactOpts, configDigest, seqNr, report, rs, ss, rawVs)
}

// Transmit is a paid mutator transaction binding the contract method 0xf957c546.
//
// Solidity: function transmit(bytes32 configDigest, uint64 seqNr, bytes report, bytes32[] rs, bytes32[] ss, bytes32 rawVs) returns()
func (_Aggregator *AggregatorTransactorSession) Transmit(configDigest [32]byte, seqNr uint64, report []byte, rs [][32]byte, ss [][32]byte, rawVs [32]byte) (*types.Transaction, error) {
	return _Aggregator.Contract.Transmit(&_Aggregator.TransactOpts, configDigest, seqNr, report, rs, ss, rawVs)
}

// AggregatorJobCompletedIterator is returned from FilterJobCompleted and is used to iterate over the raw logs and unpacked data for JobCompleted events raised by the Aggregator contract.
type AggregatorJobCompletedIterator struct {
	Event *AggregatorJobCompleted // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AggregatorJobCompletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AggregatorJobCompleted)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AggregatorJobCompleted)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AggregatorJobCompletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AggregatorJobCompletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AggregatorJobCompleted represents a JobCompleted event raised by the Aggregator contract.
type AggregatorJobCompleted struct {
	JobId        *big.Int
	Submitter    common.Address
	VectorLength *big.Int
	Timestamp    *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterJobCompleted is a free log retrieval operation binding the contract event 0xbeccb946f65bcbea37397b769e33a37dabec51c94907407fd5f2eaa81f1a0bfa.
//
// Solidity: event JobCompleted(uint256 indexed jobId, address indexed submitter, uint256 vectorLength, uint256 timestamp)
func (_Aggregator *AggregatorFilterer) FilterJobCompleted(opts *bind.FilterOpts, jobId []*big.Int, submitter []common.Address) (*AggregatorJobCompletedIterator, error) {

	var jobIdRule []interface{}
	for _, jobIdItem := range jobId {
		jobIdRule = append(jobIdRule, jobIdItem)
	}
	var submitterRule []interface{}
	for _, submitterItem := range submitter {
		submitterRule = append(submitterRule, submitterItem)
	}

	logs, sub, err := _Aggregator.contract.FilterLogs(opts, "JobCompleted", jobIdRule, submitterRule)
	if err != nil {
		return nil, err
	}
	return &AggregatorJobCompletedIterator{contract: _Aggregator.contract, event: "JobCompleted", logs: logs, sub: sub}, nil
}

// WatchJobCompleted is a free log subscription operation binding the contract event 0xbeccb946f65bcbea37397b769e33a37dabec51c94907407fd5f2eaa81f1a0bfa.
//
// Solidity: event JobCompleted(uint256 indexed jobId, address indexed submitter, uint256 vectorLength, uint256 timestamp)
func (_Aggregator *AggregatorFilterer) WatchJobCompleted(opts *bind.WatchOpts, sink chan<- *AggregatorJobCompleted, jobId []*big.Int, submitter []common.Address) (event.Subscription, error) {

	var jobIdRule []interface{}
	for _, jobIdItem := range jobId {
		jobIdRule = append(jobIdRule, jobIdItem)
	}
	var submitterRule []interface{}
	for _, submitterItem := range submitter {
		submitterRule = append(submitterRule, submitterItem)
	}

	logs, sub, err := _Aggregator.contract.WatchLogs(opts, "JobCompleted", jobIdRule, submitterRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AggregatorJobCompleted)
				if err := _Aggregator.contract.UnpackLog(event, "JobCompleted", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseJobCompleted is a log parse operation binding the contract event 0xbeccb946f65bcbea37397b769e33a37dabec51c94907407fd5f2eaa81f1a0bfa.
//
// Solidity: event JobCompleted(uint256 indexed jobId, address indexed submitter, uint256 vectorLength, uint256 timestamp)
func (_Aggregator *AggregatorFilterer) ParseJobCompleted(log types.Log) (*AggregatorJobCompleted, error) {
	event := new(AggregatorJobCompleted)
	if err := _Aggregator.contract.UnpackLog(event, "JobCompleted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AggregatorLogNewCustomerRequestIterator is returned from FilterLogNewCustomerRequest and is used to iterate over the raw logs and unpacked data for LogNewCustomerRequest events raised by the Aggregator contract.
type AggregatorLogNewCustomerRequestIterator struct {
	Event *AggregatorLogNewCustomerRequest // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AggregatorLogNewCustomerRequestIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AggregatorLogNewCustomerRequest)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AggregatorLogNewCustomerRequest)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AggregatorLogNewCustomerRequestIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AggregatorLogNewCustomerRequestIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AggregatorLogNewCustomerRequest represents a LogNewCustomerRequest event raised by the Aggregator contract.
type AggregatorLogNewCustomerRequest struct {
	RequestId *big.Int
	IpfsCid   string
	Requester common.Address
	Payment   *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterLogNewCustomerRequest is a free log retrieval operation binding the contract event 0x4498100bf5e1c1ca57375bd8b4f12c7cf4b4459bbe7c9ae6b6d55c44987bf001.
//
// Solidity: event LogNewCustomerRequest(uint256 indexed requestId, string ipfsCid, address requester, uint256 payment)
func (_Aggregator *AggregatorFilterer) FilterLogNewCustomerRequest(opts *bind.FilterOpts, requestId []*big.Int) (*AggregatorLogNewCustomerRequestIterator, error) {

	var requestIdRule []interface{}
	for _, requestIdItem := range requestId {
		requestIdRule = append(requestIdRule, requestIdItem)
	}

	logs, sub, err := _Aggregator.contract.FilterLogs(opts, "LogNewCustomerRequest", requestIdRule)
	if err != nil {
		return nil, err
	}
	return &AggregatorLogNewCustomerRequestIterator{contract: _Aggregator.contract, event: "LogNewCustomerRequest", logs: logs, sub: sub}, nil
}

// WatchLogNewCustomerRequest is a free log subscription operation binding the contract event 0x4498100bf5e1c1ca57375bd8b4f12c7cf4b4459bbe7c9ae6b6d55c44987bf001.
//
// Solidity: event LogNewCustomerRequest(uint256 indexed requestId, string ipfsCid, address requester, uint256 payment)
func (_Aggregator *AggregatorFilterer) WatchLogNewCustomerRequest(opts *bind.WatchOpts, sink chan<- *AggregatorLogNewCustomerRequest, requestId []*big.Int) (event.Subscription, error) {

	var requestIdRule []interface{}
	for _, requestIdItem := range requestId {
		requestIdRule = append(requestIdRule, requestIdItem)
	}

	logs, sub, err := _Aggregator.contract.WatchLogs(opts, "LogNewCustomerRequest", requestIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AggregatorLogNewCustomerRequest)
				if err := _Aggregator.contract.UnpackLog(event, "LogNewCustomerRequest", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseLogNewCustomerRequest is a log parse operation binding the contract event 0x4498100bf5e1c1ca57375bd8b4f12c7cf4b4459bbe7c9ae6b6d55c44987bf001.
//
// Solidity: event LogNewCustomerRequest(uint256 indexed requestId, string ipfsCid, address requester, uint256 payment)
func (_Aggregator *AggregatorFilterer) ParseLogNewCustomerRequest(log types.Log) (*AggregatorLogNewCustomerRequest, error) {
	event := new(AggregatorLogNewCustomerRequest)
	if err := _Aggregator.contract.UnpackLog(event, "LogNewCustomerRequest", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AggregatorLogNewJobForOraclesIterator is returned from FilterLogNewJobForOracles and is used to iterate over the raw logs and unpacked data for LogNewJobForOracles events raised by the Aggregator contract.
type AggregatorLogNewJobForOraclesIterator struct {
	Event *AggregatorLogNewJobForOracles // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AggregatorLogNewJobForOraclesIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AggregatorLogNewJobForOracles)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AggregatorLogNewJobForOracles)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AggregatorLogNewJobForOraclesIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AggregatorLogNewJobForOraclesIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AggregatorLogNewJobForOracles represents a LogNewJobForOracles event raised by the Aggregator contract.
type AggregatorLogNewJobForOracles struct {
	JobId   *big.Int
	IpfsCid string
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterLogNewJobForOracles is a free log retrieval operation binding the contract event 0x5af5066ee921684d5e820a56d6e5abdfb117e07345b9e29718782ba6251f68a5.
//
// Solidity: event LogNewJobForOracles(uint256 indexed jobId, string ipfsCid)
func (_Aggregator *AggregatorFilterer) FilterLogNewJobForOracles(opts *bind.FilterOpts, jobId []*big.Int) (*AggregatorLogNewJobForOraclesIterator, error) {

	var jobIdRule []interface{}
	for _, jobIdItem := range jobId {
		jobIdRule = append(jobIdRule, jobIdItem)
	}

	logs, sub, err := _Aggregator.contract.FilterLogs(opts, "LogNewJobForOracles", jobIdRule)
	if err != nil {
		return nil, err
	}
	return &AggregatorLogNewJobForOraclesIterator{contract: _Aggregator.contract, event: "LogNewJobForOracles", logs: logs, sub: sub}, nil
}

// WatchLogNewJobForOracles is a free log subscription operation binding the contract event 0x5af5066ee921684d5e820a56d6e5abdfb117e07345b9e29718782ba6251f68a5.
//
// Solidity: event LogNewJobForOracles(uint256 indexed jobId, string ipfsCid)
func (_Aggregator *AggregatorFilterer) WatchLogNewJobForOracles(opts *bind.WatchOpts, sink chan<- *AggregatorLogNewJobForOracles, jobId []*big.Int) (event.Subscription, error) {

	var jobIdRule []interface{}
	for _, jobIdItem := range jobId {
		jobIdRule = append(jobIdRule, jobIdItem)
	}

	logs, sub, err := _Aggregator.contract.WatchLogs(opts, "LogNewJobForOracles", jobIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AggregatorLogNewJobForOracles)
				if err := _Aggregator.contract.UnpackLog(event, "LogNewJobForOracles", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseLogNewJobForOracles is a log parse operation binding the contract event 0x5af5066ee921684d5e820a56d6e5abdfb117e07345b9e29718782ba6251f68a5.
//
// Solidity: event LogNewJobForOracles(uint256 indexed jobId, string ipfsCid)
func (_Aggregator *AggregatorFilterer) ParseLogNewJobForOracles(log types.Log) (*AggregatorLogNewJobForOracles, error) {
	event := new(AggregatorLogNewJobForOracles)
	if err := _Aggregator.contract.UnpackLog(event, "LogNewJobForOracles", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
