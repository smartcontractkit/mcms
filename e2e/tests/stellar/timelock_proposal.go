package stellar

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	stellarcre "github.com/smartcontractkit/chainlink-stellar/deployment/cre"
	crereceiver "github.com/smartcontractkit/chainlink-stellar/deployment/cre/receiver"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	stellarsdk "github.com/smartcontractkit/mcms/sdk/stellar"
	mcmtypes "github.com/smartcontractkit/mcms/types"
)

// TestTimelockProposalSchedule is the happy-path timelock flow every other
// chain family covers in e2e: build a schedule proposal, sign, SetRoot,
// execute through MCMS (schedule_batch), wait for the delay, then execute
// through the timelock (execute_batch). The batch carries TWO transactions —
// the shape governed forwarder allow-list changes produce — targeting the CRE
// test receiver, whose counters prove both sub-calls ran and in order.
func (s *ExecutionTestSuite) TestTimelockProposalSchedule() {
	ctx := s.T().Context()

	chain := s.deployChain()

	receiverAddress := s.deployReceiver()

	inspector := s.newInspector()

	opCount, err := inspector.GetOpCount(ctx, chain.MCMAddress)
	s.Require().NoError(err)

	// Two on_report calls with different payloads. last_value_u64 returns the
	// first 8 payload bytes little-endian, so the final value proves the
	// second call ran last.
	const firstValue, secondValue = uint64(111), uint64(222)

	batchOps := []mcmtypes.BatchOperation{
		{
			ChainSelector: s.chainSelector,
			Transactions: []mcmtypes.Transaction{
				s.createOnReportTransaction(receiverAddress, firstValue),
				s.createOnReportTransaction(receiverAddress, secondValue),
			},
		},
	}

	chainMetadata := map[mcmtypes.ChainSelector]mcmtypes.ChainMetadata{
		s.chainSelector: {
			StartingOpCount: opCount,
			MCMAddress:      chain.MCMAddress,
		},
	}

	timelockAddresses := map[mcmtypes.ChainSelector]string{
		s.chainSelector: chain.TimelockAddress,
	}

	delay := mcmtypes.MustParseDuration("2s")

	timelockProposal := mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			Version:     "v1",
			Kind:        mcmtypes.KindTimelockProposal,
			Description: "Stellar timelock schedule happy path",
			ValidUntil: uint32(
				time.Now().Add(time.Hour).Unix(),
			),
			OverridePreviousRoot: true,
			Signatures:           []mcmtypes.Signature{},
			ChainMetadata:        chainMetadata,
		},
		Action:            mcmtypes.TimelockActionSchedule,
		Delay:             delay,
		TimelockAddresses: timelockAddresses,
		Operations:        batchOps,
	}

	proposal, _, err := timelockProposal.Convert(
		ctx,
		map[mcmtypes.ChainSelector]sdk.TimelockConverter{
			s.chainSelector: stellarsdk.NewTimelockConverter(),
		},
	)
	s.Require().NoError(err)

	tree, err := proposal.MerkleTree()
	s.Require().NoError(err)

	signable, err := mcms.NewSignable(
		&proposal,
		map[mcmtypes.ChainSelector]sdk.Inspector{
			s.chainSelector: inspector,
		},
	)
	s.Require().NoError(err)
	s.Require().NoError(signable.ValidateConfigs(ctx))

	for _, key := range s.proposalSignerKeys {
		_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(key))
		s.Require().NoError(err)
	}

	quorumMet, err := signable.ValidateSignatures(ctx)
	s.Require().NoError(err)
	s.Require().True(quorumMet, "MCMS signer quorum was not met")

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)

	encoder, ok := encoders[s.chainSelector].(*stellarsdk.Encoder)
	s.Require().True(ok, "proposal encoder is not a Stellar encoder")

	executor, err := stellarsdk.NewExecutor(encoder, inspector)
	s.Require().NoError(err)

	executable, err := mcms.NewExecutable(
		&proposal,
		map[mcmtypes.ChainSelector]sdk.Executor{
			s.chainSelector: executor,
		},
	)
	s.Require().NoError(err)

	_, err = executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)

	actualRoot, _, err := inspector.GetRoot(ctx, chain.MCMAddress)
	s.Require().NoError(err)
	s.Require().Equal(tree.Root, actualRoot, "MCMS root mismatch after SetRoot")

	// MCMS calls schedule_batch on the timelock; the receiver must not have
	// been touched yet.
	_, err = executable.Execute(ctx, 0)
	s.Require().NoError(err, "MCMS should schedule the timelock operation")

	count, err := crereceiver.ReportCount(ctx, s.deployer, receiverAddress)
	s.Require().NoError(err)
	s.Require().Zero(count, "receiver must not receive reports before timelock execution")

	newOpCount, err := inspector.GetOpCount(ctx, chain.MCMAddress)
	s.Require().NoError(err)
	s.Require().Equal(opCount+1, newOpCount, "MCMS op count should advance on schedule")

	timelockExecutor, err := stellarsdk.NewTimelockExecutorWithNetworkPassphrase(
		s.StellarClient,
		s.StellarSigner,
		s.passphrase,
		s.StellarSigner.Address(),
	)
	s.Require().NoError(err)

	timelockExecutable, err := mcms.NewTimelockExecutable(
		ctx,
		&timelockProposal,
		map[mcmtypes.ChainSelector]sdk.TimelockExecutor{
			s.chainSelector: timelockExecutor,
		},
	)
	s.Require().NoError(err)

	// Readiness is ledger-timestamp based, not wall-clock: poll IsReady
	// instead of sleeping (same pattern as the Aptos suite).
	s.Require().EventuallyWithT(func(c *assert.CollectT) {
		assert.NoError(c, timelockExecutable.IsReady(ctx))
	}, 30*time.Second, 500*time.Millisecond, "timelock operation never became ready")

	_, err = timelockExecutable.Execute(ctx, 0)
	s.Require().NoError(err, "timelock execute_batch should succeed")

	// Both sub-calls ran, in order: two reports recorded, and the last value
	// is the second payload.
	count, err = crereceiver.ReportCount(ctx, s.deployer, receiverAddress)
	s.Require().NoError(err)
	s.Require().Equal(uint32(2), count, "both batch transactions must execute")

	lastValue := s.receiverLastValue(receiverAddress)
	s.Require().Equal(secondValue, lastValue, "batch transactions must execute in order")
}

// deployReceiver deploys the CRE test receiver, an unauthenticated on_report
// counter, as a benign multi-call target for timelock batches.
func (s *ExecutionTestSuite) deployReceiver() string {
	s.T().Helper()

	wasm, err := stellarcre.Artifact(stellarcre.ReceiverWasm)
	s.Require().NoError(err)

	salt := sha256.Sum256(
		[]byte(fmt.Sprintf("e2e_receiver_%d", s.nextDeploymentID())),
	)

	address, err := crereceiver.DeployReceiver(
		s.T().Context(),
		s.deployer,
		wasm,
		salt,
	)
	s.Require().NoError(err)

	return address
}

// createOnReportTransaction builds an on_report call whose payload starts with
// the given value encoded little-endian, matching the receiver's
// last_value_u64 read-back.
func (s *ExecutionTestSuite) createOnReportTransaction(
	receiverAddress string,
	value uint64,
) mcmtypes.Transaction {
	s.T().Helper()

	payload := make([]byte, 8)
	binary.LittleEndian.PutUint64(payload, value)

	args := []xdr.ScVal{
		scval.AddressToScVal(s.StellarSigner.Address()),
		scval.BytesToScVal([]byte{0x01}),
		scval.BytesToScVal(payload),
	}

	transaction, err := stellarsdk.NewTransaction(
		receiverAddress,
		"on_report",
		args,
		"CRETestReceiver",
		nil,
	)
	s.Require().NoError(err)

	return transaction
}

func (s *ExecutionTestSuite) receiverLastValue(receiverAddress string) uint64 {
	s.T().Helper()

	result, err := s.deployer.SimulateContract(
		s.T().Context(),
		receiverAddress,
		"last_value_u64",
		nil,
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	value, ok := result.GetU64()
	s.Require().True(ok, "last_value_u64 did not return u64")

	return uint64(value)
}
