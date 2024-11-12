package abi

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// ABIEncode is the equivalent of abi.encode.
// We are using this as a global util because the signing process requires encoding / decoding the data.
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
// We are using this as a global util because the signing process requires encoding / decoding the data.
// See a full set of examples https://github.com/ethereum/go-ethereum/blob/420b78659bef661a83c5c442121b13f13288c09f/accounts/abi/packing_test.go#L31
func ABIDecode(abiStr string, data []byte) ([]any, error) {
	inDef := fmt.Sprintf(`[{ "name" : "method", "type": "function", "outputs": %s}]`, abiStr)
	inAbi, err := abi.JSON(strings.NewReader(inDef))
	if err != nil {
		return nil, err
	}

	return inAbi.Unpack("method", data)
}
