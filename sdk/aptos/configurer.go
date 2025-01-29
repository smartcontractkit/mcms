package aptos

import (
	"context"
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk"

	aptosutil "github.com/smartcontractkit/mcms/e2e/utils/aptos"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Configurer = &Configurer{}

type Configurer struct {
	client *aptos.NodeClient
	auth   *aptos.Account
}

func NewConfigurer(client *aptos.NodeClient, auth *aptos.Account) *Configurer {
	return &Configurer{
		client: client,
		auth:   auth,
	}
}

func (c Configurer) SetConfig(ctx context.Context, mcmAddr string, cfg *types.Config, clearRoot bool) (string, error) {
	groupQuorum, groupParents, signerAddresses, signerGroups, err := evm.ExtractSetConfigInputs(cfg)
	if err != nil {
		return "", fmt.Errorf("unable to extract set config inputs: %w", err)
	}
	// TODO this is a const in the mcms.move contract (MAX_NUM_SIGNERS), make it available somewhere
	if len(signerAddresses) > 200 {
		return "", fmt.Errorf("too many signers (max 200)")
	}

	// Configure contract
	payload, err := aptosutil.BuildTransactionPayload(
		mcmAddr+"::mcms::set_config",
		nil,
		[]string{
			"vector<vector<u8>>",
			"vector<u8>",
			"vector<u8>",
			"vector<u8>",
			"bool",
		},
		[]any{
			signerAddresses,
			signerGroups,
			groupQuorum,
			groupParents,
			clearRoot,
		},
	)
	if err != nil {
		return "", fmt.Errorf("unable to build mcms::set_config payload: %w", err)
	}
	data, err := aptosutil.BuildSignSubmitAndWaitForTransaction(c.client, c.auth, payload)
	if err != nil {
		return "", fmt.Errorf("setting config on Aptos mcms contract: %w", err)
	}

	found := false
	for _, event := range data.Events {
		if event.Type == mcmAddr+"::mcms::ConfigSet" {
			if config, ok := event.Data["config"]; ok {
				_ = config
				// e.Logger.Debugw("✅ Set config on MCMS contract", "config", config)
				found = true
			}
		}
	}
	if !found {
		return "", fmt.Errorf("unable to find config event on Aptos mcms contract")
	}

	return data.Hash, nil
}
