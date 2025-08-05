//go:build e2e
// +build e2e

package solanae2e

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/testutils"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/access_controller"
	cpistub "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/external_program_cpi_stub"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/mcm"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/accesscontroller"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"
	timelockutils "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/timelock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

// -----------------------------------------------------------------------------
// Constants and globals

// this key matches the public key in the config.toml so it gets funded by the genesis block
const privateKey = "DmPfeHBC8Brf8s5qQXi25bmJ996v6BHRtaLc6AH51yFGSqQpUMy1oHkbbXobPNBdgGH2F29PAmoq9ZZua4K9vCc"

// EVMTestAccount is data type wrapping attributes typically needed when managing
// ethereum accounts.
type EVMTestAccount struct {
	Address       evmCommon.Address
	HexAddress    string
	PrivateKey    *ecdsa.PrivateKey
	HexPrivateKey string
}

// NewEVMTestAccount generates a new ecdsa key and returns a TestAccount structure
// with the associated attributes.
func NewEVMTestAccount(t *testing.T) EVMTestAccount {
	t.Helper()

	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	publicKeyECDSA, ok := privateKey.Public().(*ecdsa.PublicKey)
	require.True(t, ok)

	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	return EVMTestAccount{
		Address:       address,
		HexAddress:    address.Hex(),
		PrivateKey:    privateKey,
		HexPrivateKey: hexutil.Encode(crypto.FromECDSA(privateKey))[2:],
	}
}

// generateTestEVMAccounts generates a slice of EVMTestAccount structures
func generateTestEVMAccounts(t *testing.T, numAccounts int) []EVMTestAccount {
	t.Helper()

	testAccounts := make([]EVMTestAccount, numAccounts)
	for i := range testAccounts {
		testAccounts[i] = NewEVMTestAccount(t)
	}

	slices.SortFunc(testAccounts, func(a, b EVMTestAccount) int {
		return strings.Compare(strings.ToLower(a.HexAddress), strings.ToLower(b.HexAddress))
	})

	return testAccounts
}

type RoleMap map[timelock.Role]RoleAccounts

type RoleAccounts struct {
	Role             timelock.Role
	Accounts         []solana.PrivateKey
	AccessController solana.PrivateKey
}

// SolanaTestSuite is the base test suite for all solana e2e tests
type SolanaTestSuite struct {
	suite.Suite
	e2e.TestSetup

	ChainSelector             types.ChainSelector
	MCMProgramID              solana.PublicKey
	TimelockProgramID         solana.PublicKey
	AccessControllerProgramID solana.PublicKey
	CPIStubProgramID          solana.PublicKey
	Roles                     RoleMap
}

// SetupSuite runs before the test suite
func (s *SolanaTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())
	s.MCMProgramID = solana.MustPublicKeyFromBase58(s.SolanaChain.SolanaPrograms["mcm"])
	s.TimelockProgramID = solana.MustPublicKeyFromBase58(s.SolanaChain.SolanaPrograms["timelock"])
	s.AccessControllerProgramID = solana.MustPublicKeyFromBase58(s.SolanaChain.SolanaPrograms["access_controller"])
	s.CPIStubProgramID = solana.MustPublicKeyFromBase58(s.SolanaChain.SolanaPrograms["external_program_cpi_stub"])

	details, err := cselectors.GetChainDetailsByChainIDAndFamily(s.SolanaChain.ChainID, cselectors.FamilySolana)
	s.Require().NoError(err)
	s.ChainSelector = types.ChainSelector(details.ChainSelector)
}

func (s *SolanaTestSuite) SetupTest() {
	// reset all programID to a random key
	// this ensures all methods in the sdk correctly sets the programID itself
	// and not rely on the global programID to be set by something else
	key, err := solana.NewRandomPrivateKey()
	s.Require().NoError(err)
	mcm.SetProgramID(key.PublicKey())

	key, err = solana.NewRandomPrivateKey()
	s.Require().NoError(err)
	timelock.SetProgramID(key.PublicKey())

	key, err = solana.NewRandomPrivateKey()
	s.Require().NoError(err)
	access_controller.SetProgramID(key.PublicKey())
}

// InitializeMCMProgram public helper to initialize the MCM program
func InitializeMCMProgram(ctx context.Context, t *testing.T, solanaClient *rpc.Client, programID solana.PublicKey, pdaSeed [32]byte, chainSelector uint64) {
	t.Helper()

	mcm.SetProgramID(programID)
	wallet, err := solana.PrivateKeyFromBase58(privateKey)
	require.NoError(t, err)

	configPDA, err := solanasdk.FindConfigPDA(programID, pdaSeed)
	require.NoError(t, err)
	rootMetadataPDA, err := solanasdk.FindRootMetadataPDA(programID, pdaSeed)
	require.NoError(t, err)
	expiringRootAndOpCountPDA, err := solanasdk.FindExpiringRootAndOpCountPDA(programID, pdaSeed)
	require.NoError(t, err)

	// get program data account
	data, err := solanaClient.GetAccountInfoWithOpts(ctx, programID, &rpc.GetAccountInfoOpts{
		Commitment: rpc.CommitmentConfirmed,
	})
	require.NoError(t, err)
	// decode program data

	var programData struct {
		DataType uint32
		Address  solana.PublicKey
	}
	err = bin.UnmarshalBorsh(&programData, data.Bytes())
	require.NoError(t, err)

	ix, err := mcm.NewInitializeInstruction(
		chainSelector,
		pdaSeed,
		configPDA,
		wallet.PublicKey(),
		solana.SystemProgramID,
		programID,
		programData.Address,
		rootMetadataPDA,
		expiringRootAndOpCountPDA,
	).ValidateAndBuild()
	require.NoError(t, err)

	testutils.SendAndConfirm(ctx, t, solanaClient, []solana.Instruction{ix}, wallet, rpc.CommitmentConfirmed)

	// get config and validate
	var configAccount mcm.MultisigConfig
	err = common.GetAccountDataBorshInto(ctx, solanaClient, configPDA, rpc.CommitmentConfirmed, &configAccount)
	require.NoError(t, err)

	require.Equal(t, chainSelector, configAccount.ChainId)
	require.Equal(t, wallet.PublicKey(), configAccount.Owner)
}
func (s *SolanaTestSuite) SetupMCM(pdaSeed [32]byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s.T().Cleanup(cancel)

	InitializeMCMProgram(ctx, s.T(), s.SolanaClient, s.MCMProgramID, pdaSeed, uint64(s.ChainSelector))
}

func (s *SolanaTestSuite) SetupTimelock(pdaSeed [32]byte, minDelay time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s.T().Cleanup(cancel)

	timelock.SetProgramID(s.TimelockProgramID)
	access_controller.SetProgramID(s.AccessControllerProgramID)

	admin, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	_, roleMap := timelockutils.TestRoleAccounts(2)
	s.Roles = RoleMap{}
	for k, v := range roleMap {
		s.Roles[timelock.Role(k)] = RoleAccounts{
			Role:             timelock.Role(v.Role),
			Accounts:         v.Accounts,
			AccessController: v.AccessController,
		}
	}

	s.Run("init access controller", func() {
		for _, data := range roleMap {
			initAccIxs := s.getInitAccessControllersIxs(ctx, data.AccessController.PublicKey(), admin)

			testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, initAccIxs, admin, rpc.CommitmentConfirmed, common.AddSigners(data.AccessController))

			var ac access_controller.AccessController
			err = common.GetAccountDataBorshInto(ctx, s.SolanaClient, data.AccessController.PublicKey(), rpc.CommitmentConfirmed, &ac)
			if err != nil {
				s.Require().NoError(err, "failed to get account info")
			}
		}
	})

	s.Run("init timelock", func() {
		data, accErr := s.SolanaClient.GetAccountInfoWithOpts(ctx, s.TimelockProgramID, &rpc.GetAccountInfoOpts{
			Commitment: rpc.CommitmentConfirmed,
		})
		s.Require().NoError(accErr)

		var programData struct {
			DataType uint32
			Address  solana.PublicKey
		}
		s.Require().NoError(bin.UnmarshalBorsh(&programData, data.Bytes()))

		pda, err2 := solanasdk.FindTimelockConfigPDA(s.TimelockProgramID, pdaSeed)
		s.Require().NoError(err2)

		initTimelockIx, err3 := timelock.NewInitializeInstruction(
			pdaSeed,
			uint64(minDelay.Seconds()),
			pda,
			admin.PublicKey(),
			solana.SystemProgramID,
			s.TimelockProgramID,
			programData.Address,
			s.AccessControllerProgramID,
			s.Roles[timelock.Proposer_Role].AccessController.PublicKey(),
			s.Roles[timelock.Executor_Role].AccessController.PublicKey(),
			s.Roles[timelock.Canceller_Role].AccessController.PublicKey(),
			s.Roles[timelock.Bypasser_Role].AccessController.PublicKey(),
		).ValidateAndBuild()
		s.Require().NoError(err3)

		testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, []solana.Instruction{initTimelockIx}, admin, rpc.CommitmentConfirmed)

		var configAccount timelock.Config
		err = common.GetAccountDataBorshInto(ctx, s.SolanaClient, pda, rpc.CommitmentConfirmed, &configAccount)
		if err != nil {
			s.Require().NoError(err, "failed to get account info")
		}

		s.Require().Equal(admin.PublicKey(), configAccount.Owner, "Owner doesn't match")
		s.Require().Equal(uint64(minDelay.Seconds()), configAccount.MinDelay, "MinDelay doesn't match")
		s.Require().Equal(s.Roles[timelock.Proposer_Role].AccessController.PublicKey(), configAccount.ProposerRoleAccessController, "ProposerRoleAccessController doesn't match")
		s.Require().Equal(s.Roles[timelock.Executor_Role].AccessController.PublicKey(), configAccount.ExecutorRoleAccessController, "ExecutorRoleAccessController doesn't match")
		s.Require().Equal(s.Roles[timelock.Canceller_Role].AccessController.PublicKey(), configAccount.CancellerRoleAccessController, "CancellerRoleAccessController doesn't match")
		s.Require().Equal(s.Roles[timelock.Bypasser_Role].AccessController.PublicKey(), configAccount.BypasserRoleAccessController, "BypasserRoleAccessController doesn't match")
	})

	s.Run("setup roles", func() {
		for _, roleAccounts := range s.Roles {
			accounts := make([]solana.PublicKey, 0, len(roleAccounts.Accounts))
			for _, account := range roleAccounts.Accounts {
				accounts = append(accounts, account.PublicKey())
			}

			s.AssignRoleToAccounts(ctx, pdaSeed, admin, accounts, roleAccounts.Role)

			for _, account := range roleAccounts.Accounts {
				found, err := accesscontroller.HasAccess(ctx, s.SolanaClient, roleAccounts.AccessController.PublicKey(),
					account.PublicKey(), rpc.CommitmentConfirmed)
				s.Require().NoError(err)
				s.Require().True(found, "Account %s not found in %s AccessList", account.PublicKey(), roleAccounts.Role)
			}
		}
	})
}

func (s *SolanaTestSuite) AssignRoleToAccounts(
	ctx context.Context, pdaSeed solanasdk.PDASeed, auth solana.PrivateKey,
	accounts []solana.PublicKey, role timelock.Role,
) {
	instructions := s.getBatchAddAccessIxs(ctx, pdaSeed, s.Roles[role].AccessController.PublicKey(),
		role, accounts, auth, 1)
	testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, instructions, auth, rpc.CommitmentConfirmed)
}

func (s *SolanaTestSuite) SetupCPIStub(pdaSeed solanasdk.PDASeed) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s.T().Cleanup(cancel)

	cpistub.SetProgramID(s.CPIStubProgramID)

	admin, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	valuePDA, _, err := solana.FindProgramAddress([][]byte{[]byte("u8_value")}, s.CPIStubProgramID)
	s.Require().NoError(err)

	instruction, err := cpistub.NewInitializeInstruction(valuePDA, admin.PublicKey(), solana.SystemProgramID).
		ValidateAndBuild()
	s.Require().NoError(err)

	_ = testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, []solana.Instruction{instruction}, admin,
		rpc.CommitmentConfirmed)
}

func (s *SolanaTestSuite) getInitAccessControllersIxs(ctx context.Context, roleAcAccount solana.PublicKey, authority solana.PrivateKey) []solana.Instruction {
	ixs := []solana.Instruction{}

	dataSize := uint64(8 + 32 + 32 + ((32 * 64) + 8)) // discriminator + owner + proposed owner + access_list (64 max addresses + length)
	rentExemption, err := s.SolanaClient.GetMinimumBalanceForRentExemption(ctx, dataSize, rpc.CommitmentConfirmed)
	s.Require().NoError(err)

	createAccountInstruction, err := system.NewCreateAccountInstruction(rentExemption, dataSize,
		s.AccessControllerProgramID, authority.PublicKey(), roleAcAccount).ValidateAndBuild()
	s.Require().NoError(err)
	ixs = append(ixs, createAccountInstruction)

	initializeInstruction, err := access_controller.NewInitializeInstruction(roleAcAccount,
		authority.PublicKey()).ValidateAndBuild()
	s.Require().NoError(err)
	ixs = append(ixs, initializeInstruction)

	return ixs
}

func (s *SolanaTestSuite) getBatchAddAccessIxs(
	ctx context.Context, timelockID [32]byte, roleAcAccount solana.PublicKey, role timelock.Role,
	addresses []solana.PublicKey, authority solana.PrivateKey, chunkSize int,
) []solana.Instruction {
	var ac access_controller.AccessController
	err := common.GetAccountDataBorshInto(ctx, s.SolanaClient, roleAcAccount, rpc.CommitmentConfirmed, &ac)
	s.Require().NoError(err)

	ixs := []solana.Instruction{}
	for i := 0; i < len(addresses); i += chunkSize {
		end := i + chunkSize
		if end > len(addresses) {
			end = len(addresses)
		}
		chunk := addresses[i:end]
		pda, err := solanasdk.FindTimelockConfigPDA(s.TimelockProgramID, timelockID)
		s.Require().NoError(err)

		ix := timelock.NewBatchAddAccessInstruction(
			timelockID,
			role,
			pda,
			s.AccessControllerProgramID,
			roleAcAccount,
			authority.PublicKey(),
		)
		for _, address := range chunk {
			ix.Append(solana.Meta(address))
		}
		vIx, err := ix.ValidateAndBuild()
		s.Require().NoError(err)

		ixs = append(ixs, vIx)
	}

	return ixs
}

func (s *SolanaTestSuite) initOperation(ctx context.Context, op timelockutils.Operation, timelockID [32]byte, auth solana.PrivateKey) {
	ixs := s.getInitOperationIxs(timelockID, op, auth.PublicKey())
	tx := testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, ixs, auth, rpc.CommitmentConfirmed)
	s.Require().NotNil(tx)
}

func (s *SolanaTestSuite) getInitOperationIxs(timelockID [32]byte, op timelockutils.Operation, authority solana.PublicKey) []solana.Instruction {
	configPDA, err := solanasdk.FindTimelockConfigPDA(s.TimelockProgramID, timelockID)
	s.Require().NoError(err)
	operationPDA, err := solanasdk.FindTimelockOperationPDA(s.TimelockProgramID, timelockID, op.OperationID())
	s.Require().NoError(err)

	proposerAC := s.Roles[timelock.Proposer_Role].AccessController.PublicKey()

	ixs := []solana.Instruction{}
	initOpIx, ioErr := timelock.NewInitializeOperationInstruction(
		timelockID,
		op.OperationID(),
		op.Predecessor,
		op.Salt,
		op.IxsCountU32(),
		operationPDA,
		configPDA,
		proposerAC,
		authority,
		solana.SystemProgramID,
	).ValidateAndBuild()
	s.Require().NoError(ioErr)

	ixs = append(ixs, initOpIx)

	for i, instruction := range op.ToInstructionData() {
		accounts := make([]timelock.InstructionAccount, len(instruction.Accounts))
		for i := range instruction.Accounts {
			accounts[i] = timelock.InstructionAccount{
				Pubkey:     instruction.Accounts[i].Pubkey,
				IsSigner:   instruction.Accounts[i].IsSigner,
				IsWritable: instruction.Accounts[i].IsWritable,
			}
		}

		initIx, apErr := timelock.NewInitializeInstructionInstruction(
			timelockID,
			op.OperationID(),
			instruction.ProgramId,
			accounts,
			operationPDA,
			configPDA,
			proposerAC,
			authority,
			solana.SystemProgramID,
		).ValidateAndBuild()
		s.Require().NoError(apErr)

		ixs = append(ixs, initIx)

		rawData := instruction.Data
		offset := 0

		for offset < len(rawData) {
			end := offset + solanasdk.AppendIxDataChunkSize()
			if end > len(rawData) {
				end = len(rawData)
			}
			chunk := rawData[offset:end]

			appendIx, err := timelock.NewAppendInstructionDataInstruction(
				timelockID,
				op.OperationID(),
				uint32(i), // which instruction index we are chunking
				chunk,     // partial data
				operationPDA,
				configPDA,
				proposerAC,
				authority,
				solana.SystemProgramID,
			).ValidateAndBuild()
			s.Require().NoError(err)
			ixs = append(ixs, appendIx)

			offset = end
		}
	}
	finOpIx, foErr := timelock.NewFinalizeOperationInstruction(
		timelockID,
		op.OperationID(),
		operationPDA,
		configPDA,
		proposerAC,
		authority,
	).ValidateAndBuild()
	s.Require().NoError(foErr)

	ixs = append(ixs, finOpIx)

	return ixs
}

func (s *SolanaTestSuite) scheduleOperation(ctx context.Context, timelockID [32]byte, delay time.Duration, opID [32]byte) {
	operationPDA, err := solanasdk.FindTimelockOperationPDA(s.TimelockProgramID, timelockID, opID)
	s.Require().NoError(err)

	configPDA, err := solanasdk.FindTimelockConfigPDA(s.TimelockProgramID, timelockID)
	s.Require().NoError(err)

	admin, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	sbix, sberr := timelock.NewScheduleBatchInstruction(
		timelockID,
		opID,
		uint64(delay.Seconds()),
		operationPDA,
		configPDA,
		s.Roles[timelock.Proposer_Role].AccessController.PublicKey(),
		admin.PublicKey(),
	).ValidateAndBuild()
	s.Require().NoError(sberr)

	tx := testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, []solana.Instruction{sbix}, admin, rpc.CommitmentConfirmed)
	s.Require().NotNil(tx)
}

func (s *SolanaTestSuite) executeOperation(ctx context.Context, timelockID [32]byte, opID [32]byte) {
	operationPDA, err := solanasdk.FindTimelockOperationPDA(s.TimelockProgramID, timelockID, opID)
	s.Require().NoError(err)

	configPDA, err := solanasdk.FindTimelockConfigPDA(s.TimelockProgramID, timelockID)
	s.Require().NoError(err)

	signerPDA, err := solanasdk.FindTimelockSignerPDA(s.TimelockProgramID, timelockID)
	s.Require().NoError(err)

	admin, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	ix := timelock.NewExecuteBatchInstruction(
		timelockID,
		opID,
		operationPDA,
		[32]byte{},
		configPDA,
		signerPDA,
		s.Roles[timelock.Executor_Role].AccessController.PublicKey(),
		admin.PublicKey(),
	)

	vIx, err := ix.ValidateAndBuild()
	s.Require().NoError(err)

	tx := testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, []solana.Instruction{vIx}, admin, rpc.CommitmentConfirmed)
	s.Require().NotNil(tx)
}

func (s *SolanaTestSuite) waitForOperationToBeReady(ctx context.Context, timelockID [32]byte, opID [32]byte) {
	const maxAttempts = 20
	const pollInterval = 500 * time.Millisecond
	const timeBuffer = 2 * time.Second

	opPDA, err := solanasdk.FindTimelockOperationPDA(s.TimelockProgramID, timelockID, opID)
	s.Require().NoError(err)

	var opAccount timelock.Operation
	err = common.GetAccountDataBorshInto(ctx, s.SolanaClient, opPDA, rpc.CommitmentConfirmed, &opAccount)
	s.Require().NoError(err)

	if opAccount.State == timelock.Done_OperationState {
		return
	}

	scheduledTime := time.Unix(int64(opAccount.Timestamp), 0)

	// add buffer to scheduled time to ensure blockchain has advanced enough
	scheduledTimeWithBuffer := scheduledTime.Add(timeBuffer)

	for range maxAttempts {
		currentTime, err := common.GetBlockTime(ctx, s.SolanaClient, rpc.CommitmentConfirmed)
		s.Require().NoError(err)

		if currentTime.Time().After(scheduledTimeWithBuffer) || currentTime.Time().Equal(scheduledTimeWithBuffer) {
			return
		}

		time.Sleep(pollInterval)
	}

	s.Require().Fail(fmt.Sprintf("Operation %s is not ready after %d attempts", opID, maxAttempts))
}

func (s *SolanaTestSuite) contextWithLogger() context.Context {
	return context.WithValue(context.Background(), solanasdk.ContextLoggerValue, zap.NewNop().Sugar())
}
