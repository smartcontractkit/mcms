//go:build e2e

package sui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/block-vision/sui-go-sdk/models"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	module_mcms_deployer "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms_deployer"
	module_mcms_user "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms_user"
	"github.com/smartcontractkit/chainlink-sui/contracts"

	mcmslib "github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

const (
	DefaultGasBudget = uint64(300_000_000)
	UpgradeGasBudget = uint64(500_000_000)
)

type MCMSUserUpgradeTestSuite struct {
	TestSuite
}

func (s *MCMSUserUpgradeTestSuite) Test_Sui_MCMSUser_UpgradeProposal() {
	s.Run("TimelockProposal - MCMS User Upgrade through Schedule", func() {
		RunMCMSUserUpgradeProposal(s)
	})
}

func createMCMSAcceptOwnershipTransaction(suite *MCMSUserUpgradeTestSuite) (types.Transaction, error) {
	encodedCall, err := suite.mcmsAccount.Encoder().AcceptOwnershipAsTimelock(bind.Object{Id: suite.accountObj})
	if err != nil {
		return types.Transaction{}, fmt.Errorf("encoding accept ownership call: %w", err)
	}

	accountStateAddr, err := suisdk.AddressFromHex(suite.accountObj)
	if err != nil {
		return types.Transaction{}, fmt.Errorf("decoding account state address: %w", err)
	}

	// The data for MCMS dispatch should be the properly padded 32-byte account_state address
	callBytes := accountStateAddr.Bytes()

	return suisdk.NewTransaction(
		encodedCall.Module.ModuleName,
		encodedCall.Function,
		encodedCall.Module.PackageID,
		callBytes,
		"MCMS",
		[]string{},
	)
}

func RunMCMSUserUpgradeProposal(s *MCMSUserUpgradeTestSuite) {
	ctx := s.T().Context()

	// Phase 1: Deploy contracts
	s.DeployMCMSContract()
	s.DeployMCMSUserContract()
	logDeploymentInfo(s)

	// Phase 2: Configure proposer role
	proposerCount := 3
	proposerQuorum := 2
	proposerConfig := CreateConfig(proposerCount, uint8(proposerQuorum))

	proposerConfigurer, err := suisdk.NewConfigurer(s.client, s.signer, suisdk.TimelockRoleProposer, s.mcmsPackageID, s.ownerCapObj, uint64(s.chainSelector))
	s.Require().NoError(err, "creating proposer configurer")

	_, err = proposerConfigurer.SetConfig(ctx, s.mcmsObj, proposerConfig.Config, true)
	s.Require().NoError(err, "setting proposer config")

	// Phase 3: Setup ownership and registration
	executeMCMSSelfOwnershipTransfer(s.T(), ctx, s, proposerConfig)

	deployerContract, err := module_mcms_deployer.NewMcmsDeployer(s.mcmsPackageID, s.client)
	s.Require().NoError(err)

	gasBudget := UpgradeGasBudget
	_, err = deployerContract.RegisterUpgradeCap(ctx, &bind.CallOpts{
		Signer:           s.signer,
		GasBudget:        &gasBudget,
		WaitForExecution: true,
	},
		bind.Object{Id: s.depStateObj},
		bind.Object{Id: s.registryObj},
		bind.Object{Id: s.mcmsUserUpgradeCapObj},
	)
	s.Require().NoError(err)

	// Phase 4: Verify initial version
	version, err := s.mcmsUser.DevInspect().TypeAndVersion(ctx, &bind.CallOpts{
		Signer:           s.signer,
		WaitForExecution: true,
	})
	s.Require().NoError(err)
	s.Require().Equal(version, "MCMSUser 1.0.0")

	// Phase 5: Execute upgrade
	signerAddr, err := s.signer.GetAddress()
	s.Require().NoError(err)

	compiledPackage, err := bind.CompilePackage(contracts.MCMSUserV2, map[string]string{
		"mcms":       s.mcmsPackageID,
		"mcms_owner": signerAddr,
		"mcms_test":  "0x0",
	})
	s.Require().NoError(err)

	newAddress := executeUpgradePTB(s.T(), ctx, s, compiledPackage, proposerConfig)

	// Phase 6: Verify upgrade completion
	mcmsUserContract, err := module_mcms_user.NewMcmsUser(newAddress, s.client)
	s.Require().NoError(err)

	mcmsUserVersion, err := mcmsUserContract.DevInspect().TypeAndVersion(ctx, &bind.CallOpts{
		Signer:           s.signer,
		WaitForExecution: true,
	})
	s.Require().NoError(err)
	s.Require().Equal(mcmsUserVersion, "MCMSUser 2.0.0")

	s.T().Log("✅ MCMS User upgrade committed successfully - Complete MCMS → Upgrade workflow completed!")
}

func logDeploymentInfo(s *MCMSUserUpgradeTestSuite) {
	s.T().Logf("MCMS Package deployed: %s", s.mcmsPackageID)
	s.T().Logf("Registry Object ID: %s", s.registryObj)
	s.T().Logf("Deployer State ID: %s", s.depStateObj)
	s.T().Logf("MCMS State ID: %s", s.mcmsObj)
	s.T().Logf("Timelock ID: %s", s.timelockObj)
	s.T().Logf("Account State ID: %s", s.accountObj)
	s.T().Logf("Owner Cap ID: %s", s.ownerCapObj)

	s.T().Log("=== Phase 2: MCMS User Test Package Deployed ===")
	s.T().Logf("MCMS User Package deployed: %s", s.mcmsUserPackageID)
	s.T().Logf("MCMS User Owner Cap ID: %s", s.mcmsUserOwnerCapObj)
	s.T().Logf("MCMS User State Object ID: %s", s.stateObj)
}

func executeMCMSSelfOwnershipTransfer(t *testing.T, ctx context.Context, s *MCMSUserUpgradeTestSuite, proposerConfig *RoleConfig) {
	t.Helper()

	gasBudget := DefaultGasBudget
	tx, err := s.mcmsAccount.TransferOwnershipToSelf(
		ctx,
		&bind.CallOpts{
			Signer:           s.signer,
			GasBudget:        &gasBudget,
			WaitForExecution: true,
		},
		bind.Object{Id: s.ownerCapObj},
		bind.Object{Id: s.accountObj},
	)
	s.Require().NoError(err, "Failed to transfer MCMS ownership to self")
	s.Require().NotEmpty(tx, "Transaction should not be empty")

	executeMCMSSelfOwnershipAcceptanceProposal(t, ctx, s, proposerConfig)

	tx, err = s.mcmsAccount.ExecuteOwnershipTransfer(
		ctx,
		&bind.CallOpts{
			Signer:           s.signer,
			GasBudget:        &gasBudget,
			WaitForExecution: true,
		},
		bind.Object{Id: s.ownerCapObj},
		bind.Object{Id: s.accountObj},
		bind.Object{Id: s.registryObj},
		s.mcmsPackageID,
	)
	s.Require().NoError(err, "Failed to execute MCMS ownership transfer")
	s.Require().NotEmpty(tx, "Transaction should not be empty")
}

func executeMCMSSelfOwnershipAcceptanceProposal(t *testing.T, ctx context.Context, s *MCMSUserUpgradeTestSuite, proposerConfig *RoleConfig) {
	t.Helper()

	transaction, err := createMCMSAcceptOwnershipTransaction(s)
	s.Require().NoError(err)

	op := types.BatchOperation{
		ChainSelector: s.chainSelector,
		Transactions:  []types.Transaction{transaction},
	}

	role := suisdk.TimelockRoleProposer
	inspector, err := suisdk.NewInspector(s.client, s.signer, s.mcmsPackageID, role)
	s.Require().NoError(err)

	currentOpCount, err := inspector.GetOpCount(ctx, s.mcmsObj)
	s.Require().NoError(err)

	proposalConfig := ProposalBuilderConfig{
		Version:            "v1",
		Description:        "Accept MCMS self-ownership transfer via proposer with zero delay",
		ChainSelector:      s.chainSelector,
		McmsObjID:          s.mcmsObj,
		TimelockObjID:      s.timelockObj,
		McmsPackageID:      s.mcmsPackageID,
		AccountObjID:       s.accountObj,
		RegistryObjID:      s.registryObj,
		DeployerStateObjID: s.depStateObj,
		Role:               role,
		CurrentOpCount:     currentOpCount,
		Action:             types.TimelockActionSchedule,
		Delay:              &types.Duration{}, // Zero delay for immediate execution
	}

	proposalBuilder := CreateTimelockProposalBuilder(s.T(), proposalConfig, []types.BatchOperation{op})
	timelockProposal, err := proposalBuilder.Build()
	s.Require().NoError(err)

	timelockConverter, err := suisdk.NewTimelockConverter()
	s.Require().NoError(err)

	convertersMap := map[types.ChainSelector]sdk.TimelockConverter{
		s.chainSelector: timelockConverter,
	}
	proposal, _, err := timelockProposal.Convert(ctx, convertersMap)
	s.Require().NoError(err)

	inspectorsMap := map[types.ChainSelector]sdk.Inspector{
		s.chainSelector: inspector,
	}

	quorum := int(proposerConfig.Quorum)
	signable, err := SignProposal(&proposal, inspectorsMap, proposerConfig.Keys, quorum)
	s.Require().NoError(err)

	quorumMet, err := signable.ValidateSignatures(ctx)
	s.Require().NoError(err)
	s.Require().True(quorumMet)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	suiEncoder := encoders[s.chainSelector].(*suisdk.Encoder)

	executor, err := suisdk.NewExecutor(s.client, s.signer, suiEncoder, s.entrypointArgEncoder, s.mcmsPackageID, role, s.mcmsObj, s.accountObj, s.registryObj, s.timelockObj)
	s.Require().NoError(err)

	executors := map[types.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}
	executable, err := mcmslib.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	_, err = executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)

	// Schedule Operations in Timelock
	for i := range proposal.Operations {
		_, execErr := executable.Execute(ctx, i)
		s.Require().NoError(execErr)
	}

	timelockExecutor, err := suisdk.NewTimelockExecutor(s.client, s.signer, s.entrypointArgEncoder, s.mcmsPackageID, s.registryObj, s.accountObj)
	s.Require().NoError(err)

	timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
		s.chainSelector: timelockExecutor,
	}
	timelockExecutable, execErr := mcmslib.NewTimelockExecutable(ctx, timelockProposal, timelockExecutors)
	s.Require().NoError(execErr)

	// Execute the scheduled batch through the timelock
	_, terr := timelockExecutable.Execute(ctx, 0, mcmslib.WithCallProxy(s.timelockObj))
	s.Require().NoError(terr)
}

// executeUpgradePTB creates a single atomic PTB that includes:
// 1. MCMS timelock execution → produces UpgradeTicket
// 2. Package upgrade using the UpgradeTicket → produces UpgradeReceipt
// 3. Commit upgrade using the UpgradeReceipt
func executeUpgradePTB(t *testing.T, ctx context.Context, s *MCMSUserUpgradeTestSuite, compiledPackage bind.PackageArtifact, proposerConfig *RoleConfig) string {
	t.Helper()

	tx, err := suisdk.CreateUpgradeTransaction(compiledPackage, s.mcmsPackageID, s.depStateObj, s.registryObj, s.ownerCapObj, s.mcmsUserPackageID)
	s.Require().NoError(err)

	op := types.BatchOperation{
		ChainSelector: s.chainSelector,
		Transactions:  []types.Transaction{tx},
	}

	role := suisdk.TimelockRoleProposer
	inspector, err := suisdk.NewInspector(s.client, s.signer, s.mcmsPackageID, role)
	s.Require().NoError(err)

	currentOpCount, err := inspector.GetOpCount(ctx, s.mcmsObj)
	s.Require().NoError(err)

	proposalConfig := ProposalBuilderConfig{
		Version:            "v1",
		Description:        "Authorize MCMS User package upgrade via MCMS proposer with zero delay",
		ChainSelector:      s.chainSelector,
		McmsObjID:          s.mcmsObj,
		TimelockObjID:      s.timelockObj,
		McmsPackageID:      s.mcmsPackageID,
		AccountObjID:       s.accountObj,
		RegistryObjID:      s.registryObj,
		DeployerStateObjID: s.depStateObj,
		Role:               role,
		CurrentOpCount:     currentOpCount,
		Action:             types.TimelockActionSchedule,
		Delay:              &types.Duration{}, // Zero delay for immediate execution
	}

	proposalBuilder := CreateTimelockProposalBuilder(s.T(), proposalConfig, []types.BatchOperation{op})
	timelockProposal, err := proposalBuilder.Build()
	s.Require().NoError(err)

	timelockConverter, err := suisdk.NewTimelockConverter()
	s.Require().NoError(err)

	convertersMap := map[types.ChainSelector]sdk.TimelockConverter{
		s.chainSelector: timelockConverter,
	}
	proposal, _, err := timelockProposal.Convert(ctx, convertersMap)
	s.Require().NoError(err)

	inspectorsMap := map[types.ChainSelector]sdk.Inspector{
		s.chainSelector: inspector,
	}

	quorum := int(proposerConfig.Quorum)
	signable, err := SignProposal(&proposal, inspectorsMap, proposerConfig.Keys, quorum)
	s.Require().NoError(err)

	quorumMet, err := signable.ValidateSignatures(ctx)
	s.Require().NoError(err)
	s.Require().True(quorumMet)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	suiEncoder := encoders[s.chainSelector].(*suisdk.Encoder)

	executor, err := suisdk.NewExecutor(s.client, s.signer, suiEncoder, s.entrypointArgEncoder, s.mcmsPackageID, role, s.mcmsObj, s.accountObj, s.registryObj, s.timelockObj)
	s.Require().NoError(err)

	executors := map[types.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}
	executable, err := mcmslib.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	_, err = executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)

	// Schedule Operations in Timelock
	for i := range proposal.Operations {
		_, execErr := executable.Execute(ctx, i)
		s.Require().NoError(execErr)
	}

	timelockExecutor, err := suisdk.NewTimelockExecutor(s.client, s.signer, s.entrypointArgEncoder, s.mcmsPackageID, s.registryObj, s.accountObj)
	s.Require().NoError(err)

	timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
		s.chainSelector: timelockExecutor,
	}
	timelockExecutable, execErr := mcmslib.NewTimelockExecutable(ctx, timelockProposal, timelockExecutors)
	s.Require().NoError(execErr)

	// Execute the scheduled batch through the timelock
	executeRes, terr := timelockExecutable.Execute(ctx, 0, mcmslib.WithCallProxy(s.timelockObj))
	s.Require().NoError(terr)

	result, ok := executeRes.RawData.(*models.SuiTransactionBlockResponse)
	s.Require().True(ok)

	newAddress, err := getUpgradedAddress(t, result, s.mcmsPackageID)
	s.Require().NoError(err)
	s.Require().NotEmpty(newAddress)

	return newAddress
}

func getUpgradedAddress(t *testing.T, result *models.SuiTransactionBlockResponse, mcmsPackageID string) (string, error) {
	t.Helper()

	if result == nil || result.Events == nil {
		return "", errors.New("result is nil or events are nil")
	}

	for _, event := range result.Events {
		if isUpgradeEvent(event, mcmsPackageID) {
			return processUpgradeEvent(t, event)
		}
	}

	return "", errors.New("upgrade receipt committed event not found")
}

// isUpgradeEvent checks if the event is an upgrade receipt committed event
func isUpgradeEvent(event models.SuiEventResponse, mcmsPackageID string) bool {
	return event.PackageId == mcmsPackageID &&
		event.TransactionModule == "mcms_deployer" &&
		strings.Contains(event.Type, "UpgradeReceiptCommitted")
}

// processUpgradeEvent processes an upgrade event and returns the new package address
func processUpgradeEvent(t *testing.T, event models.SuiEventResponse) (string, error) {
	t.Helper()

	t.Logf("Found UpgradeReceiptCommitted event - PackageId: %s, Module: %s, Type: %s",
		event.PackageId, event.TransactionModule, event.Type)

	if event.ParsedJson == nil {
		return "", errors.New("parsed json is nil")
	}

	t.Log("MCMS User Package Upgrade Details:")

	oldAddr := event.ParsedJson["old_package_address"]
	newAddr := event.ParsedJson["new_package_address"]
	oldVer := event.ParsedJson["old_version"]
	newVer := event.ParsedJson["new_version"]

	newAddress, err := validateAddressChange(t, oldAddr, newAddr)
	if err != nil {
		return "", err
	}

	err = validateVersionIncrement(t, oldVer, newVer)
	if err != nil {
		return "", err
	}

	return newAddress, nil
}

// validateAddressChange validates that the package address changed correctly
func validateAddressChange(t *testing.T, oldAddr, newAddr any) (string, error) {
	t.Helper()

	if oldAddr == nil || newAddr == nil {
		return "", fmt.Errorf("package addresses are nil")
	}

	oldAddrStr := fmt.Sprintf("%v", oldAddr)
	newAddrStr := fmt.Sprintf("%v", newAddr)

	if oldAddrStr == newAddrStr {
		t.Errorf("ERROR: Package address did not change! Old: %v, New: %v", oldAddr, newAddr)
		return "", fmt.Errorf("package address did not change")
	}

	t.Logf("✅ MCMS User package address changed successfully: %s → %s", oldAddrStr, newAddrStr)

	return newAddrStr, nil
}

// validateVersionIncrement validates that the version incremented correctly
func validateVersionIncrement(t *testing.T, oldVer, newVer any) error {
	t.Helper()

	if oldVer == nil || newVer == nil {
		return nil // Version validation is optional
	}

	oldVersion, oldParseOk := parseVersion(t, oldVer, "old")
	newVersion, newParseOk := parseVersion(t, newVer, "new")

	if !oldParseOk || !newParseOk {
		return nil // Skip validation if parsing failed
	}

	expectedVersion := oldVersion + 1
	if newVersion != expectedVersion {
		t.Errorf("ERROR: Version did not increment correctly! Old: %.0f, New: %.0f (expected %.0f)",
			oldVersion, newVersion, expectedVersion)

		return fmt.Errorf("version did not increment correctly")
	}

	t.Logf("✅ MCMS User version incremented correctly: %.0f → %.0f", oldVersion, newVersion)

	return nil
}

// parseVersion parses a version value from interface{} to float64
func parseVersion(t *testing.T, version any, versionType string) (float64, bool) {
	t.Helper()

	switch v := version.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed, true
		}
		t.Logf("Warning: Could not parse %s version string '%s' as number", versionType, v)

		return 0, false
	default:
		t.Logf("Warning: Unsupported %s version type: %T", versionType, v)
		return 0, false
	}
}
