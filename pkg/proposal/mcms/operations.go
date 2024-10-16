package mcms

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/mcms/pkg/gethwrappers"
)

var MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP = crypto.Keccak256Hash([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP"))

type ChainIdentifier uint64

type OperationEncoder interface {
	Hash(operation ChainOperation, nonce uint32) (common.Hash, error)
}

type OperationExecutor interface {
	Execute(operation ChainOperation, nonce uint32, proof []common.Hash) error
}

type OperationMetadata struct {
	ContractType string   `json:"contractType"`
	Tags         []string `json:"tags"`
}

type ChainOperation struct {
	ChainID          ChainIdentifier `json:"chainIdentifier"`
	To               string          `json:"to"`
	Data             []byte          `json:"data"`
	AdditionalFields json.RawMessage `json:"additionalFields"`
	OperationMetadata
}

type EVMAdditionalFields struct {
	Value *big.Int `json:"value"`
}

type EVMOperationEncoder struct {
	ChainId  uint64
	Multisig common.Address
}

func (e *EVMOperationEncoder) Hash(operation ChainOperation, nonce uint32) (common.Hash, error) {
	// Unmarshal additional fields
	var additionalFields EVMAdditionalFields
	if err := json.Unmarshal(operation.AdditionalFields, &additionalFields); err != nil {
		return common.Hash{}, err
	}

	op := gethwrappers.ManyChainMultiSigOp{
		ChainId:  new(big.Int).SetUint64(e.ChainId),
		MultiSig: e.Multisig,
		Nonce:    new(big.Int).SetUint64(uint64(nonce)),
		To:       common.HexToAddress(operation.To),
		Data:     operation.Data,
		Value:    additionalFields.Value,
	}

	abi := `[{"type":"bytes32"},{"type":"tuple","components":[{"name":"chainId","type":"uint256"},{"name":"multiSig","type":"address"},{"name":"nonce","type":"uint40"},{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"}]}]`
	encoded, err := ABIEncode(abi, MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP, op)
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(encoded), nil
}

type EVMOperationExecutor struct {
	EVMOperationEncoder
	client ContractDeployBackend
	auth   *bind.TransactOpts
}

func (e *EVMOperationExecutor) Execute(operation ChainOperation, nonce uint64, proof []common.Hash) error {
	// Unmarshal additional fields
	var additionalFields EVMAdditionalFields
	if err := json.Unmarshal(operation.AdditionalFields, &additionalFields); err != nil {
		return err
	}

	op := gethwrappers.ManyChainMultiSigOp{
		ChainId:  new(big.Int).SetUint64(e.ChainId),
		MultiSig: e.Multisig,
		Nonce:    new(big.Int).SetUint64(nonce),
		To:       common.HexToAddress(operation.To),
		Data:     operation.Data,
		Value:    additionalFields.Value,
	}

	mcms, err := gethwrappers.NewManyChainMultiSig(e.Multisig, e.client)
	if err != nil {
		return err
	}

	tx, err := mcms.Execute(e.auth, op, transformHashes(proof))
	if err != nil {
		return err
	}

	receipt, err := bind.WaitMined(context.Background(), e.client, tx)
	if err != nil {
		return err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return errors.New("transaction failed")
	}

	return nil
}
