package mcms

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
	"github.com/spf13/cobra"
)

func buildSetRootCmd(proposalPath string, chainSelector uint64) *cobra.Command {
	return &cobra.Command{
		Use:   "set-root",
		Short: "Sets the Merkle Root on the MCM Contract",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			encoders, err := proposal.GetEncoders()
			if err != nil {
				fmt.Printf("Error getting encoders: %s\n", err)
				return err
			}

			// Load Private Key
			pk, err := loadPrivateKey()
			if err != nil {
				fmt.Printf("Error loading private key: %s\n", err)
				return err
			}

			auth, err := bind.NewKeyedTransactorWithChainID(pk, big.NewInt(int64(chain.EvmChainID)))
			if err != nil {
				return err
			}

			// Load RPCs
			backends, err := loadRPCs([]types.ChainSelector{types.ChainSelector(chain.Selector)})
			if err != nil {
				return err
			}
			backend := backends[types.ChainSelector(chain.Selector)]

			// Create Executor
			chainExecutor := evm.NewExecutor(
				encoders[types.ChainSelector(chain.Selector)].(*evm.Encoder),
				backend,
				auth,
			)

			// Convert proposal to executor
			e, err := proposal.Executable(map[types.ChainSelector]sdk.Executor{
				types.ChainSelector(chain.Selector): chainExecutor,
			})
			if err != nil {
				fmt.Printf("Error converting proposal to executor: %s\n", err)
				return err
			}

			// Set the root on chain
			txHash, err := e.SetRoot(types.ChainSelector(chainSelector))
			if err != nil {
				return err
			}

			_, err = EVMConfirm(context.Background(), backend, txHash)
			if err != nil {
				return err
			}
			return nil
		},
	}
}
