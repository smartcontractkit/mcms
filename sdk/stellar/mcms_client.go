package stellar

import (
	"github.com/smartcontractkit/chainlink-stellar/bindings"
	stellarmcms "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"
)

// newMCMSClient returns a McmsClient for mcmAddr (hex or contract strkey), using invoker for RPC.
func newMCMSClient(invoker bindings.Invoker, mcmAddr string) (*stellarmcms.McmsClient, error) {
	id, err := normalizeContractIDStrkey(mcmAddr)
	if err != nil {
		return nil, err
	}

	return stellarmcms.NewMcmsClient(invoker, id), nil
}
