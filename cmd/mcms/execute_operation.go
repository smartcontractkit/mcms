package mcms

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
	"github.com/spf13/cobra"
)

func buildExecuteOperationCmd(proposalPath string, chainSelector uint64) *cobra.Command {
	var index uint64

	cmd := cobra.Command{
		Use:   "execute-operation",
		Short: "Executes all operations for a given chain in an MCMS Proposal. Root must be set first.",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			proposal, err := loadProposal(proposalPath)
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

			txHash, err := e.Execute(int(index))
			if err != nil {
				return err
			}

			fmt.Printf("Transaction sent: %s", txHash)
			_, err = EVMConfirm(context.Background(), backend, txHash)
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().Uint64Var(&index, "index", 0, "Index of the operation to execute")

	return &cmd
}
