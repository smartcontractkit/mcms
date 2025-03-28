package aptos

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/api"
	"github.com/ethereum/go-ethereum/common"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-aptos/bindings/bind"
	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

const (
	SignatureVOffset    = 27
	SignatureVThreshold = 2

	ChunkSizeBytes = 50_000
)

var _ sdk.Executor = &Executor{}

type Executor struct {
	*Encoder
	*Inspector
	client aptos.AptosRpcClient
	auth   aptos.TransactionSigner

	bindingFn func(address aptos.AccountAddress, client aptos.AptosRpcClient) mcms.MCMS
}

func NewExecutor(client aptos.AptosRpcClient, auth aptos.TransactionSigner, encoder *Encoder) *Executor {
	return &Executor{
		Encoder:   encoder,
		Inspector: NewInspector(client),
		client:    client,
		auth:      auth,
		bindingFn: mcms.Bind,
	}
}

func (e Executor) ExecuteOperation(
	ctx context.Context,
	metadata types.ChainMetadata,
	nonce uint32,
	proof []common.Hash,
	op types.Operation,
) (types.TransactionResult, error) {
	mcmsAddress, err := hexToAddress(metadata.MCMAddress)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to parse MCMS address %q: %w", metadata.MCMAddress, err)
	}
	mcmsBinding := e.bindingFn(mcmsAddress, e.client)
	toAddress, err := hexToAddress(op.Transaction.To)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to parse To address %q: %w", op.Transaction.To, err)
	}
	var additionalFields AdditionalFields
	if err = json.Unmarshal(op.Transaction.AdditionalFields, &additionalFields); err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to unmarshal additional fields: %w", err)
	}
	chainID, err := chain_selectors.AptosChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return types.TransactionResult{}, err
	}
	chainIDBig := (&big.Int{}).SetUint64(chainID)
	proofBytes := make([][]byte, len(proof))
	for i, hash := range proof {
		proofBytes[i] = hash.Bytes()
	}

	opts := &bind.TransactOpts{Signer: e.auth}

	var tx *api.PendingTransaction
	if len(op.Transaction.Data) <= ChunkSizeBytes {
		tx, err = mcmsBinding.MCMS().Execute(
			opts,
			chainIDBig,
			mcmsAddress,
			uint64(nonce),
			toAddress,
			additionalFields.ModuleName,
			additionalFields.Function,
			op.Transaction.Data,
			proofBytes,
		)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("executing operation on Aptos mcms contract: %w", err)
		}
	} else {
		// Split the data into chunks
		var chunks [][]byte
		for chunk := range slices.Chunk(op.Transaction.Data, ChunkSizeBytes) {
			chunks = append(chunks, chunk)
		}
		if len(chunks) == 0 {
			chunks = append(chunks, []byte{})
		}
		// Managing the nonce we're sending with manually here, if we don't wait for transactions
		// to be mined before sending the next chunk, we'd run into a race condition with
		// sending transactions with the same sequence number twice.
		accountInfo, err := e.client.Account(e.auth.AccountAddress())
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("getting account info for %v: %w", e.auth.AccountAddress(), err)
		}
		startSeqNo, err := accountInfo.SequenceNumber()
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("getting nonce for %v: %w", e.auth.AccountAddress(), err)
		}
		for i, chunk := range chunks {
			//nolint:gosec
			seqNo := startSeqNo + uint64(i)
			opts.SequenceNumber = &seqNo
			if i == len(chunks)-1 {
				// Last chunk needs to call StageDataAndExecute
				// Also, add the proof to this last call
				tx, err = mcmsBinding.MCMSExecutor().StageDataAndExecute(
					opts,
					chainIDBig,
					mcmsAddress,
					uint64(nonce),
					toAddress,
					additionalFields.ModuleName,
					additionalFields.Function,
					chunk,
					proofBytes,
				)
				if err != nil {
					return types.TransactionResult{}, fmt.Errorf("executing data chunk %v of %v on Aptos mcms contract: %w", i, len(chunks), err)
				}

				break
			}
			// All other chunks will be staged and executed with the last chunk
			tx, err = mcmsBinding.MCMSExecutor().StageData(
				opts,
				chunk,
				nil,
			)
			if err != nil {
				return types.TransactionResult{}, fmt.Errorf("staging data chunk %v of %v on Aptos mcms contract: %w", i, len(chunks), err)
			}
		}
	}

	return types.TransactionResult{
		Hash:        tx.Hash,
		ChainFamily: chain_selectors.FamilyAptos,
		RawData:     tx,
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
	mcmsAddress, err := hexToAddress(metadata.MCMAddress)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to parse MCMS address %q: %w", metadata.MCMAddress, err)
	}
	mcmsBinding := e.bindingFn(mcmsAddress, e.client)
	chainID, err := chain_selectors.AptosChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return types.TransactionResult{}, err
	}
	chainIDBig := (&big.Int{}).SetUint64(chainID)

	proofBytes := make([][]byte, len(proof))
	for i, hash := range proof {
		proofBytes[i] = hash.Bytes()
	}
	signatures := encodeSignatures(sortedSignatures)

	opts := &bind.TransactOpts{Signer: e.auth}

	tx, err := mcmsBinding.MCMS().SetRoot(
		opts,
		root[:],
		uint64(validUntil),
		chainIDBig,
		mcmsAddress,
		metadata.StartingOpCount,
		metadata.StartingOpCount+e.TxCount,
		e.OverridePreviousRoot,
		proofBytes,
		signatures,
	)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("setting root on Aptos mcms contract: %w", err)
	}

	return types.TransactionResult{
		Hash:        tx.Hash,
		ChainFamily: chain_selectors.FamilyAptos,
		RawData:     tx,
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
