//go:build e2e

package tone2e

import (
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/xssnick/tonutils-go/tlb"

	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tracetracking"

	mcmston "github.com/smartcontractkit/mcms/sdk/ton"
)

func (s *TimelockInspectionTestSuite) TestUpdateDelay() {
	ctx := s.T().Context()

	inspector := mcmston.NewTimelockInspector(s.TonClient)
	delay, err := inspector.GetMinDelay(ctx, s.timelockAddr.String())
	s.Require().NoError(err)
	s.Require().EqualValues(0, delay)

	configurer := mcmston.NewTimelockConfigurer(s.wallet, mcmston.DefaultSendAmount)
	newDelay := uint64(120)

	result, err := configurer.UpdateDelay(ctx, s.timelockAddr.String(), newDelay)
	s.Require().NoError(err)
	s.Require().NotEmpty(result.Hash)
	s.Require().Equal(chainsel.FamilyTon, result.ChainFamily)

	tx, ok := result.RawData.(*tlb.Transaction)
	s.Require().True(ok, "TON UpdateDelay should return the sent transaction")

	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)

	delay, err = inspector.GetMinDelay(ctx, s.timelockAddr.String())
	s.Require().NoError(err)
	s.Require().Equal(newDelay, delay)
}
