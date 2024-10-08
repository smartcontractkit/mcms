package merkle

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMerkleTree(t *testing.T) {
	t.Parallel()

	leaves := []common.Hash{
		crypto.Keccak256Hash([]byte("leaf1")),
		crypto.Keccak256Hash([]byte("leaf2")),
		crypto.Keccak256Hash([]byte("leaf3")),
		crypto.Keccak256Hash([]byte("leaf4")),
	}

	tree := NewMerkleTree(leaves)

	assert.NotNil(t, tree)
	assert.Len(t, tree.Layers, 2) // 4 leaves -> 2 intermediate layers + 1 root layer
	assert.Equal(t, tree.Root, common.HexToHash("0xbe80f348526b4646bc0697bf2fe649f1835863538924cb6b91ad4eb57ced0181"))
}

func TestNewMerkleTree_OddNumberOfLeaves(t *testing.T) {
	t.Parallel()

	leaves := []common.Hash{
		crypto.Keccak256Hash([]byte("leaf1")),
		crypto.Keccak256Hash([]byte("leaf2")),
		crypto.Keccak256Hash([]byte("leaf3")),
	}

	tree := NewMerkleTree(leaves)

	assert.NotNil(t, tree)
	assert.Len(t, tree.Layers, 2) // 3 leaves -> 2 intermediate layers + 1 root layer
	assert.Equal(t, tree.Root, common.HexToHash("0xbc3400d9b5f5f07751fe2d9a996880924186aac669555dd72b4ea02f1be7d73f"))
}

func TestNewMerkleTree_OddIntermediateLayer(t *testing.T) {
	t.Parallel()

	leaves := []common.Hash{
		crypto.Keccak256Hash([]byte("leaf1")),
		crypto.Keccak256Hash([]byte("leaf2")),
		crypto.Keccak256Hash([]byte("leaf3")),
		crypto.Keccak256Hash([]byte("leaf4")),
		crypto.Keccak256Hash([]byte("leaf5")),
	}

	tree := NewMerkleTree(leaves)

	assert.NotNil(t, tree)
	assert.Len(t, tree.Layers, 3) // 5 leaves -> 3 intermediate layers + 1 root layer
	assert.Equal(t, tree.Root, common.HexToHash("0xa949d6a972ac4f3447bdcae39d90951efacac97c831ec6f684881368e5adb8e6"))
}

func TestGetProof_EvenNumberOfLeaves(t *testing.T) {
	t.Parallel()

	leaves := []common.Hash{
		crypto.Keccak256Hash([]byte("leaf1")),
		crypto.Keccak256Hash([]byte("leaf2")),
		crypto.Keccak256Hash([]byte("leaf3")),
		crypto.Keccak256Hash([]byte("leaf4")),
	}

	tree := NewMerkleTree(leaves)

	for _, leaf := range leaves {
		proof, err := tree.GetProof(leaf)
		require.NoError(t, err)
		assert.Len(t, proof, 2) // Proof should contain 2 hashes for 4 leaves

		// Verify the proof
		computedHash := leaf
		for _, siblingHash := range proof {
			// Sort the pair of hashes before hashing
			computedHash = hashPair(computedHash, siblingHash)
		}
		assert.Equal(t, tree.Root, computedHash)
	}
}

func TestGetProof_OddNumberOfLeaves(t *testing.T) {
	t.Parallel()

	leaves := []common.Hash{
		crypto.Keccak256Hash([]byte("leaf1")),
		crypto.Keccak256Hash([]byte("leaf2")),
		crypto.Keccak256Hash([]byte("leaf3")),
	}

	tree := NewMerkleTree(leaves)

	for _, leaf := range leaves {
		proof, err := tree.GetProof(leaf)
		require.NoError(t, err)
		assert.Len(t, proof, 2) // Proof should contain 2 hashes for 4 leaves

		// Verify the proof
		computedHash := leaf
		for _, siblingHash := range proof {
			// Sort the pair of hashes before hashing
			computedHash = hashPair(computedHash, siblingHash)
		}
		assert.Equal(t, tree.Root, computedHash)
	}
}

func TestGetProof_OddIntermediateLayer(t *testing.T) {
	t.Parallel()

	leaves := []common.Hash{
		crypto.Keccak256Hash([]byte("leaf1")),
		crypto.Keccak256Hash([]byte("leaf2")),
		crypto.Keccak256Hash([]byte("leaf3")),
		crypto.Keccak256Hash([]byte("leaf4")),
		crypto.Keccak256Hash([]byte("leaf5")),
	}

	tree := NewMerkleTree(leaves)

	for _, leaf := range leaves {
		proof, err := tree.GetProof(leaf)
		require.NoError(t, err)
		assert.Len(t, proof, 3) // Proof should contain 3 hashes for 5 leaves

		// Verify the proof
		computedHash := leaf
		for _, siblingHash := range proof {
			// Sort the pair of hashes before hashing
			computedHash = hashPair(computedHash, siblingHash)
		}
		assert.Equal(t, tree.Root, computedHash)
	}
}

func TestGetProof_HashNotFound(t *testing.T) {
	t.Parallel()

	leaves := []common.Hash{
		crypto.Keccak256Hash([]byte("leaf1")),
		crypto.Keccak256Hash([]byte("leaf2")),
		crypto.Keccak256Hash([]byte("leaf3")),
	}

	tree := NewMerkleTree(leaves)
	nonExistentHash := crypto.Keccak256Hash([]byte("non-existent"))

	proof, err := tree.GetProof(nonExistentHash)
	require.Error(t, err)
	assert.Nil(t, proof)
	assert.IsType(t, &ErrMerkleTreeNodeNotFound{}, err)
}

func TestErrMerkleTreeNodeNotFound_Error(t *testing.T) {
	t.Parallel()

	hash := crypto.Keccak256Hash([]byte("non-existent"))
	err := &ErrMerkleTreeNodeNotFound{TargetHash: hash}

	expectedErrorMessage := "merkle tree does not contain hash: " + hash.String()
	assert.Equal(t, expectedErrorMessage, err.Error())
}
