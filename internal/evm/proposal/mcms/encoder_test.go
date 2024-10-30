package evm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
	"github.com/smartcontractkit/mcms/internal/evm/bindings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEVMEncoder_HashOperation(t *testing.T) {
	encoder := NewEVMEncoder(5, 1, false)

	metadata := mcms.ChainMetadata{
		StartingOpCount: 0,
		MCMAddress:      "0x1",
	}

	operation := NewEVMOperation(common.HexToAddress("0x2"), []byte("data"), new(big.Int).SetUint64(1000000000000000000), "", []string{})

	hash, err := encoder.HashOperation(0, metadata, mcms.ChainOperation{
		ChainSelector: mcms.ChainSelector(chain_selectors.EvmChainIdToChainSelector()[1]),
		Operation:     operation,
	})
	require.NoError(t, err)
	expectedHash := "0x4bd3162db447382fc6e5b2a8eb24e19e901feecb90c5071bac5deee3cb58cb97" // Replace with actual expected hash value
	assert.Equal(t, expectedHash, hash.Hex())
}

func TestEVMEncoder_HashMetadata(t *testing.T) {
	encoder := NewEVMEncoder(5, 1, false)

	metadata := mcms.ChainMetadata{
		StartingOpCount: 0,
		MCMAddress:      "0x1",
	}

	hash, err := encoder.HashMetadata(metadata)
	require.NoError(t, err)
	expectedHash := "0x3d030bfa3fcbbfa780ab87e0368f9487a271df7654cc2d1ba82f3be9d4933366" // Replace with actual expected hash value
	assert.Equal(t, expectedHash, hash.Hex())
}

func TestEVMEncoder_ToGethOperation(t *testing.T) {
	encoder := NewEVMEncoder(5, 1, false)

	metadata := mcms.ChainMetadata{
		StartingOpCount: 0,
		MCMAddress:      "0x1",
	}

	operation := NewEVMOperation(common.HexToAddress("0x2"), []byte("data"), new(big.Int).SetUint64(1000000000000000000), "", []string{})

	op, err := encoder.ToGethOperation(0, metadata, mcms.ChainOperation{
		ChainSelector: mcms.ChainSelector(chain_selectors.EvmChainIdToChainSelector()[1]),
		Operation:     operation,
	})
	require.NoError(t, err)

	expectedOp := bindings.ManyChainMultiSigOp{
		ChainId:  new(big.Int).SetUint64(1),
		MultiSig: common.HexToAddress("0x1"),
		Nonce:    new(big.Int).SetUint64(0),
		To:       common.HexToAddress("0x2"),
		Data:     []byte("data"),
		Value:    new(big.Int).SetUint64(1000000000000000000),
	}

	assert.Equal(t, expectedOp, op)
}

func TestEVMEncoder_ToGethRootMetadata(t *testing.T) {
	encoder := NewEVMEncoder(5, 1, false)

	metadata := mcms.ChainMetadata{
		StartingOpCount: 0,
		MCMAddress:      "0x1",
	}

	rootMetadata := encoder.ToGethRootMetadata(metadata)

	expectedRootMetadata := bindings.ManyChainMultiSigRootMetadata{
		ChainId:              new(big.Int).SetUint64(1),
		MultiSig:             common.HexToAddress("0x1"),
		PreOpCount:           new(big.Int).SetUint64(0),
		PostOpCount:          new(big.Int).SetUint64(5),
		OverridePreviousRoot: false,
	}

	assert.Equal(t, expectedRootMetadata, rootMetadata)
}
