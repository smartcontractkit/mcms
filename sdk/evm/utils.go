package evm

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

const (
	EthereumSignatureVOffset    = 27
	EthereumSignatureVThreshold = 2
)

type ContractDeployBackend interface {
	bind.ContractBackend
	bind.DeployBackend
}

// TransformHashes transforms a slice of common.Hash to a slice of [32]byte.
func TransformHashes(hashes []common.Hash) [][32]byte {
	bs := make([][32]byte, 0, len(hashes))
	for _, h := range hashes {
		bs = append(bs, [32]byte(h))
	}

	return bs
}

// TransformSignatures transforms a slice of types.Signature to a slice of
// bindings.ManyChainMultiSigSignature.
func TransformSignatures(signatures []types.Signature) []bindings.ManyChainMultiSigSignature {
	sigs := make([]bindings.ManyChainMultiSigSignature, 0, len(signatures))
	for _, sig := range signatures {
		sigs = append(sigs, toGethSignature(sig))
	}

	return sigs
}

// toGethSignature converts a types.Signature to a bindings.ManyChainMultiSigSignature.
func toGethSignature(s types.Signature) bindings.ManyChainMultiSigSignature {
	if s.V < EthereumSignatureVThreshold {
		s.V += EthereumSignatureVOffset
	}

	return bindings.ManyChainMultiSigSignature{
		R: [32]byte(s.R.Bytes()),
		S: [32]byte(s.S.Bytes()),
		V: s.V,
	}
}

// ABIEncode is the equivalent of abi.encode.
// See a full set of examples https://github.com/ethereum/go-ethereum/blob/420b78659bef661a83c5c442121b13f13288c09f/accounts/abi/packing_test.go#L31
func ABIEncode(abiStr string, values ...any) ([]byte, error) {
	// Create a dummy method with arguments
	inDef := fmt.Sprintf(`[{ "name" : "method", "type": "function", "inputs": %s}]`, abiStr)
	inAbi, err := abi.JSON(strings.NewReader(inDef))
	if err != nil {
		return nil, err
	}

	res, err := inAbi.Pack("method", values...)
	if err != nil {
		return nil, err
	}

	return res[4:], nil
}

// ABIDecode is the equivalent of abi.decode.
// See a full set of examples https://github.com/ethereum/go-ethereum/blob/420b78659bef661a83c5c442121b13f13288c09f/accounts/abi/packing_test.go#L31
func ABIDecode(abiStr string, data []byte) ([]any, error) {
	inDef := fmt.Sprintf(`[{ "name" : "method", "type": "function", "outputs": %s}]`, abiStr)
	inAbi, err := abi.JSON(strings.NewReader(inDef))
	if err != nil {
		return nil, err
	}

	return inAbi.Unpack("method", data)
}
