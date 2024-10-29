package mcms

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/pkg/errors"
	"github.com/smartcontractkit/mcms/pkg/gethwrappers"
)

func TestCalculateTransactionCounts(t *testing.T) {
	t.Parallel()

	transactions := []ChainOperation{
		{ChainSelector: TestChain1},
		{ChainSelector: TestChain1},
		{ChainSelector: TestChain2},
	}

	expected := map[ChainSelector]uint64{
		TestChain1: 2,
		TestChain2: 1,
	}

	result := calculateTransactionCounts(transactions)
	assert.Equal(t, expected, result)
}

func TestBuildRootMetadatas_Success(t *testing.T) {
	t.Parallel()

	chainMetadata := map[ChainSelector]ChainMetadata{
		TestChain1: {MCMAddress: common.HexToAddress("0x1"), StartingOpCount: 0},
		TestChain2: {MCMAddress: common.HexToAddress("0x2"), StartingOpCount: 3},
	}
	txCounts := map[ChainSelector]uint64{
		TestChain1: 2,
		TestChain2: 1,
	}

	expected := map[ChainSelector]gethwrappers.ManyChainMultiSigRootMetadata{
		TestChain1: {
			ChainId:              new(big.Int).SetUint64(uint64(1337)),
			MultiSig:             common.HexToAddress("0x1"),
			PreOpCount:           big.NewInt(0),
			PostOpCount:          big.NewInt(2),
			OverridePreviousRoot: true,
		},
		TestChain2: {
			ChainId:              new(big.Int).SetUint64(11155111),
			MultiSig:             common.HexToAddress("0x2"),
			PreOpCount:           big.NewInt(3),
			PostOpCount:          big.NewInt(4),
			OverridePreviousRoot: true,
		},
	}

	result, err := buildRootMetadatas(chainMetadata, txCounts, true, false)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestBuildRootMetadatas_InvalidChainID(t *testing.T) {
	t.Parallel()

	chainMetadata := map[ChainSelector]ChainMetadata{
		0: {MCMAddress: common.HexToAddress("0x1"), StartingOpCount: 0},
	}
	txCounts := map[ChainSelector]uint64{
		0: 1,
	}

	result, err := buildRootMetadatas(chainMetadata, txCounts, true, false)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.IsType(t, &errors.InvalidChainIDError{}, err)
}

func TestBuildOperations(t *testing.T) {
	t.Parallel()

	transactions := []ChainOperation{
		{ChainSelector: TestChain1,
			Operation: Operation{
				To: common.HexToAddress("0x1"), Data: common.Hex2Bytes("0x"), Value: big.NewInt(1),
			},
		},
		{ChainSelector: TestChain1,
			Operation: Operation{
				To: common.HexToAddress("0x2"), Data: common.Hex2Bytes("0x"), Value: big.NewInt(2),
			},
		},
		{ChainSelector: TestChain2,
			Operation: Operation{
				To: common.HexToAddress("0x3"), Data: common.Hex2Bytes("0x"), Value: big.NewInt(3),
			},
		},
	}
	rootMetadatas := map[ChainSelector]gethwrappers.ManyChainMultiSigRootMetadata{
		TestChain1: {
			ChainId:    new(big.Int).SetUint64(uint64(1337)),
			MultiSig:   common.HexToAddress("0x1"),
			PreOpCount: big.NewInt(0),
		},
		TestChain2: {
			ChainId:    new(big.Int).SetUint64(uint64(11155111)),
			MultiSig:   common.HexToAddress("0x2"),
			PreOpCount: big.NewInt(0),
		},
	}
	txCounts := map[ChainSelector]uint64{
		TestChain1: 2,
		TestChain2: 1,
	}

	expected := map[ChainSelector][]gethwrappers.ManyChainMultiSigOp{
		TestChain1: {
			{
				ChainId:  new(big.Int).SetUint64(uint64(1337)),
				MultiSig: common.HexToAddress("0x1"),
				Nonce:    big.NewInt(0),
				To:       common.HexToAddress("0x1"),
				Data:     common.FromHex("0x"),
				Value:    big.NewInt(1),
			},
			{
				ChainId:  new(big.Int).SetUint64(uint64(1337)),
				MultiSig: common.HexToAddress("0x1"),
				Nonce:    big.NewInt(1),
				To:       common.HexToAddress("0x2"),
				Data:     common.FromHex("0x"),
				Value:    big.NewInt(2),
			},
		},
		TestChain2: {
			{
				ChainId:  new(big.Int).SetUint64(uint64(11155111)),
				MultiSig: common.HexToAddress("0x2"),
				Nonce:    big.NewInt(0),
				To:       common.HexToAddress("0x3"),
				Data:     common.FromHex("0x"),
				Value:    big.NewInt(3),
			},
		},
	}

	result, _ := buildOperations(transactions, rootMetadatas, txCounts)
	assert.Equal(t, expected, result)
}

func TestSortedChainSelectors(t *testing.T) {
	t.Parallel()

	chainMetadata := map[ChainSelector]ChainMetadata{
		TestChain2: {},
		TestChain1: {},
		TestChain3: {},
	}

	expected := []ChainSelector{TestChain1, TestChain3, TestChain2}

	result := sortedChainSelectors(chainMetadata)
	assert.Equal(t, expected, result)
}

func TestBuildMerkleTree(t *testing.T) {
	t.Parallel()

	selectors := []ChainSelector{TestChain1, TestChain2}
	ops := map[ChainSelector][]gethwrappers.ManyChainMultiSigOp{
		TestChain1: {
			{
				ChainId:  new(big.Int).SetUint64(uint64(1337)),
				MultiSig: common.HexToAddress("0x1"),
				Nonce:    big.NewInt(0),
				To:       common.HexToAddress("0x1"),
				Data:     common.FromHex("0x"),
				Value:    big.NewInt(1),
			},
		},
		TestChain2: {
			{
				ChainId:  new(big.Int).SetUint64(uint64(11155111)),
				MultiSig: common.HexToAddress("0x2"),
				Nonce:    big.NewInt(0),
				To:       common.HexToAddress("0x2"),
				Data:     common.FromHex("0x"),
				Value:    big.NewInt(2),
			},
		},
	}
	rootMetadatas := map[ChainSelector]gethwrappers.ManyChainMultiSigRootMetadata{
		TestChain1: {
			ChainId:              big.NewInt(1),
			MultiSig:             common.HexToAddress("0x1"),
			PreOpCount:           big.NewInt(0),
			PostOpCount:          big.NewInt(1),
			OverridePreviousRoot: false,
		},
		TestChain2: {
			ChainId:              big.NewInt(2),
			MultiSig:             common.HexToAddress("0x2"),
			PreOpCount:           big.NewInt(0),
			PostOpCount:          big.NewInt(1),
			OverridePreviousRoot: false,
		},
	}

	tree, err := buildMerkleTree(selectors, rootMetadatas, ops)
	require.NoError(t, err)
	assert.NotNil(t, tree)
	assert.NotEmpty(t, tree.Root)
}

func TestMetadataEncoder(t *testing.T) {
	t.Parallel()

	rootMetadata := gethwrappers.ManyChainMultiSigRootMetadata{
		ChainId:              new(big.Int).SetUint64(uint64(1337)),
		MultiSig:             common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		PreOpCount:           big.NewInt(0),
		PostOpCount:          big.NewInt(1),
		OverridePreviousRoot: true,
	}

	hash, err := metadataEncoder(rootMetadata)
	require.NoError(t, err)
	assert.Equal(t, common.HexToHash("0xc38c406774af2c0a887d4793f40712670e8833c6d71251fdb4f8251b6e0c96e5"), hash)
}

func TestTxEncoder(t *testing.T) {
	t.Parallel()

	op := gethwrappers.ManyChainMultiSigOp{
		ChainId:  new(big.Int).SetUint64(uint64(1337)),
		MultiSig: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		Nonce:    big.NewInt(1),
		To:       common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcdef"),
		Value:    big.NewInt(1000),
		Data:     []byte("data"),
	}

	hash, err := txEncoder(op)
	require.NoError(t, err)
	assert.Equal(t, common.HexToHash("0xea87ccae6f56402661aca3f9119809f710068ad47a8b6bf5376fbe25b989d28a"), hash)
}
