package mcms

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/spf13/cast"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/pkg/config"
	"github.com/smartcontractkit/mcms/pkg/gethwrappers"
)

type ContractDeployBackend interface {
	bind.ContractBackend
	bind.DeployBackend
}

func transformMCMAddresses(metadatas map[ChainIdentifier]ChainMetadata) map[ChainIdentifier]common.Address {
	m := make(map[ChainIdentifier]common.Address)
	for k, v := range metadatas {
		m[k] = v.MCMAddress
	}

	return m
}

func transformSignatures(signatures []Signature) []gethwrappers.ManyChainMultiSigSignature {
	sigs := make([]gethwrappers.ManyChainMultiSigSignature, len(signatures))
	for i, sig := range signatures {
		sigs[i] = sig.ToGethSignature()
	}

	return sigs
}

func transformHashes(hashes []common.Hash) [][32]byte {
	m := make([][32]byte, len(hashes))
	for i, h := range hashes {
		m[i] = [32]byte(h)
	}

	return m
}

func transformMCMSConfigs(configs map[ChainIdentifier]gethwrappers.ManyChainMultiSigConfig) (map[ChainIdentifier]*config.Config, error) {
	m := make(map[ChainIdentifier]*config.Config)
	for k, v := range configs {
		cfg, err := config.NewConfigFromRaw(v)
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

// FromFile generic function to read a file and unmarshal its contents into the provided struct
func FromFile(filePath string, out any) error {
	// Load file from path
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Unmarshal JSON into the provided struct
	err = json.Unmarshal(fileBytes, out)
	if err != nil {
		return err
	}

	return nil
}

// WriteProposalToFile writes a proposal to the provided file path
func WriteProposalToFile(proposal any, filePath string) error {
	proposalBytes, err := json.Marshal(proposal)
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, proposalBytes, 0600)
	if err != nil {
		return err
	}

	return nil
}

// SafeCastIntToUint32 safely converts an int to uint32 using cast and checks for overflow
func SafeCastIntToUint32(value int) (uint32, error) {
	if value < 0 || value > math.MaxUint32 {
		return 0, fmt.Errorf("value %d exceeds uint32 range", value)
	}

	return cast.ToUint32E(value)
}

// SafeCastInt64ToUint32 safely converts an int64 to uint32 and checks for overflow and negative values
func SafeCastInt64ToUint32(value int64) (uint32, error) {
	// Check if the value is negative
	if value < 0 {
		return 0, fmt.Errorf("value %d is negative and cannot be converted to uint32", value)
	}

	// Check if the value exceeds the maximum value of uint32
	if value > int64(math.MaxUint32) {
		return 0, fmt.Errorf("value %d exceeds uint32 range", value)
	}

	// Use cast to convert value to uint32 safely
	return cast.ToUint32E(value)
}

// SafeCastUint64ToUint8 safely converts an int to uint8 using cast and checks for overflow
func SafeCastUint64ToUint8(value uint64) (uint8, error) {
	if value > math.MaxUint8 {
		return 0, fmt.Errorf("value %d exceeds uint8 range", value)
	}

	return cast.ToUint8E(value)
}
