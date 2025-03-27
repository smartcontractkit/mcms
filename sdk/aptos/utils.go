package aptos

import (
	"github.com/aptos-labs/aptos-go-sdk"
)

func pointerTo[T any](v T) *T {
	return &v
}

func hexToAddress(address string) (aptos.AccountAddress, error) {
	addr := aptos.AccountAddress{}
	if err := addr.ParseStringRelaxed(address); err != nil {
		return aptos.AccountAddress{}, err
	}
	return addr, nil
}
