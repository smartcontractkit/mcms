package sdk

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

type SolanaAdditionalFields struct {
	// MSIGName is a differentiator/seed for supporting
	// multiple multisigs with a single deployed program
	// only applicable to Solana
	MSIGName []byte
}

type AddrMetadata struct {
	MCMAddress             string
	SolanaAdditionalFields SolanaAdditionalFields
}

// Inspector is an interface for inspecting on chain state of MCMS contracts.
type Inspector interface {
	GetConfig(ctx context.Context, addr AddrMetadata) (*types.Config, error)
	GetOpCount(ctx context.Context, addr AddrMetadata) (uint64, error)
	GetRoot(ctx context.Context, addr AddrMetadata) (common.Hash, uint32, error)
	GetRootMetadata(ctx context.Context, addr AddrMetadata) (types.ChainMetadata, error)
}
