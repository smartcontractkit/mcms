package mcms

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
	"github.com/spf13/cobra"
)

func buildDeployCmd(chainSelector uint64) *cobra.Command {
	return &cobra.Command{
		Use:   "set-root",
		Short: "Sets the Merkle Root on the MCM Contract",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get EVM chain ID
			chain, exists := chainsel.ChainBySelector(chainSelector)
			if !exists {
				return errors.New("chain not found")
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

			// Deploy MCM contract
			_, tx, instance, err := bindings.DeployManyChainMultiSig(auth, backend)
			if err != nil {
				return err
			}

			_, err = EVMConfirm(context.Background(), backend, tx.Hash().Hex())
			if err != nil {
				return err
			}
			fmt.Printf("Transaction %s confirmed\n", tx.Hash().Hex())
			fmt.Printf("MCMS Contract deployed at %s\n", instance.Address().Hex())
			return nil
		},
	}
}
