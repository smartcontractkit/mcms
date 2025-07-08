package sui

import (
	"context"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Configurer = &Configurer{}

type Configurer struct {
	client aptos.AptosRpcClient
	auth   aptos.TransactionSigner
	role   TimelockRole

	bindingFn func(address aptos.AccountAddress, client aptos.AptosRpcClient) mcms.MCMS
}

func NewConfigurer(client aptos.AptosRpcClient, auth aptos.TransactionSigner, role TimelockRole) *Configurer {
	return &Configurer{
		client:    client,
		auth:      auth,
		role:      role,
		bindingFn: mcms.Bind,
	}
}

func (c Configurer) SetConfig(ctx context.Context, mcmsAddr string, cfg *types.Config, clearRoot bool) (types.TransactionResult, error) {

	return types.TransactionResult{}, nil
}
