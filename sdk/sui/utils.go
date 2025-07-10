package sui

import (
	"encoding/hex"
	"errors"
	"strings"
)

const AddressLen = 32

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
