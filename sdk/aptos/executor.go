package aptos

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/ethereum/go-ethereum/common"
	chain_selectors "github.com/smartcontractkit/chain-selectors"

	aptosutil "github.com/smartcontractkit/mcms/e2e/utils/aptos"
	"github.com/smartcontractkit/mcms/sdk"
	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Executor = &Executor{}

type Executor struct {
	*Encoder
	*Inspector
	client *aptos.NodeClient
	auth   *aptos.Account
}

func NewExecutor(client *aptos.NodeClient, auth *aptos.Account, encoder *Encoder) *Executor {
	return &Executor{
		Encoder:   encoder,
		Inspector: NewInspector(client),
		client:    client,
		auth:      auth,
	}
}

func (e Executor) ExecuteOperation(
	ctx context.Context,
	metadata types.ChainMetadata,
	nonce uint32,
	proof []common.Hash,
	op types.Operation,
) (types.TransactionResult, error) {
	var mcmsAddress aptos.AccountAddress
	if err := mcmsAddress.ParseStringRelaxed(metadata.MCMAddress); err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to parse mcm address: %w", err)
	}
	var toAddress aptos.AccountAddress
	if err := toAddress.ParseStringRelaxed(op.Transaction.To); err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to parse to address: %w", err)
	}
	var additionalFields AdditionalFields
	if err := json.Unmarshal(op.Transaction.AdditionalFields, &additionalFields); err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to unmarshal additional fields: %w", err)
	}
	chainID, err := chain_selectors.AptosChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return types.TransactionResult{}, &sdkerrors.InvalidChainIDError{
			ReceivedChainID: e.ChainSelector,
		}
	}
	chainIDBig := (&big.Int{}).SetUint64(chainID)

	payload, err := aptosutil.BuildTransactionPayload(
		metadata.MCMAddress+"::mcms::execute",
		nil,
		[]string{
			"u256",
			"address",
			"u64",
			"address",
			"0x1::string::String",
			"0x1::string::String",
			"vector<u8>",
			"vector<vector<u8>>",
		},
		[]any{
			chainIDBig,
			mcmsAddress,
			uint64(nonce),
			toAddress,
			additionalFields.ModuleName,
			additionalFields.Function,
			op.Transaction.Data,
			proof,
		},
	)
	if err != nil {
		return types.TransactionResult{}, err
	}
	data, err := aptosutil.BuildSignSubmitAndWaitForTransaction(e.client, e.auth, payload)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("setting root on Aptos mcms contract: %w", err)
	}

	found := false
	for _, event := range data.Events {
		if event.Type == mcmsAddress.StringLong()+"::mcms::OpExecuted" {
			if nonce, ok := event.Data["nonce"]; ok {
				_ = nonce
				found = true
			}
		}
	}
	if !found {
		return types.TransactionResult{}, fmt.Errorf("unable to find config event on Aptos mcms contract")
	}
	return types.TransactionResult{
		Hash:           data.Hash,
		ChainFamily:    chain_selectors.FamilyAptos,
		RawTransaction: data,
	}, nil
}

func (e Executor) SetRoot(
	ctx context.Context,
	metadata types.ChainMetadata,
	proof []common.Hash,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
) (types.TransactionResult, error) {
	var mcmsAddress aptos.AccountAddress
	if err := mcmsAddress.ParseStringRelaxed(metadata.MCMAddress); err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to parse mcm address: %w", err)
	}
	chainID, err := chain_selectors.AptosChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return types.TransactionResult{}, &sdkerrors.InvalidChainIDError{
			ReceivedChainID: e.ChainSelector,
		}
	}
	chainIDBig := (&big.Int{}).SetUint64(chainID)

	signatures := encodeSignatures(sortedSignatures)

	payload, err := aptosutil.BuildTransactionPayload(
		metadata.MCMAddress+"::mcms::set_root",
		nil,
		[]string{
			"vector<u8>",
			"u64",
			"u256",
			"address",
			"u64",
			"u64",
			"bool",
			"vector<vector<u8>>",
			"vector<vector<u8>>",
		},
		[]any{
			root,
			uint64(validUntil),
			chainIDBig,
			mcmsAddress,
			metadata.StartingOpCount,
			metadata.StartingOpCount + e.TxCount,
			e.OverridePreviousRoot,
			proof,
			signatures,
		},
	)
	if err != nil {
		return types.TransactionResult{}, err
	}
	data, err := aptosutil.BuildSignSubmitAndWaitForTransaction(e.client, e.auth, payload)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("setting root on Aptos mcms contract: %w", err)
	}

	found := false
	for _, event := range data.Events {
		if event.Type == mcmsAddress.StringLong()+"::mcms::NewRoot" {
			if root, ok := event.Data["root"]; ok {
				_ = root
				found = true
			}
		}
	}
	if !found {
		return types.TransactionResult{}, fmt.Errorf("unable to find config event on Aptos mcms contract")
	}
	return types.TransactionResult{
		Hash:           data.Hash,
		ChainFamily:    chain_selectors.FamilyAptos,
		RawTransaction: data,
	}, nil
}

func encodeSignatures(signatures []types.Signature) [][]byte {
	sigs := make([][]byte, len(signatures))
	for i, signature := range signatures {
		sigs[i] = append(signature.R.Bytes(), signature.S.Bytes()...)
		if signature.V <= 4 {
			sigs[i] = append(sigs[i], signature.V+27)
		} else {
			sigs[i] = append(sigs[i], signature.V)
		}
	}
	return sigs
}
