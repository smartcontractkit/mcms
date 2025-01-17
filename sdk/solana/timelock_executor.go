package solana

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockExecutor = (*TimelockExecutor)(nil)

// Executor is an Executor implementation for Solana chains, allowing for the execution of
// operations on the MCMS contract
type TimelockExecutor struct {
	*TimelockInspector
	client *rpc.Client
	auth   solana.PrivateKey
}

// NewTimelockExecutor creates a new TimelockExecutor for Solana chains
func NewTimelockExecutor(client *rpc.Client, auth solana.PrivateKey) *TimelockExecutor {
	return &TimelockExecutor{
		TimelockInspector: NewTimelockInspector(client),
		client:            client,
		auth:              auth,
	}
}

func (e *TimelockExecutor) Client() *rpc.Client {
	return e.client
}

func (e *TimelockExecutor) AuthPublicKey() solana.PublicKey {
	return e.auth.PublicKey()
}

func (e *TimelockExecutor) Execute(
	ctx context.Context, bop types.BatchOperation, timelockAddress string, predecessor common.Hash, salt common.Hash,
) (string, error) {
	return "", fmt.Errorf("not implemented")
}
