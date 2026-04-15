//go:build e2e

package evme2e

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	chainsel "github.com/smartcontractkit/chain-selectors"

	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/sdk/evm"
)

func (s *TimelockInspectionTestSuite) TestUpdateDelay() {
	ctx := s.T().Context()

	timelockContract := testutils.DeployTimelockContract(&s.Suite, s.ClientA, s.auth, s.publicKey.String())
	addr := timelockContract.Address().Hex()

	configurer := evm.NewTimelockConfigurer(s.ClientA, s.auth)

	delay, err := configurer.GetMinDelay(ctx, addr)
	s.Require().NoError(err, "Failed to get initial min delay")
	s.Require().EqualValues(0, delay)

	newDelay := uint64(120)
	result, err := configurer.UpdateDelay(ctx, addr, newDelay)
	s.Require().NoError(err, "Failed to update delay")
	s.Require().NotEmpty(result.Hash, "Transaction hash should not be empty")
	s.Require().Equal(chainsel.FamilyEVM, result.ChainFamily, "Chain family should be EVM")

	receipt, err := testutils.WaitMinedWithTxHash(ctx, s.ClientA, common.HexToHash(result.Hash))
	s.Require().NoError(err, "Failed to wait for transaction to be mined")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status, "Transaction was not successful")

	delay, err = configurer.GetMinDelay(ctx, addr)
	s.Require().NoError(err, "Failed to get updated min delay")
	s.Require().Equal(newDelay, delay, "Delay should match the updated value")
}
