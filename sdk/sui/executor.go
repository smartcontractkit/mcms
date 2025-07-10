package sui

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/ethereum/go-ethereum/common"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	module_mcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
	bindutils "github.com/smartcontractkit/chainlink-sui/bindings/utils"

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
	client        sui.ISuiAPI
	signer        bindutils.SuiSigner
	mcmsPackageId string
	mcms          module_mcms.IMcms
}

func NewExecutor(client sui.ISuiAPI, signer bindutils.SuiSigner, encoder *Encoder, mcmsPackageId string, role TimelockRole) (*Executor, error) {
	mcms, err := module_mcms.NewMcms(mcmsPackageId, client)
	if err != nil {
		return nil, err
	}

	inspector, err := NewInspector(client, signer, mcmsPackageId, role)
	if err != nil {
		return nil, err
	}

	return &Executor{
		Encoder:       encoder,
		Inspector:     inspector,
		client:        client,
		signer:        signer,
		mcmsPackageId: mcmsPackageId,
		mcms:          mcms,
	}, nil
}

func (e Executor) ExecuteOperation(
	ctx context.Context,
	metadata types.ChainMetadata,
	nonce uint32,
	proof []common.Hash,
	op types.Operation,
) (types.TransactionResult, error) {
	var additionalFields AdditionalFields
	if err := json.Unmarshal(op.Transaction.AdditionalFields, &additionalFields); err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to unmarshal additional fields: %w", err)
	}

	var additionalFieldsMetadata AdditionalFieldsMetadata
	if len(metadata.AdditionalFields) > 0 {
		if err := json.Unmarshal(metadata.AdditionalFields, &additionalFieldsMetadata); err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to unmarshal additional fields metadata: %w", err)
		}
	}

	chainID, err := chain_selectors.SuiChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return types.TransactionResult{}, err
	}
	chainIDBig := big.NewInt(int64(chainID))

	proofBytes := make([][]byte, len(proof))
	for i, hash := range proof {
		proofBytes[i] = hash.Bytes()
	}

	opts := &bind.CallOpts{
		Signer: e.signer,
	}

	stateObj := bind.Object{Id: metadata.MCMAddress}
	clockObj := bind.Object{Id: "0x6"} // Clock object ID in Sui

	tx, err := e.mcms.Execute(
		ctx,
		opts,
		stateObj,
		clockObj,
		additionalFieldsMetadata.Role.Byte(),
		chainIDBig,
		common.FromHex(metadata.MCMAddress),
		uint64(nonce),
		common.FromHex(op.Transaction.To),
		additionalFields.ModuleName,
		additionalFields.Function,
		op.Transaction.Data,
		proofBytes,
	)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("executing operation on Sui mcms contract: %w", err)
	}

	return types.TransactionResult{
		Hash:        tx.Digest,
		ChainFamily: chain_selectors.FamilySui,
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
	var additionalFieldsMetadata AdditionalFieldsMetadata
	if len(metadata.AdditionalFields) > 0 {
		if err := json.Unmarshal(metadata.AdditionalFields, &additionalFieldsMetadata); err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to unmarshal additional fields metadata: %w", err)
		}
	}

	chainID, err := chain_selectors.SuiChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return types.TransactionResult{}, err
	}
	chainIDBig := big.NewInt(int64(chainID))

	proofBytes := make([][]byte, len(proof))
	for i, hash := range proof {
		proofBytes[i] = hash.Bytes()
	}
	signatures := encodeSignatures(sortedSignatures)

	opts := &bind.CallOpts{
		Signer: e.signer,
	}

	stateObj := bind.Object{Id: metadata.MCMAddress}
	clockObj := bind.Object{Id: "0x6"} // Clock object ID in Sui

	tx, err := e.mcms.SetRoot(
		ctx,
		opts,
		stateObj,
		clockObj,
		additionalFieldsMetadata.Role.Byte(),
		root[:],
		uint64(validUntil),
		chainIDBig,
		common.FromHex(metadata.MCMAddress),
		metadata.StartingOpCount,
		metadata.StartingOpCount+e.TxCount,
		e.OverridePreviousRoot,
		proofBytes,
		signatures,
	)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("setting root on Sui mcms contract: %w", err)
	}

	return types.TransactionResult{
		Hash:        tx.Digest,
		ChainFamily: chain_selectors.FamilySui,
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
