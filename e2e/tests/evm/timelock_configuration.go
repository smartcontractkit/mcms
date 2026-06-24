//go:build e2e

package evme2e

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	chainsel "github.com/smartcontractkit/chain-selectors"

	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
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

func (s *TimelockInspectionTestSuite) TestGrantRole() {
	ctx := s.T().Context()

	timelockContract := testutils.DeployTimelockContract(&s.Suite, s.ClientA, s.auth, s.publicKey.String())
	addr := timelockContract.Address().Hex()
	target := s.signerAddresses[0]
	role := sdk.TimelockRoleExecutor
	roleHash, err := role.Hash()
	s.Require().NoError(err)

	hasRole, err := timelockContract.HasRole(&bind.CallOpts{Context: ctx}, [32]byte(roleHash), target)
	s.Require().NoError(err, "Failed to inspect initial role")
	s.Require().False(hasRole, "Target should not have role before GrantRole")

	configurer := evm.NewTimelockConfigurer(s.ClientA, s.auth)
	result, err := configurer.GrantRole(ctx, addr, role, target.Hex())
	s.Require().NoError(err, "Failed to grant role")
	s.Require().NotEmpty(result.Hash, "Transaction hash should not be empty")
	s.Require().Equal(chainsel.FamilyEVM, result.ChainFamily, "Chain family should be EVM")

	rawTx, ok := result.RawData.(*types.Transaction)
	s.Require().True(ok, "RawData should contain an EVM transaction")
	s.Require().Equal(timelockContract.Address(), *rawTx.To())

	receipt, err := testutils.WaitMinedWithTxHash(ctx, s.ClientA, common.HexToHash(result.Hash))
	s.Require().NoError(err, "Failed to wait for transaction to be mined")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status, "Transaction was not successful")

	hasRole, err = timelockContract.HasRole(&bind.CallOpts{Context: ctx}, [32]byte(roleHash), target)
	s.Require().NoError(err, "Failed to inspect granted role")
	s.Require().True(hasRole, "Target should have role after GrantRole")
}

func (s *TimelockInspectionTestSuite) TestGrantRoleNoSend() {
	ctx := s.T().Context()

	timelockContract := testutils.DeployTimelockContract(&s.Suite, s.ClientA, s.auth, s.publicKey.String())
	addr := timelockContract.Address().Hex()
	role := sdk.TimelockRoleProposer
	roleHash, err := role.Hash()
	s.Require().NoError(err)
	target := s.signerAddresses[0]

	noSendAuth := *s.auth
	noSendAuth.NoSend = true
	configurer := evm.NewTimelockConfigurer(s.ClientA, &noSendAuth)
	result, err := configurer.GrantRole(ctx, addr, role, target.Hex())
	s.Require().NoError(err, "Failed to prepare role grant transactions")
	s.Require().NotEmpty(result.Hash, "Transaction hash should not be empty")
	s.Require().Equal(chainsel.FamilyEVM, result.ChainFamily, "Chain family should be EVM")

	timelockABI, err := bindings.RBACTimelockMetaData.GetAbi()
	s.Require().NoError(err)
	grantRole := timelockABI.Methods["grantRole"]

	rawTx, ok := result.RawData.(*types.Transaction)
	s.Require().True(ok, "RawData should contain an EVM transaction")
	s.Require().Equal(timelockContract.Address(), *rawTx.To())
	s.Require().NotEmpty(rawTx.Data())
	s.Require().GreaterOrEqual(len(rawTx.Data()), 4)
	s.Require().Equal(grantRole.ID, rawTx.Data()[:4])

	decoded, err := grantRole.Inputs.Unpack(rawTx.Data()[4:])
	s.Require().NoError(err)
	s.Require().Len(decoded, 2)
	s.Require().Equal([32]byte(roleHash), decoded[0])
	s.Require().Equal(target, decoded[1])

	hasRole, err := timelockContract.HasRole(&bind.CallOpts{Context: ctx}, [32]byte(roleHash), target)
	s.Require().NoError(err, "Failed to inspect role")
	s.Require().False(hasRole, "NoSend should not broadcast GrantRole transaction")
}
