package merkle

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMerkleTree tests the creation of a new Merkle tree
func TestNewMerkleTree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		leaves         []common.Hash
		expectedLayers int
		expectedRoot   common.Hash
	}{
		{
			name: "Even number of leaves",
			leaves: []common.Hash{
				crypto.Keccak256Hash([]byte("leaf1")),
				crypto.Keccak256Hash([]byte("leaf2")),
				crypto.Keccak256Hash([]byte("leaf3")),
				crypto.Keccak256Hash([]byte("leaf4")),
			},
			expectedLayers: 2,
			expectedRoot:   common.HexToHash("0xbe80f348526b4646bc0697bf2fe649f1835863538924cb6b91ad4eb57ced0181"),
		},
		{
			name:           "Empty tree",
			leaves:         []common.Hash{},
			expectedLayers: 0,
			expectedRoot:   common.Hash{},
		},
		{
			name: "Odd number of leaves",
			leaves: []common.Hash{
				crypto.Keccak256Hash([]byte("leaf1")),
				crypto.Keccak256Hash([]byte("leaf2")),
				crypto.Keccak256Hash([]byte("leaf3")),
			},
			expectedLayers: 2,
			expectedRoot:   common.HexToHash("0xbc3400d9b5f5f07751fe2d9a996880924186aac669555dd72b4ea02f1be7d73f"),
		},
		{
			name: "Odd intermediate layer",
			leaves: []common.Hash{
				crypto.Keccak256Hash([]byte("leaf1")),
				crypto.Keccak256Hash([]byte("leaf2")),
				crypto.Keccak256Hash([]byte("leaf3")),
				crypto.Keccak256Hash([]byte("leaf4")),
				crypto.Keccak256Hash([]byte("leaf5")),
			},
			expectedLayers: 3,
			expectedRoot:   common.HexToHash("0xa949d6a972ac4f3447bdcae39d90951efacac97c831ec6f684881368e5adb8e6"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tree := NewMerkleTree(tt.leaves)

			assert.NotNil(t, tree)
			assert.Len(t, tree.Layers, tt.expectedLayers)
			assert.Equal(t, tt.expectedRoot, tree.Root)
		})
	}
}

// TestGetProof tests the generation of Merkle proofs, including handling of non-existent hashes
func TestGetProof(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		leaves           []common.Hash
		expectedProofLen int
		nonExistentHash  common.Hash // If nonExistentHash is nil, the test case will skip the non-existent hash check
	}{
		{
			name: "Even number of leaves",
			leaves: []common.Hash{
				crypto.Keccak256Hash([]byte("leaf1")),
				crypto.Keccak256Hash([]byte("leaf2")),
				crypto.Keccak256Hash([]byte("leaf3")),
				crypto.Keccak256Hash([]byte("leaf4")),
			},
			expectedProofLen: 2,
			nonExistentHash:  crypto.Keccak256Hash([]byte("non-existent")),
		},
		{
			name: "Odd number of leaves",
			leaves: []common.Hash{
				crypto.Keccak256Hash([]byte("leaf1")),
				crypto.Keccak256Hash([]byte("leaf2")),
				crypto.Keccak256Hash([]byte("leaf3")),
			},
			expectedProofLen: 2,
			nonExistentHash:  crypto.Keccak256Hash([]byte("non-existent")),
		},
		{
			name: "Odd intermediate layer",
			leaves: []common.Hash{
				crypto.Keccak256Hash([]byte("leaf1")),
				crypto.Keccak256Hash([]byte("leaf2")),
				crypto.Keccak256Hash([]byte("leaf3")),
				crypto.Keccak256Hash([]byte("leaf4")),
				crypto.Keccak256Hash([]byte("leaf5")),
			},
			expectedProofLen: 3,
			nonExistentHash:  crypto.Keccak256Hash([]byte("non-existent")),
		},
		{
			name: "Non-existent hash only",
			leaves: []common.Hash{
				crypto.Keccak256Hash([]byte("leaf1")),
				crypto.Keccak256Hash([]byte("leaf2")),
				crypto.Keccak256Hash([]byte("leaf3")),
			},
			expectedProofLen: 0,
			nonExistentHash:  crypto.Keccak256Hash([]byte("non-existent")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tree := NewMerkleTree(tt.leaves)

			// Test valid proofs if expectedProofLen is greater than 0
			if tt.expectedProofLen > 0 {
				for _, leaf := range tt.leaves {
					proof, err := tree.GetProof(leaf)
					require.NoError(t, err)
					assert.Len(t, proof, tt.expectedProofLen)

					// Verify the proof
					computedHash := leaf
					for _, siblingHash := range proof {
						computedHash = hashPair(computedHash, siblingHash)
					}
					assert.Equal(t, tree.Root, computedHash)
				}
			}

			// Test for non-existent hash if provided
			proof, err := tree.GetProof(tt.nonExistentHash)
			require.Error(t, err)
			assert.Nil(t, proof)

			var merkleErr *TreeNodeNotFoundError
			assert.ErrorAs(t, err, &merkleErr)
		})
	}
}

// TestErrMerkleTreeNodeNotFound tests different TreeNodeNotFoundError error messages
func TestErrMerkleTreeNodeNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		targetHash       common.Hash
		expectedErrorMsg string
	}{
		{
			name:             "Non-existent hash",
			targetHash:       crypto.Keccak256Hash([]byte("non-existent")),
			expectedErrorMsg: "merkle tree does not contain hash: " + crypto.Keccak256Hash([]byte("non-existent")).String(),
		},
		{
			name:             "Empty hash",
			targetHash:       common.Hash{},
			expectedErrorMsg: "merkle tree does not contain hash: " + common.Hash{}.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := &TreeNodeNotFoundError{TargetHash: tt.targetHash}
			assert.Equal(t, tt.expectedErrorMsg, err.Error())
		})
	}
}

// TestGetProofs tests the GetProofs method for different tree configurations
func TestGetProofs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		leaves      []common.Hash
		expectError bool
		expectedLen int
	}{
		{
			name: "Even number of leaves",
			leaves: []common.Hash{
				crypto.Keccak256Hash([]byte("leaf1")),
				crypto.Keccak256Hash([]byte("leaf2")),
				crypto.Keccak256Hash([]byte("leaf3")),
				crypto.Keccak256Hash([]byte("leaf4")),
			},
			expectError: false,
			expectedLen: 4, // 4 leaves, should return 4 proofs
		},
		{
			name: "Odd number of leaves",
			leaves: []common.Hash{
				crypto.Keccak256Hash([]byte("leaf1")),
				crypto.Keccak256Hash([]byte("leaf2")),
				crypto.Keccak256Hash([]byte("leaf3")),
			},
			expectError: false,
			expectedLen: 3, // 3 leaves, should return 3 proofs
		},
		{
			name: "Single leaf",
			leaves: []common.Hash{
				crypto.Keccak256Hash([]byte("leaf1")),
			},
			expectError: true, // Single leaf should not be possible to generate a proof for
			expectedLen: 0,    // 1 leaf, should return 1 proof (empty)
		},
		{
			name:        "Empty leaves",
			leaves:      []common.Hash{}, // No leaves provided
			expectError: true,            // Should error since tree has no layers
			expectedLen: 0,               // No leaves, should return 0 proofs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create Merkle tree with given leaves
			tree := NewMerkleTree(tt.leaves)

			// Call GetProofs to get proofs for all leaves
			proofs, err := tree.GetProofs()

			// Check if an error was expected
			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, proofs)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, proofs)
				assert.Len(t, proofs, tt.expectedLen)

				// Verify proofs for each leaf
				for leaf, proof := range proofs {
					computedHash := leaf

					// Special case for single leaf: the proof should be empty and the leaf itself should be the root
					if len(tt.leaves) == 1 {
						// Check that the proof is empty
						assert.Empty(t, proof)
						// The leaf should be the root
						assert.Equal(t, tree.Root, leaf)
					} else {
						// Recompute the root using the proof
						for _, siblingHash := range proof {
							computedHash = hashPair(computedHash, siblingHash)
						}
						// The computed root should match the tree's root
						assert.Equal(t, tree.Root, computedHash)
					}
				}
			}
		})
	}
}
