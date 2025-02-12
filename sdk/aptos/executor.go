package aptos

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/api"
	"github.com/ethereum/go-ethereum/common"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-internal-integrations/aptos/bindings/bind"
	"github.com/smartcontractkit/chainlink-internal-integrations/aptos/bindings/mcms"
	module_mcms "github.com/smartcontractkit/chainlink-internal-integrations/aptos/bindings/mcms/mcms"

	"github.com/smartcontractkit/mcms/sdk"
	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/types"
)

const (
	SignatureVOffset    = 27
	SignatureVThreshold = 2

	ChunkSizeBytes = 100_000
)

var _ sdk.Executor = &Executor{}

type Executor struct {
	*Encoder
	*Inspector
	client aptos.AptosRpcClient
	auth   aptos.TransactionSigner
}

func NewExecutor(client aptos.AptosRpcClient, auth aptos.TransactionSigner, encoder *Encoder) *Executor {
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

	mcmsC := mcms.Bind(mcmsAddress, e.client)
	opts := &bind.TransactOpts{Signer: e.auth}

	var tx *api.PendingTransaction
	if len(op.Transaction.Data) <= ChunkSizeBytes {
		tx, err = mcmsC.MCMS.Execute(
			opts,
			module_mcms.Op{
				ChainId:    *chainIDBig,
				Multisig:   mcmsAddress,
				Nonce:      uint64(nonce),
				To:         toAddress,
				ModuleName: additionalFields.ModuleName,
				Function:   additionalFields.Function,
				Data:       op.Transaction.Data,
			},
			proof,
		)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("executing operation on Aptos mcms contract: %w", err)
		}
	} else {
		fmt.Println("Data is too large to execute in a single transaction, splitting into chunks")
		fmt.Println("Data length:", len(op.Transaction.Data))
		fmt.Println("Chunk size:", ChunkSizeBytes)
		// Split the data into chunks
		var chunks [][]byte
		for i := 0; i < len(op.Transaction.Data); i += ChunkSizeBytes {
			end := i + ChunkSizeBytes
			if end > len(op.Transaction.Data) {
				end = len(op.Transaction.Data)
			}
			chunks = append(chunks, op.Transaction.Data[i:end])
		}
		if len(chunks) == 0 {
			chunks = append(chunks, []byte{})
		}
		fmt.Println("Number of chunks:", len(chunks))
		for i, chunk := range chunks {
			if i == len(chunks)-1 {
				// Last chunk needs to call StageDataAndExecute
				// Also, add the proof to this last call
				maxGas := uint64(2_000_000)
				opts.MaxGasAmount = &maxGas
				tx, err = mcmsC.MCMSExecutor.StageDataAndExecute(
					opts,
					module_mcms.Op{
						ChainId:    *chainIDBig,
						Multisig:   mcmsAddress,
						Nonce:      uint64(nonce),
						To:         toAddress,
						ModuleName: additionalFields.ModuleName,
						Function:   additionalFields.Function,
						Data:       chunk,
					},
					proof,
				)
				if err != nil {
					return types.TransactionResult{}, fmt.Errorf("executing data chunk %v of %v on Aptos mcms contract: %w", i, len(chunks), err)
				}
				break
			}
			// All other chunks will be staged and executed with the last chunk
			tx, err := mcmsC.MCMSExecutor.StageData(
				opts,
				chunk,
				nil,
			)
			if err != nil {
				return types.TransactionResult{}, fmt.Errorf("staging data chunk %v of %v on Aptos mcms contract: %w", i, len(chunks), err)
			}
			_, err = e.client.WaitForTransaction(tx.Hash)
			if err != nil {
				return types.TransactionResult{}, fmt.Errorf("waiting for data chunk %v of %v on Aptos mcms contract: %w", i, len(chunks), err)
			}
		}
	}

	return types.TransactionResult{
		Hash:           tx.Hash,
		ChainFamily:    chain_selectors.FamilyAptos,
		RawTransaction: tx,
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

	mcmsC := mcms.Bind(mcmsAddress, e.client)
	opts := &bind.TransactOpts{Signer: e.auth}

	tx, err := mcmsC.MCMS.SetRoot(
		opts,
		root,
		uint64(validUntil),
		module_mcms.RootMetadata{
			ChainId:              *chainIDBig,
			Multisig:             mcmsAddress,
			PreOpCount:           metadata.StartingOpCount,
			PostOpCount:          metadata.StartingOpCount + e.TxCount,
			OverridePreviousRoot: e.OverridePreviousRoot,
		},
		proof,
		signatures,
	)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("setting root on Aptos mcms contract: %w", err)
	}

	return types.TransactionResult{
		Hash:           tx.Hash,
		ChainFamily:    chain_selectors.FamilyAptos,
		RawTransaction: tx,
	}, nil
}

func encodeSignatures(signatures []types.Signature) [][]byte {
	sigs := make([][]byte, len(signatures))
	for i, signature := range signatures {
		sigs[i] = append(signature.R.Bytes(), signature.S.Bytes()...)
		if signature.V <= SignatureVThreshold {
			sigs[i] = append(sigs[i], signature.V+SignatureVOffset)
		} else {
			sigs[i] = append(sigs[i], signature.V)
		}
	}

	return sigs
}
