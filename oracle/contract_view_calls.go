package main

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"OCR3-thesis/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

func callContractNoFrom(ctx context.Context, client *rpc.Client, abiJSON string, contract common.Address, method string, params ...interface{}) ([]interface{}, error) {
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("parse ABI: %w", err)
	}

	input, err := parsedABI.Pack(method, params...)
	if err != nil {
		return nil, fmt.Errorf("pack %s: %w", method, err)
	}

	callArg := map[string]string{
		"to":   contract.Hex(),
		"data": hexutil.Encode(input),
	}

	var resultHex string
	if err := client.CallContext(ctx, &resultHex, "eth_call", callArg, "latest"); err != nil {
		return nil, err
	}

	output, err := hexutil.Decode(resultHex)
	if err != nil {
		return nil, fmt.Errorf("decode %s result: %w", method, err)
	}

	values, err := parsedABI.Unpack(method, output)
	if err != nil {
		return nil, fmt.Errorf("unpack %s result: %w", method, err)
	}
	return values, nil
}

func readAddressNoFrom(ctx context.Context, client *rpc.Client, abiJSON string, contract common.Address, method string) (common.Address, error) {
	values, err := callContractNoFrom(ctx, client, abiJSON, contract, method)
	if err != nil {
		return common.Address{}, err
	}
	if len(values) != 1 {
		return common.Address{}, fmt.Errorf("%s returned %d values", method, len(values))
	}

	addr, ok := values[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("%s returned %T, expected address", method, values[0])
	}
	return addr, nil
}

func readAggregatorAddressNoFrom(ctx context.Context, client *rpc.Client, aggregator common.Address, method string) (common.Address, error) {
	return readAddressNoFrom(ctx, client, contracts.AggregatorMetaData.ABI, aggregator, method)
}

func readAggregatorFilterPolicyNoFrom(ctx context.Context, client *rpc.Client, aggregator common.Address) (uint8, *big.Int, error) {
	values, err := callContractNoFrom(ctx, client, contracts.AggregatorMetaData.ABI, aggregator, "getFilterPolicy")
	if err != nil {
		return 0, nil, err
	}
	if len(values) != 2 {
		return 0, nil, fmt.Errorf("getFilterPolicy returned %d values", len(values))
	}

	filterType, ok := values[0].(uint8)
	if !ok {
		return 0, nil, fmt.Errorf("getFilterPolicy returned %T for filter type", values[0])
	}
	threshold, ok := values[1].(*big.Int)
	if !ok {
		return 0, nil, fmt.Errorf("getFilterPolicy returned %T for threshold", values[1])
	}
	return filterType, threshold, nil
}

func readAggregatorIsCompletedNoFrom(ctx context.Context, client *rpc.Client, aggregator common.Address, jobID *big.Int) (bool, error) {
	values, err := callContractNoFrom(ctx, client, contracts.AggregatorMetaData.ABI, aggregator, "isCompleted", jobID)
	if err != nil {
		return false, err
	}
	if len(values) != 1 {
		return false, fmt.Errorf("isCompleted returned %d values", len(values))
	}

	done, ok := values[0].(bool)
	if !ok {
		return false, fmt.Errorf("isCompleted returned %T, expected bool", values[0])
	}
	return done, nil
}
