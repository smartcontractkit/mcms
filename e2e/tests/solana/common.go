//go:build e2e
// +build e2e

package solanae2e

import (
	"context"
	"crypto/ecdsa"
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
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/access_controller"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/timelock"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/accesscontroller"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"
	timelockutils "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/timelock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

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

// SolanaTestSuite is the base test suite for all solana e2e tests
type SolanaTestSuite struct {
	suite.Suite
	e2e.TestSetup

	ChainSelector             types.ChainSelector
	MCMProgramID              solana.PublicKey
	TimelockProgramID         solana.PublicKey
	AccessControllerProgramID solana.PublicKey
	Roles                     timelockutils.RoleMap
}

// SetupMCM initializes the MCM account with the given PDA seed
func (s *SolanaTestSuite) SetupMCM(pdaSeed [32]byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s.T().Cleanup(cancel)

	mcm.SetProgramID(s.MCMProgramID)

	wallet, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	configPDA, err := solanasdk.FindConfigPDA(s.MCMProgramID, pdaSeed)
	s.Require().NoError(err)
	rootMetadataPDA, err := solanasdk.FindRootMetadataPDA(s.MCMProgramID, pdaSeed)
	s.Require().NoError(err)
	expiringRootAndOpCountPDA, err := solanasdk.FindExpiringRootAndOpCountPDA(s.MCMProgramID, pdaSeed)
	s.Require().NoError(err)

	// get program data account
	data, err := s.SolanaClient.GetAccountInfoWithOpts(ctx, s.MCMProgramID, &rpc.GetAccountInfoOpts{
		Commitment: rpc.CommitmentConfirmed,
	})
	s.Require().NoError(err)

	// decode program data
	var programData struct {
		DataType uint32
		Address  solana.PublicKey
	}
	err = bin.UnmarshalBorsh(&programData, data.Bytes())
	s.Require().NoError(err)

	ix, err := mcm.NewInitializeInstruction(
		uint64(s.ChainSelector),
		pdaSeed,
		configPDA,
		wallet.PublicKey(),
		solana.SystemProgramID,
		s.MCMProgramID,
		programData.Address,
		rootMetadataPDA,
		expiringRootAndOpCountPDA,
	).ValidateAndBuild()
	s.Require().NoError(err)

	testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, []solana.Instruction{ix}, wallet, rpc.CommitmentConfirmed)

	// get config and validate
	var configAccount mcm.MultisigConfig
	err = common.GetAccountDataBorshInto(ctx, s.SolanaClient, configPDA, rpc.CommitmentConfirmed, &configAccount)
	s.Require().NoError(err, "failed to get account data")

	s.Require().Equal(uint64(s.ChainSelector), configAccount.ChainId)
	s.Require().Equal(wallet.PublicKey(), configAccount.Owner)
}

func (s *SolanaTestSuite) SetupTimelockWorker(pdaSeed [32]byte, minDelay time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s.T().Cleanup(cancel)

	timelock.SetProgramID(s.TimelockProgramID)
	access_controller.SetProgramID(s.AccessControllerProgramID)

	admin, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	_, roleMap := timelockutils.TestRoleAccounts(2)
	s.Roles = roleMap

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
			roleMap[timelock.Proposer_Role].AccessController.PublicKey(),
			roleMap[timelock.Executor_Role].AccessController.PublicKey(),
			roleMap[timelock.Canceller_Role].AccessController.PublicKey(),
			roleMap[timelock.Bypasser_Role].AccessController.PublicKey(),
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
		s.Require().Equal(roleMap[timelock.Proposer_Role].AccessController.PublicKey(), configAccount.ProposerRoleAccessController, "ProposerRoleAccessController doesn't match")
		s.Require().Equal(roleMap[timelock.Executor_Role].AccessController.PublicKey(), configAccount.ExecutorRoleAccessController, "ExecutorRoleAccessController doesn't match")
		s.Require().Equal(roleMap[timelock.Canceller_Role].AccessController.PublicKey(), configAccount.CancellerRoleAccessController, "CancellerRoleAccessController doesn't match")
		s.Require().Equal(roleMap[timelock.Bypasser_Role].AccessController.PublicKey(), configAccount.BypasserRoleAccessController, "BypasserRoleAccessController doesn't match")
	})

	s.Run("setup roles", func() {
		for role, data := range roleMap {
			addresses := make([]solana.PublicKey, 0, len(data.Accounts))
			for _, account := range data.Accounts {
				addresses = append(addresses, account.PublicKey())
			}
			batchAddAccessIxs := s.getBatchAddAccessIxs(ctx, pdaSeed,
				data.AccessController.PublicKey(), role, addresses, admin, 10)
			s.Require().NoError(err)

			for _, ix := range batchAddAccessIxs {
				testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, []solana.Instruction{ix}, admin, rpc.CommitmentConfirmed)
			}

			for _, account := range data.Accounts {
				found, err := accesscontroller.HasAccess(ctx, s.SolanaClient, data.AccessController.PublicKey(),
					account.PublicKey(), rpc.CommitmentConfirmed)
				s.Require().NoError(err)
				s.Require().True(found, "Account %s not found in %s AccessList", account.PublicKey(), data.Role)
			}
		}
	})

}

// SetupSuite runs before the test suite
func (s *SolanaTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())
	s.MCMProgramID = solana.MustPublicKeyFromBase58(s.SolanaChain.SolanaPrograms["mcm"])
	s.TimelockProgramID = solana.MustPublicKeyFromBase58(s.SolanaChain.SolanaPrograms["timelock"])
	s.AccessControllerProgramID = solana.MustPublicKeyFromBase58(s.SolanaChain.SolanaPrograms["access_controller"])

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

func (s *SolanaTestSuite) getBatchAddAccessIxs(ctx context.Context, timelockID [32]byte,
	roleAcAccount solana.PublicKey, role timelock.Role, addresses []solana.PublicKey,
	authority solana.PrivateKey, chunkSize int) []solana.Instruction {
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
