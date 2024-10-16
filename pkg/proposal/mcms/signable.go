package mcms

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/mcms/pkg/merkle"
)

type Signable struct {
	Proposal       *MCMSProposal
	Tree           *merkle.MerkleTree
	MetadataHashes map[ChainIdentifier]common.Hash

	// Represents the hash and chain nonce of the transaction at
	// Proposal.Transactions[i]
	OperationHashes []common.Hash
	TxNonces        []uint64
}

func NewSignable(
	proposal *MCMSProposal,
	tree *merkle.MerkleTree,
	metadataHashes map[ChainIdentifier]common.Hash,
	operationHashes []common.Hash,
	txNonces []uint64,
) *Signable {
	return &Signable{
		Proposal:        proposal,
		Tree:            tree,
		MetadataHashes:  metadataHashes,
		OperationHashes: operationHashes,
		TxNonces:        txNonces,
	}
}

func (e *Signable) SigningHash() (common.Hash, error) {
	// Convert validUntil to [32]byte
	var validUntilBytes [32]byte
	binary.BigEndian.PutUint32(validUntilBytes[28:], e.Proposal.ValidUntil) // Place the uint32 in the last 4 bytes

	hashToSign := crypto.Keccak256Hash(e.Tree.Root.Bytes(), validUntilBytes[:])

	return toEthSignedMessageHash(hashToSign), nil
}

func (e *Signable) SigningMessage() ([]byte, error) {
	return ABIEncode(`[{"type":"bytes32"},{"type":"uint32"}]`, e.Tree.Root, e.Proposal.ValidUntil)
}

func toEthSignedMessageHash(messageHash common.Hash) common.Hash {
	// Add the Ethereum signed message prefix
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	data := append(prefix, messageHash.Bytes()...)

	// Hash the prefixed message
	return crypto.Keccak256Hash(data)
}
