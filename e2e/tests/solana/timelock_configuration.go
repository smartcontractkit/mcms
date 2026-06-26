//go:build e2e

package solanae2e

import (
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/accesscontroller"

	"github.com/smartcontractkit/mcms/sdk"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

var (
	testPDASeedTimelockGrantRole       = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'g', 'r', 'a', 'n', 't'}
	testPDASeedTimelockGrantRoleNoSend = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'g', 'r', 'n', 's'}
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

func (s *TestSuite) TestGrantRole() {
	ctx := s.T().Context()
	s.SetupTimelock(testPDASeedTimelockGrantRole, 1*time.Second)

	admin, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	target, err := solana.NewRandomPrivateKey()
	s.Require().NoError(err)

	timelockAddr := solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockGrantRole)
	role := sdk.TimelockRoleExecutor
	accessController := s.Roles[timelock.Executor_Role].AccessController.PublicKey()

	hasAccess, err := accesscontroller.HasAccess(ctx, s.SolanaClient, accessController, target.PublicKey(), rpc.CommitmentConfirmed)
	s.Require().NoError(err, "Failed to inspect initial role access")
	s.Require().False(hasAccess, "Target should not have role before GrantRole")

	configurer := solanasdk.NewTimelockConfigurer(s.SolanaClient, admin)
	result, err := configurer.GrantRole(ctx, timelockAddr, role, target.PublicKey().String())
	s.Require().NoError(err, "Failed to grant role")
	s.Require().NotEmpty(result.Hash, "Transaction hash should not be empty")
	s.Require().Equal(chainsel.FamilySolana, result.ChainFamily, "Chain family should be Solana")

	hasAccess, err = accesscontroller.HasAccess(ctx, s.SolanaClient, accessController, target.PublicKey(), rpc.CommitmentConfirmed)
	s.Require().NoError(err, "Failed to inspect granted role access")
	s.Require().True(hasAccess, "Target should have role after GrantRole")
}

func (s *TestSuite) TestGrantRoleNoSend() {
	ctx := s.T().Context()
	s.SetupTimelock(testPDASeedTimelockGrantRoleNoSend, 1*time.Second)

	admin, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	target, err := solana.NewRandomPrivateKey()
	s.Require().NoError(err)

	timelockAddr := solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockGrantRoleNoSend)
	role := sdk.TimelockRoleProposer
	accessController := s.Roles[timelock.Proposer_Role].AccessController.PublicKey()

	configurer := solanasdk.NewTimelockConfigurer(
		s.SolanaClient,
		admin,
		solanasdk.WithDoNotSendTimelockInstructionsOnChain(),
	)
	result, err := configurer.GrantRole(ctx, timelockAddr, role, target.PublicKey().String())
	s.Require().NoError(err, "Failed to prepare grant role transaction")
	s.Require().Empty(result.Hash, "Transaction hash should be empty when not sending")
	s.Require().Equal(chainsel.FamilySolana, result.ChainFamily, "Chain family should be Solana")

	tx, ok := result.RawData.(types.Transaction)
	s.Require().True(ok, "RawData should contain a Solana MCMS transaction")
	s.Require().Equal(s.TimelockProgramID.String(), tx.To)
	s.Require().Equal("RBACTimelock", tx.OperationMetadata.ContractType)
	s.Require().Equal([]string{"RBACTimelock", "GrantRole"}, tx.OperationMetadata.Tags)

	hasAccess, err := accesscontroller.HasAccess(ctx, s.SolanaClient, accessController, target.PublicKey(), rpc.CommitmentConfirmed)
	s.Require().NoError(err, "Failed to inspect role access")
	s.Require().False(hasAccess, "NoSend should not broadcast GrantRole transaction")
}
