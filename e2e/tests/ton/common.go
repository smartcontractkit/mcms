//go:build e2e
// +build e2e

package tone2e

import (
	"math/big"

	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"
	"github.com/xssnick/tonutils-go/address"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ccip/bindings/common"
)

const (
	EnvPathContracts = "PATH_CONTRACTS_TON"

	PathContractsMCMS     = "mcms.MCMS.compiled.json"
	PathContractsTimelock = "mcms.RBACTimelock.compiled.json"
)

// TODO: duplicated utils with unit tests [START]

func must[E any](out E, err error) E {
	if err != nil {
		panic(err)
	}
	return out
}

// TODO: duplicated utils with unit tests [END]

func MCMSContractDataFrom(owner *address.Address, chainId int64) mcms.Data {
	return mcms.Data{
		ID: 4,
		Ownable: common.Ownable2Step{
			Owner:        owner,
			PendingOwner: nil,
		},
		Oracle:  tvm.ZeroAddress,
		Signers: must(tvm.MakeDict(map[*big.Int]mcms.Signer{}, tvm.KeyUINT256)),
		Config: mcms.Config{
			Signers:      must(tvm.MakeDictFrom([]mcms.Signer{}, tvm.KeyUINT8)),
			GroupQuorums: must(tvm.MakeDictFrom([]mcms.GroupQuorum{}, tvm.KeyUINT8)),
			GroupParents: must(tvm.MakeDictFrom([]mcms.GroupParent{}, tvm.KeyUINT8)),
		},
		SeenSignedHashes: must(tvm.MakeDict(map[*big.Int]mcms.SeenSignedHash{}, tvm.KeyUINT256)),
		RootInfo: mcms.RootInfo{
			ExpiringRootAndOpCount: mcms.ExpiringRootAndOpCount{
				Root:       big.NewInt(0),
				ValidUntil: 0,
				OpCount:    0,
				OpPendingInfo: mcms.OpPendingInfo{
					ValidAfter:             0,
					OpFinalizationTimeout:  0,
					OpPendingReceiver:      tvm.ZeroAddress,
					OpPendingBodyTruncated: big.NewInt(0),
				},
			},
			RootMetadata: mcms.RootMetadata{
				ChainID:              big.NewInt(chainId),
				MultiSig:             tvm.ZeroAddress,
				PreOpCount:           0,
				PostOpCount:          0,
				OverridePreviousRoot: false,
			},
		},
	}
}
