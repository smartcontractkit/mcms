package mcms

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
	"github.com/spf13/cobra"
)

func buildSetConfigCmd(chainSelector uint64) *cobra.Command {
	var (
		configPath string
		address    string
		clearRoot  bool
	)

	cmd := &cobra.Command{
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

			// Load config
			config, err := loadConfig(configPath)
			if err != nil {
				return err
			}

			// Set config
			configurer := evm.NewConfigurer(backend, auth)
			txHash, err := configurer.SetConfig(address, config, clearRoot)
			if err != nil {
				return err
			}

			// Confirm transaction
			_, err = EVMConfirm(context.Background(), backend, txHash)
			if err != nil {
				return err
			}
			fmt.Printf("Transaction %s confirmed\n", txHash)
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "Path to the config file")
	cmd.Flags().StringVar(&address, "address", "", "Address to set the config on")
	cmd.Flags().BoolVar(&clearRoot, "clearRoot", false, "Clear the root after setting the config")

	return cmd
}
