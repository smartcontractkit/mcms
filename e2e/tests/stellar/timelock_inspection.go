package stellar

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	stellarbindings "github.com/smartcontractkit/chainlink-stellar/bindings"
	timelockbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/timelock"
	stellardeployer "github.com/smartcontractkit/chainlink-stellar/deployment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	stellarsdk "github.com/smartcontractkit/mcms/sdk/stellar"
	mcmtypes "github.com/smartcontractkit/mcms/types"
)

// TimelockInspectionTestSuite exercises the read-side
// sdk.TimelockInspector against a freshly deployed Stellar RBAC timelock.
type TimelockInspectionTestSuite struct {
	suite.Suite
	e2e.TestSetup

	deployer      *stellardeployer.Deployer
	chainSelector mcmtypes.ChainSelector
	passphrase    string
	stellarSigner stellarbindings.Signer

	mcmsAddress     string
	timelockAddress string
	inspector       *stellarsdk.TimelockInspector

	deploymentCounter uint64
}

func (s *TimelockInspectionTestSuite) SetupSuite() {
	env := bootstrapStellarChainEnv(s.T())

	s.TestSetup = env.testSetup
	s.stellarSigner = env.signer
	s.deployer = env.deployer
	s.chainSelector = env.chainSelector
	s.passphrase = env.passphrase

	_, signerAddresses := generateSortedSigners(s.T(), 1)
	mcmsConfig := &mcmtypes.Config{
		Quorum:  1,
		Signers: []common.Address{signerAddresses[0]},
	}

	deploymentID := s.nextDeploymentID()

	s.mcmsAddress = deployStellarMCMSContract(
		s.T(),
		s.deployer,
		s.stellarSigner.Address(),
		s.chainSelector,
		mcmsConfig,
		fmt.Sprintf("e2e_tlinsp_%d", deploymentID),
	)

	roles := defaultTimelockRoleConfig(s.mcmsAddress, s.stellarSigner.Address())

	s.timelockAddress = deployStellarTimelockContract(
		s.T(),
		s.deployer,
		"tlinsp",
		deploymentID,
		roles,
	)

	inspector, err := stellarsdk.NewTimelockInspectorWithNetworkPassphrase(
		s.StellarClient,
		s.stellarSigner,
		s.passphrase,
	)
	s.Require().NoError(err, "failed to create Stellar timelock inspector")
	s.inspector = inspector
}

// TestGetProposers verifies the proposer set granted at initialization.
func (s *TimelockInspectionTestSuite) TestGetProposers() {
	ctx := s.T().Context()

	proposers, err := s.inspector.GetProposers(ctx, s.timelockAddress)
	s.Require().NoError(err, "getting proposers")
	s.Require().ElementsMatch(
		[]string{s.mcmsAddress, s.stellarSigner.Address()},
		proposers,
		"proposers should be the MCMS instance and the operator",
	)
}

// TestGetExecutors verifies GetExecutors is unsupported on Stellar: the
// timelock defines no executor role, execution is permissionless once an
// operation is ready.
func (s *TimelockInspectionTestSuite) TestGetExecutors() {
	ctx := s.T().Context()

	executors, err := s.inspector.GetExecutors(ctx, s.timelockAddress)
	s.Require().Error(err, "GetExecutors should return an error on Stellar")
	s.Require().Contains(err.Error(), "unsupported on Stellar")
	s.Require().Nil(executors)
}

// TestGetBypassers verifies the bypasser set granted at initialization.
func (s *TimelockInspectionTestSuite) TestGetBypassers() {
	ctx := s.T().Context()

	bypassers, err := s.inspector.GetBypassers(ctx, s.timelockAddress)
	s.Require().NoError(err, "getting bypassers")
	s.Require().ElementsMatch(
		[]string{s.mcmsAddress, s.stellarSigner.Address()},
		bypassers,
		"bypassers should be the MCMS instance and the operator",
	)
}

// TestGetCancellers verifies the canceller set granted at initialization.
func (s *TimelockInspectionTestSuite) TestGetCancellers() {
	ctx := s.T().Context()

	cancellers, err := s.inspector.GetCancellers(ctx, s.timelockAddress)
	s.Require().NoError(err, "getting cancellers")
	s.Require().ElementsMatch(
		[]string{s.mcmsAddress, s.stellarSigner.Address()},
		cancellers,
		"cancellers should be the MCMS instance and the operator",
	)
}

// TestGetMinDelay verifies the timelock was initialized with zero delay.
func (s *TimelockInspectionTestSuite) TestGetMinDelay() {
	ctx := s.T().Context()

	delay, err := s.inspector.GetMinDelay(ctx, s.timelockAddress)
	s.Require().NoError(err, "getting min delay")
	s.Require().Zero(delay, "timelock should be initialized with zero delay")
}

// TestIsInitialized verifies the initialization probe after deployment.
func (s *TimelockInspectionTestSuite) TestIsInitialized() {
	ctx := s.T().Context()

	initialized, err := s.inspector.IsInitialized(ctx, s.timelockAddress)
	s.Require().NoError(err, "checking initialization")
	s.Require().True(initialized, "timelock should be initialized")
}

// TestOperationLifecycle walks an operation through the full timelock
// lifecycle: unknown -> scheduled (ready immediately, since the min delay is
// zero) -> executed. The boolean probes are derived on-chain from the stored
// ready-at timestamp: 0 = nonexistent, 1 = done, otherwise the ready-at time.
func (s *TimelockInspectionTestSuite) TestOperationLifecycle() {
	ctx := s.T().Context()

	var unknownID [32]byte
	copy(unknownID[:], "stellar-inspection-unknown-op")

	isOp, err := s.inspector.IsOperation(ctx, s.timelockAddress, unknownID)
	s.Require().NoError(err)
	s.Require().False(isOp, "unknown op ID should not be an operation")

	isPending, err := s.inspector.IsOperationPending(ctx, s.timelockAddress, unknownID)
	s.Require().NoError(err)
	s.Require().False(isPending, "unknown op ID should not be pending")

	isReady, err := s.inspector.IsOperationReady(ctx, s.timelockAddress, unknownID)
	s.Require().NoError(err)
	s.Require().False(isReady, "unknown op ID should not be ready")

	isDone, err := s.inspector.IsOperationDone(ctx, s.timelockAddress, unknownID)
	s.Require().NoError(err)
	s.Require().False(isDone, "unknown op ID should not be done")

	calls := s.readOnlyCalls()
	var predecessor [32]byte
	salt := s.operationSalt("stellar-inspection-lifecycle")

	timelockClient := timelockbindings.NewTimelockClient(s.deployer, s.timelockAddress)

	opID, err := timelockClient.HashOperationBatch(ctx, calls, predecessor, salt)
	s.Require().NoError(err, "hashing operation batch")

	err = timelockClient.ScheduleBatch(ctx, s.stellarSigner.Address(), calls, predecessor, salt, 0)
	s.Require().NoError(err, "scheduling operation batch")

	// With zero delay the operation is ready as soon as the ledger closes.
	s.Require().EventuallyWithT(func(c *assert.CollectT) {
		isOp, opErr := s.inspector.IsOperation(ctx, s.timelockAddress, opID)
		if !assert.NoError(c, opErr) || !assert.True(c, isOp, "scheduled op should be an operation") {
			return
		}

		isPending, pendingErr := s.inspector.IsOperationPending(ctx, s.timelockAddress, opID)
		if !assert.NoError(c, pendingErr) || !assert.True(c, isPending, "scheduled op should be pending") {
			return
		}

		isReady, readyErr := s.inspector.IsOperationReady(ctx, s.timelockAddress, opID)
		if !assert.NoError(c, readyErr) || !assert.True(c, isReady, "zero-delay op should be ready") {
			return
		}
	}, 30*time.Second, 2*time.Second, "operation should become ready")

	err = timelockClient.ExecuteBatch(ctx, calls, predecessor, salt)
	s.Require().NoError(err, "executing operation batch")

	s.Require().EventuallyWithT(func(c *assert.CollectT) {
		isDone, err := s.inspector.IsOperationDone(ctx, s.timelockAddress, opID)
		if !assert.NoError(c, err) || !assert.True(c, isDone, "executed op should be done") {
			return
		}

		isPending, err := s.inspector.IsOperationPending(ctx, s.timelockAddress, opID)
		if !assert.NoError(c, err) || !assert.False(c, isPending, "done op should not be pending") {
			return
		}
	}, 30*time.Second, 2*time.Second, "operation should be recorded as done")
}

// TestCancelledOperationRemovedFromState verifies the documented cancellation
// semantics: cancelling deletes the stored timestamp, so a cancelled operation
// is indistinguishable from one that was never scheduled.
func (s *TimelockInspectionTestSuite) TestCancelledOperationRemovedFromState() {
	ctx := s.T().Context()

	calls := s.readOnlyCalls()
	var predecessor [32]byte
	salt := s.operationSalt("stellar-inspection-cancelled")

	timelockClient := timelockbindings.NewTimelockClient(s.deployer, s.timelockAddress)

	opID, err := timelockClient.HashOperationBatch(ctx, calls, predecessor, salt)
	s.Require().NoError(err, "hashing operation batch")

	err = timelockClient.ScheduleBatch(ctx, s.stellarSigner.Address(), calls, predecessor, salt, 0)
	s.Require().NoError(err, "scheduling operation batch")

	s.Require().EventuallyWithT(func(c *assert.CollectT) {
		scheduled, schedErr := s.inspector.IsOperation(ctx, s.timelockAddress, opID)
		if !assert.NoError(c, schedErr) || !assert.True(c, scheduled, "scheduled op should be an operation") {
			return
		}
	}, 30*time.Second, 2*time.Second, "operation should be scheduled")

	err = timelockClient.Cancel(ctx, s.stellarSigner.Address(), opID)
	s.Require().NoError(err, "cancelling operation batch")

	s.Require().EventuallyWithT(func(c *assert.CollectT) {
		cancelled, cancelErr := s.inspector.IsOperation(ctx, s.timelockAddress, opID)
		if !assert.NoError(c, cancelErr) || !assert.False(c, cancelled, "cancelled op should not be an operation") {
			return
		}
	}, 30*time.Second, 2*time.Second, "cancelled operation should be removed from state")
}

// readOnlyCalls builds a one-call batch invoking a read function on the MCMS
// instance: sufficient to schedule and execute, with no state changes.
func (s *TimelockInspectionTestSuite) readOnlyCalls() timelockbindings.Calls {
	s.T().Helper()

	encoded, err := stellarsdk.EncodeSorobanInvokePayload("get_op_count", nil)
	s.Require().NoError(err)

	payload, err := stellarsdk.DecodeSorobanInvokePayload(encoded)
	s.Require().NoError(err)

	return timelockbindings.Calls{
		Inner: []timelockbindings.Call{
			{
				Target:   s.mcmsAddress,
				Function: "get_op_count",
				ArgsXdr:  payload.ArgsXDR,
			},
		},
	}
}

func (s *TimelockInspectionTestSuite) operationSalt(purpose string) [32]byte {
	s.T().Helper()

	salt := [32]byte(crypto.Keccak256Hash([]byte(purpose)))

	return salt
}

func (s *TimelockInspectionTestSuite) nextDeploymentID() uint64 {
	s.deploymentCounter++

	return s.deploymentCounter
}
