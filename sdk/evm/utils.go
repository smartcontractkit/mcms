package evm

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	chainselremote "github.com/smartcontractkit/chain-selectors/remote"

	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
)

const (
	SignatureVOffset    = 27
	SignatureVThreshold = 2

	// SimulatedEVMChainID is the chain ID used for simulated chains.
	SimulatedEVMChainID = 1337
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

// getEVMChainID returns the EVM chain ID for the given chain selector.
//
// To support simulated chains in testing, the isSim flag can be set to true. Simulated chains
// always have EVM chain ID of 1337. We need to override the chain ID for setRoot to execute and
// not throw WrongChainId.
func getEVMChainID(ctx context.Context, sel types.ChainSelector, isSim bool) (uint64, error) {
	if isSim {
		return SimulatedEVMChainID, nil
	}

	evmChain, exists, err := chainselremote.EvmChainBySelector(ctx, uint64(sel), chainselremote.WithFallbackToLocal(true))
	if err != nil || !exists {
		return 0, &sdkerrors.InvalidChainIDError{
			ReceivedChainID: sel,
		}
	}

	return evmChain.EvmChainID, nil
}
