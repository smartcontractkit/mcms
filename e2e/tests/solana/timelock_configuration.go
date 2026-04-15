//go:build e2e

package solanae2e

import (
	"time"

	"github.com/gagliardetto/solana-go"

	chainsel "github.com/smartcontractkit/chain-selectors"

	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
)

func (s *TestSuite) TestUpdateDelay() {
	ctx := s.T().Context()
	initialDelay := 1 * time.Second
	s.SetupTimelock(testPDASeedTimelockUpdateDelay, initialDelay)

	admin, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	timelockAddr := solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockUpdateDelay)
	configurer := solanasdk.NewTimelockConfigurer(s.SolanaClient, admin)

	delay, err := configurer.GetMinDelay(ctx, timelockAddr)
	s.Require().NoError(err, "Failed to get initial min delay")
	s.Require().Equal(uint64(initialDelay.Seconds()), delay, "Initial delay should match configured value")

	newDelay := uint64(120)
	result, err := configurer.UpdateDelay(ctx, timelockAddr, newDelay)
	s.Require().NoError(err, "Failed to update delay")
	s.Require().NotEmpty(result.Hash, "Transaction hash should not be empty")
	s.Require().Equal(chainsel.FamilySolana, result.ChainFamily, "Chain family should be Solana")

	delay, err = configurer.GetMinDelay(ctx, timelockAddr)
	s.Require().NoError(err, "Failed to get updated min delay")
	s.Require().Equal(newDelay, delay, "Delay should match the updated value")
}
