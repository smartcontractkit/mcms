package evm

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

const (
	SignatureVOffset    = 27
	SignatureVThreshold = 2
)

type ContractDeployBackend interface {
	bind.ContractBackend
	bind.DeployBackend
}

// transformHashes transforms a slice of common.Hash to a slice of [32]byte.
func transformHashes(hashes []common.Hash) [][32]byte {
	bs := make([][32]byte, 0, len(hashes))
	for _, h := range hashes {
		bs = append(bs, [32]byte(h))
	}

	return bs
}

// transformSignatures transforms a slice of types.Signature to a slice of
// bindings.ManyChainMultiSigSignature.
func transformSignatures(signatures []types.Signature) []bindings.ManyChainMultiSigSignature {
	sigs := make([]bindings.ManyChainMultiSigSignature, 0, len(signatures))
	for _, sig := range signatures {
		sigs = append(sigs, toGethSignature(sig))
	}

	return sigs
}

// toGethSignature converts a types.Signature to a bindings.ManyChainMultiSigSignature.
func toGethSignature(s types.Signature) bindings.ManyChainMultiSigSignature {
	if s.V < SignatureVThreshold {
		s.V += SignatureVOffset
	}

	return bindings.ManyChainMultiSigSignature{
		R: [32]byte(s.R.Bytes()),
		S: [32]byte(s.S.Bytes()),
		V: s.V,
	}
}
