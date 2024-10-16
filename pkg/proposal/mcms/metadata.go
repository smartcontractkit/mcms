package mcms

import (
	"context"
	"errors"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/mcms/pkg/gethwrappers"
)

var MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA = crypto.Keccak256Hash([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA"))

type ChainMetadata struct {
	StartingOpCount uint64 `json:"startingOpCount"`
	MCMAddress      string `json:"mcmAddress"`
}

type MetadataEncoder interface {
	Hash(metadata ChainMetadata, txCount uint64, overridePreviousRoot bool) (common.Hash, error)
}

type MetadataExecutor interface {
	Execute(
		metadata ChainMetadata,
		txCount uint64,
		overridePreviousRoot bool,
		root [32]byte,
		validUntil uint32,
		signingHash common.Hash,
		signatures []Signature,
		proof []common.Hash,
	) error
}

type EVMMetadataEncoder struct {
	ChainId uint64
}

func (e *EVMMetadataEncoder) Hash(metadata ChainMetadata, txCount uint64, overridePreviousRoot bool) (common.Hash, error) {
	metadataObj := gethwrappers.ManyChainMultiSigRootMetadata{
		ChainId:              new(big.Int).SetUint64(e.ChainId),
		MultiSig:             common.HexToAddress(metadata.MCMAddress),
		PreOpCount:           new(big.Int).SetUint64(metadata.StartingOpCount),
		PostOpCount:          new(big.Int).SetUint64(metadata.StartingOpCount + txCount),
		OverridePreviousRoot: overridePreviousRoot,
	}

	abi := `[{"type":"bytes32"},{"type":"tuple","components":[{"name":"chainId","type":"uint256"},{"name":"multiSig","type":"address"},{"name":"preOpCount","type":"uint40"},{"name":"postOpCount","type":"uint40"},{"name":"overridePreviousRoot","type":"bool"}]}]`
	encoded, err := ABIEncode(abi, MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA, metadataObj)
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(encoded), nil
}

type EVMMetadataExecutor struct {
	EVMMetadataEncoder
	client ContractDeployBackend
	auth   *bind.TransactOpts
}

func (e *EVMMetadataExecutor) Execute(
	metadata ChainMetadata,
	txCount uint64,
	overridePreviousRoot bool,
	root [32]byte,
	validUntil uint32,
	signingHash common.Hash,
	signatures []Signature,
	proof []common.Hash,
) error {
	metadataObj := gethwrappers.ManyChainMultiSigRootMetadata{
		ChainId:              new(big.Int).SetUint64(e.ChainId),
		MultiSig:             common.HexToAddress(metadata.MCMAddress),
		PreOpCount:           new(big.Int).SetUint64(metadata.StartingOpCount),
		PostOpCount:          new(big.Int).SetUint64(metadata.StartingOpCount + txCount),
		OverridePreviousRoot: overridePreviousRoot,
	}

	mcms, err := gethwrappers.NewManyChainMultiSig(common.HexToAddress(metadata.MCMAddress), e.client)
	if err != nil {
		return err
	}

	// Sort signatures by recovered address
	sortedSignatures := signatures
	sort.Slice(sortedSignatures, func(i, j int) bool {
		recoveredSignerA, _ := sortedSignatures[i].Recover(signingHash)
		recoveredSignerB, _ := sortedSignatures[j].Recover(signingHash)

		return recoveredSignerA.Cmp(recoveredSignerB) < 0
	})

	tx, err := mcms.SetRoot(
		e.auth,
		[32]byte(root),
		validUntil,
		metadataObj,
		transformHashes(proof),
		transformSignatures(sortedSignatures),
	)

	receipt, err := bind.WaitMined(context.Background(), e.client, tx)
	if err != nil {
		return err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return errors.New("transaction failed")
	}

	return nil
}
