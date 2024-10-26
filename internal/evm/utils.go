package evm

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/internal/core/config"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
	"github.com/smartcontractkit/mcms/internal/evm/bindings"
)

const EthereumSignatureVOffset = 27
const EthereumSignatureVThreshold = 2

func ToGethSignature(s mcms.Signature) bindings.ManyChainMultiSigSignature {
	if s.V < EthereumSignatureVThreshold {
		s.V += EthereumSignatureVOffset
	}

	return bindings.ManyChainMultiSigSignature{
		R: [32]byte(s.R.Bytes()),
		S: [32]byte(s.S.Bytes()),
		V: s.V,
	}
}

type ContractDeployBackend interface {
	bind.ContractBackend
	bind.DeployBackend
}

func transformMCMAddresses(metadatas map[mcms.ChainSelector]mcms.ChainMetadata) map[mcms.ChainSelector]common.Address {
	m := make(map[mcms.ChainSelector]common.Address)
	for k, v := range metadatas {
		m[k] = common.HexToAddress(v.MCMAddress)
	}

	return m
}

func transformSignatures(signatures []mcms.Signature) []bindings.ManyChainMultiSigSignature {
	sigs := make([]bindings.ManyChainMultiSigSignature, len(signatures))
	for i, sig := range signatures {
		sigs[i] = ToGethSignature(sig)
	}

	return sigs
}

func transformMCMSConfigs(configs map[mcms.ChainSelector]bindings.ManyChainMultiSigConfig) (map[mcms.ChainSelector]*config.Config, error) {
	m := make(map[mcms.ChainSelector]*config.Config)
	for k, v := range configs {
		configurator := EVMConfigurator{}
		cfg, err := configurator.ToConfig(v)
		if err != nil {
			return nil, err
		}
		m[k] = cfg
	}

	return m, nil
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
