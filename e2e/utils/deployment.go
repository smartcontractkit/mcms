package testutils

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/suite"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
)

// DeployMCMSContract is a helper to deploy the MCMS contract
func DeployMCMSContract(s *suite.Suite, client *ethclient.Client, auth *bind.TransactOpts, signerAddresses []common.Address) *bindings.ManyChainMultiSig {
	_, tx, instance, err := bindings.DeployManyChainMultiSig(auth, client)
	s.Require().NoError(err, "Failed to deploy contract")

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Set configurations
	signerGroups := []uint8{0, 1}   // Two groups: Group 0 and Group 1
	groupQuorums := [32]uint8{1, 1} // Quorum 1 for both groups
	groupParents := [32]uint8{0, 0} // Group 0 is its own parent; Group 1's parent is Group 0
	clearRoot := true

	tx, err = instance.SetConfig(
		auth,
		signerAddresses,
		signerGroups,
		groupQuorums,
		groupParents,
		clearRoot,
	)
	s.Require().NoError(err, "Failed to set contract configuration")
	receipt, err = bind.WaitMined(context.Background(), client, tx)
	s.Require().NoError(err, "Failed to mine configuration transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	return instance
}

// DeployTimelockContract is a helper to deploy the Timelock contract
func DeployTimelockContract(s *suite.Suite, client *ethclient.Client, auth *bind.TransactOpts, adminAddress string) *bindings.RBACTimelock {
	_, tx, instance, err := bindings.DeployRBACTimelock(
		auth,
		client,
		big.NewInt(0),
		common.HexToAddress(adminAddress),
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
	)
	s.Require().NoError(err, "Failed to deploy contract")

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	return instance
}
