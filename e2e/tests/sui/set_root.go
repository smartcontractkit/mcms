//go:build e2e

package sui

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

// SetRootTestSuite defines the test suite for Sui SetRoot tests
type SetRootTestSuite struct {
	SuiTestSuite
}

// SetupSuite runs before the test suite
func (s *SetRootTestSuite) SetupSuite() {
	s.SuiTestSuite.SetupSuite()
	s.SuiTestSuite.DeployMCMSContract()
}

// TestSetRoot tests the SetRoot functionality by setting a root on the MCMS contract
func (s *SetRootTestSuite) TestSetRoot() {
	ctx := context.Background()

	// Create a test signer
	signerKey, err := crypto.GenerateKey()
	s.Require().NoError(err, "Failed to generate test signer key")
	signerAddress := crypto.PubkeyToAddress(signerKey.PublicKey)

	// Create configuration with the test signer
	mcmConfig := types.Config{
		Quorum:  1,
		Signers: []common.Address{signerAddress},
	}

	// Create metadata for this chain
	metadata := types.ChainMetadata{
		StartingOpCount: 0,
		MCMAddress:      s.mcmsObj,
		AdditionalFields: []byte(`{
			"role": 2
		}`),
	}

	// Build a test proposal
	validUntil := time.Now().Add(10 * time.Hour)
	proposal, err := mcms.NewProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil.Unix())).
		SetDescription(fmt.Sprintf("SetRoot test proposal - %v", validUntil.UnixMilli())).
		SetOverridePreviousRoot(true).
		AddChainMetadata(s.chainSelector, metadata).
		AddOperation(types.Operation{
			ChainSelector: s.chainSelector,
			Transaction: types.Transaction{
				To:   s.accountObj, // Use account object as target
				Data: []byte("test-transaction-data"),
				AdditionalFields: json.RawMessage(`{
					"module_name": "mcms_account",
					"function": "accept_ownership_as_timelock"
				}`),
			},
		}).
		Build()
	s.Require().NoError(err, "Failed to build proposal")

	// Create inspectors and encoders
	inspector, err := suisdk.NewInspector(s.client, s.signer, s.mcmsPackageId, suisdk.TimelockRoleProposer)
	s.Require().NoError(err, "Failed to create inspector")

	inspectors := map[types.ChainSelector]sdk.Inspector{
		s.chainSelector: inspector,
	}

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err, "Failed to get encoders")
	encoder := encoders[s.chainSelector].(*suisdk.Encoder)

	executor, err := suisdk.NewExecutor(s.client, s.signer, encoder, s.mcmsPackageId, suisdk.TimelockRoleProposer, s.accountObj, s.registryObj, s.timelockObj)
	s.Require().NoError(err, "Failed to create executor")

	executors := map[types.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}

	// Sign the proposal
	signable, err := mcms.NewSignable(proposal, inspectors)
	s.Require().NoError(err, "Failed to create signable")
	s.Require().NotNil(signable, "Signable should not be nil")

	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(signerKey))
	s.Require().NoError(err, "Failed to sign proposal")

	s.Run("SetConfig and SetRoot", func() {
		// First, set the configuration
		configurer, err := suisdk.NewConfigurer(s.client, s.signer, suisdk.TimelockRoleProposer, s.mcmsPackageId, s.ownerCapObj, uint64(s.chainSelector))
		s.Require().NoError(err, "Failed to create configurer")

		configTx, err := configurer.SetConfig(ctx, s.mcmsObj, &mcmConfig, true)
		s.Require().NoError(err, "Failed to set config")
		s.T().Logf("✅ SetConfig in tx: %s", configTx.Hash)

		// Create executable for SetRoot
		executable, err := mcms.NewExecutable(proposal, executors)
		s.Require().NoError(err, "Failed to create executable")

		// Call SetRoot
		setRootResult, err := executable.SetRoot(ctx, s.chainSelector)
		s.Require().NoError(err, "Failed to set root")
		s.T().Logf("✅ SetRoot in tx: %s", setRootResult.Hash)

		// Verify the root was set by checking with inspector
		root, validUntilActual, err := inspector.GetRoot(ctx, s.mcmsObj)
		s.Require().NoError(err, "Failed to get root after setting")

		// Root should not be empty after setting
		s.Require().NotEqual(common.Hash{}, root, "Root should not be empty after SetRoot")
		s.Require().Greater(validUntilActual, uint32(0), "ValidUntil should be greater than 0 after SetRoot")

		// Verify root metadata
		rootMetadata, err := inspector.GetRootMetadata(ctx, s.mcmsObj)
		s.Require().NoError(err, "Failed to get root metadata")
		s.Require().Equal(uint64(0), rootMetadata.StartingOpCount, "StartingOpCount should match metadata")
		s.Require().NotEmpty(rootMetadata.MCMAddress, "MCMAddress should not be empty")
	})
}

// TestSetRootMultipleSigners tests SetRoot with multiple signers and higher quorum
func (s *SetRootTestSuite) TestSetRootMultipleSigners() {
	ctx := context.Background()

	// Create multiple test signers
	signerKey1, err := crypto.GenerateKey()
	s.Require().NoError(err, "Failed to generate first signer key")
	signerAddress1 := crypto.PubkeyToAddress(signerKey1.PublicKey)

	signerKey2, err := crypto.GenerateKey()
	s.Require().NoError(err, "Failed to generate second signer key")
	signerAddress2 := crypto.PubkeyToAddress(signerKey2.PublicKey)

	// Create configuration with multiple signers and quorum of 2
	mcmConfig := types.Config{
		Quorum:  2,
		Signers: []common.Address{signerAddress1, signerAddress2},
	}

	// Create metadata for this chain
	metadata := types.ChainMetadata{
		StartingOpCount: 0,
		MCMAddress:      s.mcmsObj,
		AdditionalFields: []byte(`{
			"role": 2
		}`),
	}

	// Build a test proposal
	validUntil := time.Now().Add(10 * time.Hour)
	proposal, err := mcms.NewProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil.Unix())).
		SetDescription(fmt.Sprintf("Multi-signer SetRoot test - %v", validUntil.UnixMilli())).
		SetOverridePreviousRoot(true).
		AddChainMetadata(s.chainSelector, metadata).
		AddOperation(types.Operation{
			ChainSelector: s.chainSelector,
			Transaction: types.Transaction{
				To:   s.accountObj,
				Data: []byte("multi-signer-test-data"),
				AdditionalFields: json.RawMessage(`{
					"module_name": "mcms_account", 
					"function": "accept_ownership_as_timelock"
				}`),
			},
		}).
		Build()
	s.Require().NoError(err, "Failed to build multi-signer proposal")

	// Create inspectors
	multiInspector, err := suisdk.NewInspector(s.client, s.signer, s.mcmsPackageId, suisdk.TimelockRoleProposer)
	s.Require().NoError(err, "Failed to create inspector for multi-signer test")

	multiInspectors := map[types.ChainSelector]sdk.Inspector{
		s.chainSelector: multiInspector,
	}

	// Create signable and sign with both signers
	signable, err := mcms.NewSignable(proposal, multiInspectors)
	s.Require().NoError(err, "Failed to create signable for multi-signer test")

	// Sign with first signer
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(signerKey1))
	s.Require().NoError(err, "Failed to sign with first signer")

	// Sign with second signer
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(signerKey2))
	s.Require().NoError(err, "Failed to sign with second signer")

	// Note: We can't directly access Signatures field, but signing should work
	s.T().Log("Successfully signed proposal with both signers")

	s.Run("SetConfig and SetRoot with multiple signers", func() {
		// Set configuration with multiple signers
		configurer, err := suisdk.NewConfigurer(s.client, s.signer, suisdk.TimelockRoleProposer, s.mcmsPackageId, s.ownerCapObj, uint64(s.chainSelector))
		s.Require().NoError(err, "Failed to create configurer for multi-signer test")

		configTx, err := configurer.SetConfig(ctx, s.mcmsObj, &mcmConfig, true)
		s.Require().NoError(err, "Failed to set multi-signer config")
		s.T().Logf("✅ Multi-signer SetConfig in tx: %s", configTx.Hash)

		// Create executable
		encoders, err := proposal.GetEncoders()
		s.Require().NoError(err, "Failed to get encoders for multi-signer test")
		encoder := encoders[s.chainSelector].(*suisdk.Encoder)

		multiExecutor, err := suisdk.NewExecutor(s.client, s.signer, encoder, s.mcmsPackageId, suisdk.TimelockRoleProposer, s.accountObj, s.registryObj, s.timelockObj)
		s.Require().NoError(err, "Failed to create executor for multi-signer test")

		multiExecutors := map[types.ChainSelector]sdk.Executor{
			s.chainSelector: multiExecutor,
		}

		executable, err := mcms.NewExecutable(proposal, multiExecutors)
		s.Require().NoError(err, "Failed to create executable for multi-signer test")

		// Call SetRoot
		setRootResult, err := executable.SetRoot(ctx, s.chainSelector)
		s.Require().NoError(err, "Failed to set root with multiple signers")
		s.T().Logf("✅ Multi-signer SetRoot in tx: %s", setRootResult.Hash)

		// Verify the root was set
		root, validUntilActual, err := multiInspector.GetRoot(ctx, s.mcmsObj)
		s.Require().NoError(err, "Failed to get root after multi-signer SetRoot")
		s.Require().NotEqual(common.Hash{}, root, "Root should not be empty after multi-signer SetRoot")
		s.Require().Greater(validUntilActual, uint32(0), "ValidUntil should be greater than 0 after multi-signer SetRoot")
	})
}
