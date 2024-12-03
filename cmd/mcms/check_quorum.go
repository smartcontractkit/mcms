package mcms

import (
	"errors"
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
	"github.com/spf13/cobra"
)

func buildMCMSCheckQuorumCmd(proposalPath string, chainSelector uint64) *cobra.Command {
	return &cobra.Command{
		Use:   "check-quorum",
		Short: "Determines whether the provided signatures meet the quorum to set the root",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Checking quorum for proposal %s\n", proposalPath)
			proposal, err := mcms.LoadProposal(proposalPath)
			if err != nil {
				fmt.Printf("Error loading proposal: %s\n", err)
				return err
			}

			// Get EVM chain ID
			chain, exists := chainsel.ChainBySelector(chainSelector)
			if !exists {
				return errors.New("chain not found")
			}

			// Load RPCs
			backends, err := loadRPCs([]types.ChainSelector{types.ChainSelector(chain.Selector)})
			if err != nil {
				return err
			}

			// Create Inspector
			chainInspector := evm.NewInspector(backends[types.ChainSelector(chain.Selector)])

			// Convert proposal to signable
			s, err := proposal.Signable(map[types.ChainSelector]sdk.Inspector{
				types.ChainSelector(chain.Selector): chainInspector,
			})
			if err != nil {
				fmt.Printf("Error converting proposal to signable: %s\n", err)
				return err
			}
			quorumMet, err := s.CheckQuorum(types.ChainSelector(chainSelector))
			if err != nil {
				fmt.Printf("Error checking quorum: %s\n", err)
				return err
			}
			if quorumMet {
				fmt.Printf("Signature Quorum met!")
			} else {
				fmt.Printf("Signature Quorum not met!")
				return errors.New("signature Quorum not met")
			}
			return nil
		},
	}
}
