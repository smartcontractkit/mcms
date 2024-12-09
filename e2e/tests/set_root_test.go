//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/suite"

	"github.com/smartcontractkit/mcms"
	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	mcmtypes "github.com/smartcontractkit/mcms/types"
)

// SetRootTestSuite tests the SetRoot functionality
type SetRootTestSuite struct {
	suite.Suite
	mcmsContract     *bindings.ManyChainMultiSig
	deployerKey      common.Address
	signerAddresses  []common.Address
	auth             *bind.TransactOpts
	timelockContract *bindings.RBACTimelock
	chainSelector    mcmtypes.ChainSelector
	TestSetup
}

// SetupSuite runs before the test suite
func (s *SetRootTestSuite) SetupSuite() {
	s.TestSetup = *InitializeSharedTestSetup(s.T())
	// Get deployer's private key
	privateKeyHex := s.Settings.PrivateKeys[0]
	privateKey, err := crypto.HexToECDSA(privateKeyHex[2:]) // Strip "0x" prefix
	s.Require().NoError(err, "Invalid private key")

	// Define signer addresses
	s.signerAddresses = []common.Address{
		common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"),
		common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"),
	}

	// Parse ChainID from string to int64
	chainID, ok := new(big.Int).SetString(s.BlockchainA.Out.ChainID, 10)
	s.Require().True(ok, "Failed to parse chain ID")

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	s.Require().NoError(err, "Failed to create transactor")
	s.auth = auth

	s.mcmsContract = s.deployMCMSContract()
	s.timelockContract = s.deployTimelockContract(s.mcmsContract.Address().String())
	s.deployerKey = crypto.PubkeyToAddress(privateKey.PublicKey)
	chainDetails, err := cselectors.GetChainDetailsByChainIDAndFamily(s.BlockchainA.Out.ChainID, s.Config.Settings.ChainFamily)
	s.Require().NoError(err)
	s.chainSelector = mcmtypes.ChainSelector(chainDetails.ChainSelector)
}

// deployMCMSContract is a helper to deploy the MCMS contract with the required configuration for the test.
func (s *SetRootTestSuite) deployMCMSContract() *bindings.ManyChainMultiSig {
	_, tx, instance, err := bindings.DeployManyChainMultiSig(s.auth, s.Client)
	s.Require().NoError(err, "Failed to deploy contract")

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), s.Client, tx)
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Set configurations
	signerGroups := []uint8{0, 1}   // Two groups: Group 0 and Group 1
	groupQuorums := [32]uint8{1, 1} // Quorum 1 for both groups
	groupParents := [32]uint8{0, 0} // Group 0 is its own parent; Group 1's parent is Group 0
	clearRoot := true

	tx, err = instance.SetConfig(s.auth, s.signerAddresses, signerGroups, groupQuorums, groupParents, clearRoot)
	s.Require().NoError(err, "Failed to set contract configuration")
	receipt, err = bind.WaitMined(context.Background(), s.Client, tx)
	s.Require().NoError(err, "Failed to mine configuration transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	return instance
}

// deployContract is a helper to deploy the contract
func (s *SetRootTestSuite) deployTimelockContract(mcmsAddress string) *bindings.RBACTimelock {
	_, tx, instance, err := bindings.DeployRBACTimelock(
		s.auth,
		s.Client,
		big.NewInt(0),
		common.HexToAddress(mcmsAddress),
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
	)
	s.Require().NoError(err, "Failed to deploy contract")

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), s.Client, tx)
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	return instance
}

// TestSetRootProposal sets the root of the MCMS contract
func (s *SetRootTestSuite) TestSetRootProposal() {
	builder := mcms.NewProposalBuilder()
	builder.
		SetVersion("v1").
		SetValidUntil(1794610529).
		SetDescription("proposal to test SetRoot").
		SetOverridePreviousRoot(true).
		AddChainMetadata(
			s.chainSelector,
			mcmtypes.ChainMetadata{MCMAddress: s.mcmsContract.Address().String()},
		).
		AddOperation(mcmtypes.Operation{
			ChainSelector: s.chainSelector,
			Transaction: mcmtypes.Transaction{
				To:               s.signerAddresses[0].Hex(),
				Data:             []byte("0x"),
				AdditionalFields: json.RawMessage(`{"value": 3}`),
			},
		})
	proposal, err := builder.Build()
	s.Require().NoError(err)

	// Sign the proposal
	inspectors := map[mcmtypes.ChainSelector]sdk.Inspector{
		s.chainSelector: evm.NewInspector(s.Client),
	}
	signable, err := mcms.NewSignable(proposal, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)

	err = signable.ValidateConfigs()
	s.Require().NoError(err)

	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(testutils.ParsePrivateKey(s.Settings.PrivateKeys[1])))
	s.Require().NoError(err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	s.Require().NoError(err)
	s.Require().True(quorumMet)

	// Create the chain MCMS proposal executor
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[mcmtypes.ChainSelector(s.chainSelector)].(*evm.Encoder)

	executor := evm.NewExecutor(encoder, s.Client, s.auth)
	executorsMap := map[mcmtypes.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}
	executable, err := mcms.NewExecutable(proposal, executorsMap)
	s.Require().NoError(err)

	// Call SetRoot
	txHash, err := executable.SetRoot(s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(txHash)

	receipt, err := testutils.WaitMinedWithTxHash(context.Background(), s.Client, common.HexToHash(txHash))
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)
}

// TestSetRootTimelockProposal sets the root of the MCMS contract from a timelock proposal type.
func (s *SetRootTestSuite) TestSetRootTimelockProposal() {
	builder := mcms.NewTimelockProposalBuilder()
	builder.
		SetVersion("v1").
		SetValidUntil(1794610529).
		SetDescription("proposal to test SetRoot").
		SetOverridePreviousRoot(true).
		SetAction(mcmtypes.TimelockActionSchedule).
		SetDelay(mcmtypes.MustParseDuration("24h")).
		SetTimelockAddresses(map[mcmtypes.ChainSelector]string{
			s.chainSelector: s.timelockContract.Address().String(),
		}).
		AddChainMetadata(
			s.chainSelector,
			mcmtypes.ChainMetadata{MCMAddress: s.mcmsContract.Address().String()},
		).
		AddOperation(mcmtypes.BatchOperation{
			ChainSelector: s.chainSelector,
			Transactions: []mcmtypes.Transaction{
				{
					To:               s.signerAddresses[0].Hex(),
					Data:             []byte("0x01"),
					AdditionalFields: json.RawMessage(`{"value": 3}`),
				},
				{
					To:               s.signerAddresses[0].Hex(),
					Data:             []byte("0x02"),
					AdditionalFields: json.RawMessage(`{"value": 4}`),
				},
			},
		})
	proposalTimelock, err := builder.Build()
	s.Require().NoError(err)
	proposal, err := proposalTimelock.Convert()
	s.Require().NoError(err)

	// Sign proposal
	inspectors := map[mcmtypes.ChainSelector]sdk.Inspector{
		s.chainSelector: evm.NewInspector(s.Client),
	}
	signable, err := mcms.NewSignable(&proposal, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)

	err = signable.ValidateConfigs()
	s.Require().NoError(err)

	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(testutils.ParsePrivateKey(s.Settings.PrivateKeys[1])))
	s.Require().NoError(err)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[mcmtypes.ChainSelector(s.chainSelector)].(*evm.Encoder)

	executor := evm.NewExecutor(encoder, s.Client, s.auth)
	executorsMap := map[mcmtypes.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}
	// Create the chain MCMS proposal executor
	executable, err := mcms.NewExecutable(&proposal, executorsMap)
	s.Require().NoError(err)
	// Call SetRoot
	txHash, err := executable.SetRoot(s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(txHash)
	// Check receipt
	receipt, err := testutils.WaitMinedWithTxHash(context.Background(), s.Client, common.HexToHash(txHash))
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)
}
