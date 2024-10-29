package merkle

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// HashPairSize defines the size of hash pairs when computing parent nodes.
const HashPairSize = 2

// Tree represents a cryptographic Merkle tree used to verify data integrity.
type Tree struct {
	// Root is the final hash at the top of the Merkle tree, derived from the hashes of all the
	// leaf nodes.
	Root common.Hash

	// Layers contains all tree layers, starting from the leaves. Each subsequent layer is derived
	// by hashing pairs of nodes from the previous layer, ultimately leading to the root.
	Layers [][]common.Hash
}

// NewTree constructs a Merkle tree from a list of leaf hashes.
// It recursively hashes pairs of leaves until a single root hash is obtained.
func NewTree(leaves []common.Hash) *Tree {
	layers := make([][]common.Hash, 0)
	if len(leaves) == 0 {
		return &Tree{
			Root:   common.Hash{},
			Layers: layers,
		}
	}

	currHashes := leaves
	for len(currHashes) > 1 {
		// Duplicate the last hash if the number of current hashes is odd.
		if len(currHashes)%2 != 0 {
			currHashes = append(currHashes, currHashes[len(currHashes)-1])
		}

		// Append the current layer of hashes to the layers.
		layers = append(layers, currHashes)

		// Create a list of parent hashes by hashing pairs of the current hashes.
		tempHashes := make([]common.Hash, len(currHashes)/HashPairSize)
		for i := 0; i < len(currHashes); i += HashPairSize {
			tempHashes[i/HashPairSize] = hashPair(currHashes[i], currHashes[i+1])
		}

		// Move up to the next layer (parent hashes).
		currHashes = tempHashes
	}

	// Return the Merkle tree with the computed layers and root hash.
	return &Tree{
		Root:   currHashes[0],
		Layers: layers,
	}
}

// GetProof generates a Merkle proof for a given leaf hash.
// A proof is a set of sibling hashes needed to reconstruct the root from this leaf.
func (t *Tree) GetProof(hash common.Hash) ([]common.Hash, error) {
	proof := make([]common.Hash, 0)
	targetHash := hash

	// Traverse each layer of the tree.
	for i := range t.Layers {
		found := false

		for j, h := range t.Layers[i] {
			if h != targetHash {
				continue
			}

			// Append the sibling hash to the proof.
			siblingIdx := j ^ 1
			siblingHash := t.Layers[i][siblingIdx]
			proof = append(proof, siblingHash)

			// Hash the target hash with its sibling to get the parent hash for the next layer.
			targetHash = hashPair(targetHash, siblingHash)

			found = true

			break
		}

		// Return an error if the hash is not found in the current layer (shouldn't happen).
		if !found {
			return nil, NewTreeNodeNotFoundError(targetHash)
		}
	}

	return proof, nil
}

// GetProofs generates Merkle proofs for all leaves in the tree.
// It returns a map where the keys are the leaf hashes and the values are their corresponding proofs.
func (t *Tree) GetProofs() (map[common.Hash][]common.Hash, error) {
	proofs := make(map[common.Hash][]common.Hash)
	if len(t.Layers) == 0 {
		return nil, ErrNoLayers
	}

	// General case: iterate over all leaves in the first layer
	for _, leaf := range t.Layers[0] {
		proof, err := t.GetProof(leaf)
		if err != nil {
			return nil, err
		}

		proofs[leaf] = proof
	}

	return proofs, nil
}

// TreeNodeNotFoundError indicates that a target hash could not be found in the tree.
type TreeNodeNotFoundError struct {
	// TargetHash is the hash that couldn't be found in the tree.
	TargetHash common.Hash
}

// NewTreeNodeNotFoundError creates a new TreeNodeNotFoundError with the given target hash.
func NewTreeNodeNotFoundError(targetHash common.Hash) *TreeNodeNotFoundError {
	return &TreeNodeNotFoundError{
		TargetHash: targetHash,
	}
}

// Error implements the error interface for TreeNodeNotFoundError.
func (e *TreeNodeNotFoundError) Error() string {
	return "merkle tree does not contain hash: " + e.TargetHash.String()
}

// ErrNoLayers indicates that the Merkle tree has no layers.
var ErrNoLayers = errors.New("no layers in the Merkle tree")

// hashPair takes two hashes and returns their sorted combined hash.
// Sorting ensures deterministic results regardless of input order.
func hashPair(a, b common.Hash) common.Hash {
	if a.Cmp(b) < 0 {
		return efficientHash(a, b)
	}

	return efficientHash(b, a)
}

// efficientHash combines two hashes and computes their Keccak256 hash.
func efficientHash(a, b common.Hash) common.Hash {
	// Create a buffer of 64 bytes to hold both hashes (32 bytes each).
	var combinedHash [64]byte

	// Copy the first hash into the first 32 bytes.
	copy(combinedHash[:32], a[:])

	// Copy the second hash into the next 32 bytes.
	copy(combinedHash[32:], b[:])

	// Compute and return the Keccak256 hash of the combined data.
	return crypto.Keccak256Hash(combinedHash[:])
}
