//go:build e2e

package sui

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/aptos-labs/aptos-go-sdk/bcs"
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
	SuiTestSuite
}

func (s *MCMSUserUpgradeTestSuite) Test_Sui_MCMSUser_UpgradeProposal() {
	s.T().Run("TimelockProposal - MCMS User Upgrade through Schedule", func(t *testing.T) {
		RunMCMSUserUpgradeProposal(s)
	})
}

type MCMSProposalExecutor struct {
	suite  *MCMSUserUpgradeTestSuite
	role   suisdk.TimelockRole
	config *RoleConfig
}

func NewMCMSProposalExecutor(suite *MCMSUserUpgradeTestSuite, role suisdk.TimelockRole, config *RoleConfig) *MCMSProposalExecutor {
	return &MCMSProposalExecutor{
		suite:  suite,
		role:   role,
		config: config,
	}
}

func (e *MCMSProposalExecutor) ExecuteProposal(ctx context.Context, description string, op types.BatchOperation) (*types.TransactionResult, error) {
	inspector, err := suisdk.NewInspector(e.suite.client, e.suite.signer, e.suite.mcmsPackageID, e.role)
	if err != nil {
		return nil, fmt.Errorf("creating inspector: %w", err)
	}

	currentOpCount, err := inspector.GetOpCount(ctx, e.suite.mcmsObj)
	if err != nil {
		return nil, fmt.Errorf("getting op count: %w", err)
	}

	proposalConfig := ProposalBuilderConfig{
		Version:        "v1",
		Description:    description,
		ChainSelector:  e.suite.chainSelector,
		McmsObjID:      e.suite.mcmsObj,
		TimelockObjID:  e.suite.timelockObj,
		McmsPackageID:  e.suite.mcmsPackageID,
		AccountObjID:   e.suite.accountObj,
		RegistryObjID:  e.suite.registryObj,
		Role:           e.role,
		CurrentOpCount: currentOpCount,
		Action:         types.TimelockActionSchedule,
		Delay:          &types.Duration{}, // Zero delay for immediate execution
	}

	proposalBuilder := CreateTimelockProposalBuilder(e.suite.T(), proposalConfig, []types.BatchOperation{op})
	timelockProposal, err := proposalBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("building timelock proposal: %w", err)
	}

	timelockConverter, err := suisdk.NewTimelockConverter()
	if err != nil {
		return nil, fmt.Errorf("creating timelock converter: %w", err)
	}

	convertersMap := map[types.ChainSelector]sdk.TimelockConverter{
		e.suite.chainSelector: timelockConverter,
	}
	proposal, _, err := timelockProposal.Convert(ctx, convertersMap)
	if err != nil {
		return nil, fmt.Errorf("converting proposal: %w", err)
	}

	inspectorsMap := map[types.ChainSelector]sdk.Inspector{
		e.suite.chainSelector: inspector,
	}

	quorum := int(e.config.Quorum)
	signable, err := SignProposal(&proposal, inspectorsMap, e.config.Keys, quorum)
	if err != nil {
		return nil, fmt.Errorf("signing proposal: %w", err)
	}

	quorumMet, err := signable.ValidateSignatures(ctx)
	if err != nil {
		return nil, fmt.Errorf("validating signatures: %w", err)
	}
	if !quorumMet {
		return nil, fmt.Errorf("quorum not met")
	}

	encoders, err := proposal.GetEncoders()
	if err != nil {
		return nil, fmt.Errorf("getting encoders: %w", err)
	}
	suiEncoder := encoders[e.suite.chainSelector].(*suisdk.Encoder)

	executor, err := suisdk.NewExecutor(e.suite.client, e.suite.signer, suiEncoder, e.suite.mcmsPackageID, e.role, e.suite.mcmsObj, e.suite.accountObj, e.suite.registryObj, e.suite.timelockObj)
	if err != nil {
		return nil, fmt.Errorf("creating executor: %w", err)
	}

	executors := map[types.ChainSelector]sdk.Executor{
		e.suite.chainSelector: executor,
	}
	executable, err := mcmslib.NewExecutable(&proposal, executors)
	if err != nil {
		return nil, fmt.Errorf("creating executable: %w", err)
	}

	_, err = executable.SetRoot(ctx, e.suite.chainSelector)
	if err != nil {
		return nil, fmt.Errorf("setting root: %w", err)
	}

	// Schedule Operations in Timelock
	for i := range proposal.Operations {
		_, execErr := executable.Execute(ctx, i)
		if execErr != nil {
			return nil, fmt.Errorf("executing operation %d: %w", i, execErr)
		}
	}

	timelockExecutor, err := suisdk.NewTimelockExecutor(e.suite.client, e.suite.signer, e.suite.mcmsPackageID, e.suite.registryObj, e.suite.accountObj)
	if err != nil {
		return nil, fmt.Errorf("creating timelock executor: %w", err)
	}

	timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
		e.suite.chainSelector: timelockExecutor,
	}
	timelockExecutable, execErr := mcmslib.NewTimelockExecutable(ctx, timelockProposal, timelockExecutors)
	if execErr != nil {
		return nil, fmt.Errorf("creating timelock executable: %w", execErr)
	}

	// Execute the scheduled batch through the timelock
	executeRes, terr := timelockExecutable.Execute(ctx, 0, mcmslib.WithCallProxy(e.suite.timelockObj))
	if terr != nil {
		return nil, fmt.Errorf("executing timelock batch: %w", terr)
	}

	return &executeRes, nil
}

func createMCMSAcceptOwnershipTransaction(suite *MCMSUserUpgradeTestSuite) (types.Transaction, error) {
	encodedCall, err := suite.mcmsAccount.Encoder().AcceptOwnershipAsTimelock(bind.Object{Id: suite.accountObj})
	if err != nil {
		return types.Transaction{}, fmt.Errorf("encoding accept ownership call: %w", err)
	}

	callBytes := suite.extractByteArgsFromEncodedCall(*encodedCall)

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
	s.DeployMCMSContract()
	s.DeployMCMSUserContract()

	ctx := context.Background()

	gasBudget := UpgradeGasBudget
	opts := &bind.CallOpts{
		Signer:           s.signer,
		GasBudget:        &gasBudget,
		WaitForExecution: true,
	}

	s.T().Logf("MCMS Package deployed: %s", s.mcmsPackageID)
	s.T().Logf("Registry Object ID: %s", s.registryObj)
	s.T().Logf("Deployer State ID: %s", s.depStateObj)
	s.T().Logf("MCMS State ID: %s", s.mcmsObj)
	s.T().Logf("Timelock ID: %s", s.timelockObj)
	s.T().Logf("Account State ID: %s", s.accountObj)
	s.T().Logf("Owner Cap ID: %s", s.ownerCapObj)

	s.T().Log("=== Phase 2: MCMS User Test Package Deployed ===")
	s.T().Logf("MCMS User Package deployed: %s", s.mcmsUserPackageId)
	s.T().Logf("MCMS User Owner Cap ID: %s", s.mcmsUserOwnerCapObj)
	s.T().Logf("MCMS User State Object ID: %s", s.stateObj)

	proposerCount := 3
	proposerQuorum := 2
	proposerConfig := CreateConfig(proposerCount, uint8(proposerQuorum))
	proposerConfigurer, err := suisdk.NewConfigurer(s.client, s.signer, suisdk.TimelockRoleProposer, s.mcmsPackageID, s.ownerCapObj, uint64(s.chainSelector))
	s.Require().NoError(err, "creating proposer configurer")
	_, err = proposerConfigurer.SetConfig(ctx, s.mcmsObj, proposerConfig.Config, true)
	s.Require().NoError(err, "setting proposer config")

	executeMCMSSelfOwnershipTransfer(s.T(), ctx, s, proposerConfig)

	deployerContract, err := module_mcms_deployer.NewMcmsDeployer(s.mcmsPackageID, s.client)
	s.Require().NoError(err)

	// Register UpgradeCap with MCMS deployer
	_, err = deployerContract.RegisterUpgradeCap(ctx, opts,
		bind.Object{Id: s.depStateObj},
		bind.Object{Id: s.registryObj},
		bind.Object{Id: s.mcmsUserUpgradeCapObj},
	)
	s.Require().NoError(err)

	// Check version
	version, err := s.mcmsUser.DevInspect().TypeAndVersion(ctx, opts)
	s.Require().NoError(err)
	s.Require().Equal(version, "MCMSUser 1.0.0")

	s.T().Log("=== Phase 5: Execute MCMS User Upgrade ===")

	newAddress, err := executeUpgradePTB(s.T(), ctx, s, proposerConfig, contracts.MCMSUserV2)
	s.Require().NoError(err, "Failed to execute two-phase MCMS User upgrade")

	// Verify upgrade to Version 2.0.0
	mcmsUserContract, err := module_mcms_user.NewMcmsUser(newAddress, s.client)
	s.Require().NoError(err)

	mcmsUserVersion, err := mcmsUserContract.DevInspect().TypeAndVersion(ctx, opts)
	s.Require().NoError(err)
	s.Require().Equal(mcmsUserVersion, "MCMSUser 2.0.0")

	s.T().Log("✅ MCMS User upgrade committed successfully - Complete MCMS → Upgrade workflow completed!")
}

// serializeAuthorizeUpgradeParams serializes parameters for authorize_upgrade function
// This follows the BCS serialization format expected by the Sui move function
func serializeAuthorizeUpgradeParams(policy uint8, digest []byte, packageAddress string) ([]byte, error) {
	// The authorize_upgrade function expects:
	// - policy: u8
	// - digest: vector<u8>
	// - package_address: address

	// Convert package address to bytes for BCS serialization
	packageAddrBytes, err := hex.DecodeString(strings.TrimPrefix(packageAddress, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode package address: %w", err)
	}

	// Use proper BCS serialization like the existing codebase
	return bcs.SerializeSingle(func(ser *bcs.Serializer) {
		ser.U8(policy)                   // u8 policy
		ser.WriteBytes(digest)           // vector<u8> digest
		ser.FixedBytes(packageAddrBytes) // address (32-byte fixed bytes)
	})
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

	proposalExecutor := NewMCMSProposalExecutor(s, suisdk.TimelockRoleProposer, proposerConfig)
	_, err = proposalExecutor.ExecuteProposal(ctx, "Accept MCMS self-ownership transfer via proposer with zero delay", op)
	s.Require().NoError(err, "Failed to execute MCMS self-ownership acceptance proposal")
}

func createUpgradeTransaction(compiledPackage bind.PackageArtifact, mcmsPackageID, depStateObj, registryObj, mcmsUserPackageId string) (types.Transaction, error) {
	upgradePolicy := uint8(0) // Compatible upgrade policy
	data, err := serializeAuthorizeUpgradeParams(upgradePolicy, compiledPackage.Digest, mcmsUserPackageId)
	if err != nil {
		return types.Transaction{}, fmt.Errorf("serializing authorize upgrade params: %w", err)
	}

	// Convert modules from base64 strings to raw bytes
	moduleBytes := make([][]byte, len(compiledPackage.Modules))
	for i, moduleBase64 := range compiledPackage.Modules {
		decoded, err := base64.StdEncoding.DecodeString(moduleBase64)
		if err != nil {
			return types.Transaction{}, fmt.Errorf("decoding module %d: %w", i, err)
		}
		moduleBytes[i] = decoded
	}

	depAddresses := make([]models.SuiAddress, len(compiledPackage.Dependencies))
	for i, dep := range compiledPackage.Dependencies {
		depAddresses[i] = models.SuiAddress(dep)
	}

	// Create transaction targeting mcms_deployer::authorize_upgrade with upgrade data
	return suisdk.NewTransactionWithUpgradeData(
		"mcms_deployer",             // Module name
		"authorize_upgrade",         // Function name
		mcmsPackageID,               // Package ID (mcms_deployer is in MCMS package)
		data,                        // BCS-encoded parameters
		"MCMS",                      // Contract type
		[]string{"upgrade", "mcms"}, // Tags
		depStateObj,                 // Main state object (DeployerState)
		[]string{registryObj},       // Internal objects (Registry for validation)
		moduleBytes,                 // Compiled modules for upgrade
		depAddresses,                // Dependencies for upgrade
		mcmsUserPackageId,           // Package being upgraded
	)
}

func executeUpgradePTB(t *testing.T, ctx context.Context, s *MCMSUserUpgradeTestSuite, proposerConfig *RoleConfig, packageType contracts.Package) (address string, err error) {
	t.Helper()

	signerAddr, err := s.signer.GetAddress()
	s.Require().NoError(err)

	compiledPackage, err := bind.CompilePackage(packageType, map[string]string{
		"mcms":       s.mcmsPackageID,
		"mcms_owner": signerAddr,
		"mcms_test":  "0x0", // Will be replaced with actual address during compilation
	})
	s.Require().NoError(err)

	return executeAtomicUpgradePTB(t, ctx, s, compiledPackage, proposerConfig)
}

// executeAtomicUpgradePTB creates a single atomic PTB that includes:
// 1. MCMS timelock execution → produces UpgradeTicket
// 2. Package upgrade using the UpgradeTicket → produces UpgradeReceipt
// 3. Commit upgrade using the UpgradeReceipt
func executeAtomicUpgradePTB(t *testing.T, ctx context.Context, s *MCMSUserUpgradeTestSuite, compiledPackage bind.PackageArtifact, proposerConfig *RoleConfig) address string {
	t.Helper()

	tx, err := createUpgradeTransaction(compiledPackage, s.mcmsPackageID, s.depStateObj, s.registryObj, s.mcmsUserPackageId)
	s.Require().NoError(err)

	op := types.BatchOperation{
		ChainSelector: s.chainSelector,
		Transactions:  []types.Transaction{tx},
	}

	proposalExecutor := NewMCMSProposalExecutor(s, suisdk.TimelockRoleProposer, proposerConfig)
	executeRes, err := proposalExecutor.ExecuteProposal(ctx, "Authorize MCMS User package upgrade via MCMS proposer with zero delay", op)
	s.Require().NoError(err, "Failed to execute upgrade authorization proposal")

	result, ok := executeRes.RawData.(*models.SuiTransactionBlockResponse)
	s.Require().True(ok)

	newAddress, err := getUpgradedAddress(t, result, s.mcmsPackageID)
	s.Require().NoError(err)
	s.Require().NotEmpty(newAddress)

	return newAddress
}

func getUpgradedAddress(t *testing.T, result *models.SuiTransactionBlockResponse, mcmsPackageID string) (address string, err error) {
	t.Helper()

	if result == nil || result.Events == nil {
		return "", fmt.Errorf("result is nil or events are nil")
	}

	var newAddress string

	for _, event := range result.Events {
		if event.PackageId == mcmsPackageID &&
			event.TransactionModule == "mcms_deployer" &&
			strings.Contains(event.Type, "UpgradeReceiptCommitted") {
			t.Logf("Found UpgradeReceiptCommitted event - PackageId: %s, Module: %s, Type: %s",
				event.PackageId, event.TransactionModule, event.Type)

			if event.ParsedJson == nil {
				return "", fmt.Errorf("parsed json is nil")
			}

			t.Log("MCMS User Package Upgrade Details:")

			oldAddr := event.ParsedJson["old_package_address"]
			newAddr := event.ParsedJson["new_package_address"]
			oldVer := event.ParsedJson["old_version"]
			newVer := event.ParsedJson["new_version"]

			if oldAddr != nil && newAddr != nil {
				oldAddrStr := fmt.Sprintf("%v", oldAddr)
				newAddrStr := fmt.Sprintf("%v", newAddr)

				if oldAddrStr == newAddrStr {
					t.Errorf("ERROR: Package address did not change! Old: %v, New: %v", oldAddr, newAddr)
					return "", fmt.Errorf("package address did not change")
				}
				newAddress = newAddrStr
				t.Logf("✅ MCMS User package address changed successfully: %s → %s", oldAddrStr, newAddrStr)
			}

			// Validate version increment
			if oldVer != nil && newVer != nil {
				var oldVersion, newVersion float64
				var parseOk bool

				switch v := oldVer.(type) {
				case float64:
					oldVersion = v
					parseOk = true
				case int:
					oldVersion = float64(v)
					parseOk = true
				case string:
					if parsed, err := strconv.ParseFloat(v, 64); err == nil {
						oldVersion = parsed
						parseOk = true
					} else {
						t.Logf("Warning: Could not parse old version string '%s' as number: %v", v, err)
					}
				}

				if parseOk {
					switch v := newVer.(type) {
					case float64:
						newVersion = v
					case int:
						newVersion = float64(v)
					case string:
						if parsed, err := strconv.ParseFloat(v, 64); err == nil {
							newVersion = parsed
						} else {
							t.Logf("Warning: Could not parse new version string '%s' as number: %v", v, err)
							parseOk = false
						}
					default:
						parseOk = false
					}
				}

				if parseOk {
					expectedVersion := oldVersion + 1
					if newVersion != expectedVersion {
						t.Errorf("ERROR: Version did not increment correctly! Old: %.0f, New: %.0f (expected %.0f)",
							oldVersion, newVersion, expectedVersion)

						return "", fmt.Errorf("version did not increment correctly")
					}
					t.Logf("✅ MCMS User version incremented correctly: %.0f → %.0f", oldVersion, newVersion)
				}

				return newAddress, nil
			}
		}
	}

	return "", fmt.Errorf("upgrade receipt committed event not found")
}
