package evm

import (
	"github.com/smartcontractkit/ccip-owner-contracts/gethwrappers"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
	"github.com/smartcontractkit/mcms/internal/evm/bindings"
)

func ToGethSignature(s mcms.Signature) bindings.ManyChainMultiSigSignature {
	if s.V < EthereumSignatureVThreshold {
		s.V += EthereumSignatureVOffset
	}

	return gethwrappers.ManyChainMultiSigSignature{
		R: [32]byte(s.R.Bytes()),
		S: [32]byte(s.S.Bytes()),
		V: s.V,
	}
}
