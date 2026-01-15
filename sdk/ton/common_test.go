package ton_test

import (
	"github.com/smartcontractkit/mcms/internal/testutils"

	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tlbe"
)

func must[E any](out E, err error) E {
	if err != nil {
		panic(err)
	}

	return out
}

func AsUint160Addr(s testutils.ECDSASigner) *tlbe.Uint160 {
	return tlbe.NewUint160(s.Address().Big())
}
