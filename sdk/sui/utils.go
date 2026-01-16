package sui

import (
	"encoding/hex"
	"errors"
	"strings"
)

const AddressLen = 32

// TODO: this interface should come from chainlink-sui when available
type SuiSigner interface {
	// Sign signs the given message and returns the serialized signature.
	Sign(message []byte) ([]string, error)

	// GetAddress returns the Sui address derived from the signer's public key
	GetAddress() (string, error)
}

type Address [AddressLen]uint8

func AddressFromHex(str string) (*Address, error) {
	if strings.HasPrefix(str, "0x") || strings.HasPrefix(str, "0X") {
		str = str[2:]
	}
	if len(str)%2 != 0 {
		str = "0" + str
	}
	data, err := hex.DecodeString(str)
	if err != nil {
		return nil, err
	}
	if len(data) > AddressLen {
		return nil, errors.New("address length exceeds 32 bytes")
	}
	var address Address
	copy(address[AddressLen-len(data):], data[:])

	return &address, nil
}

func (a Address) Bytes() []byte {
	return a[:]
}

func (a Address) Hex() string {
	return hex.EncodeToString(a[:])
}
