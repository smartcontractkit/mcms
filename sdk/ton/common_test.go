package ton_test

import (
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tlbe"
	"github.com/smartcontractkit/mcms/internal/testutils"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

// TODO: duplicated utils with e2e tests [START]

func must[E any](out E, err error) E {
	if err != nil {
		panic(err)
	}

	return out
}

func makeRandomTestWallet(api wallet.TonAPI, networkGlobalID int32) (*wallet.Wallet, error) {
	v5r1Config := wallet.ConfigV5R1Final{
		NetworkGlobalID: networkGlobalID,
		Workchain:       0,
	}

	return wallet.FromSeed(api, wallet.NewSeed(), v5r1Config)
}

// TODO: duplicated utils with e2e tests [END]

func AsUint160Addr(s testutils.ECDSASigner) *tlbe.Uint160 {
	return tlbe.NewUint160(s.Address().Big())
}
