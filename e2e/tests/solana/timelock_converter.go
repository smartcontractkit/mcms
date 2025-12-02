//go:build e2e

package solanae2e

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/go-cmp/cmp"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	cpistub "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/external_program_cpi_stub"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"
	solanaCommon "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"

	"github.com/smartcontractkit/mcms"
	e2eutils "github.com/smartcontractkit/mcms/e2e/utils/solana"
	"github.com/smartcontractkit/mcms/sdk"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

var testPDASeedTimelockConverter = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'l', 'o', 'c', 'k', 'c', 'o', 'n', 'v', 'e', 'r', 't', 'e', 'r'}

func (s *TestSuite) TestTimelockConverter() {
	// --- arrange ---
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	s.T().Cleanup(cancel)

	wallet, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	s.SetupMCM(testPDASeedTimelockConverter)
	s.SetupTimelock(testPDASeedTimelockConverter, 1*time.Second)
	s.SetupCPIStub(testPDASeedTimelockConverter)

	mcmSignerPDA, err := solanasdk.FindSignerPDA(s.MCMProgramID, testPDASeedTimelockConverter)
	s.Require().NoError(err)
	s.AssignRoleToAccounts(ctx, testPDASeedTimelockConverter, wallet, []solana.PublicKey{mcmSignerPDA},
		timelock.Proposer_Role)
	s.AssignRoleToAccounts(ctx, testPDASeedTimelockConverter, wallet, []solana.PublicKey{mcmSignerPDA},
		timelock.Bypasser_Role)

	timelockSignerPDA, err := solanasdk.FindTimelockSignerPDA(s.TimelockProgramID, testPDASeedTimelockConverter)
	s.Require().NoError(err)

	e2eutils.FundAccounts(s.T(), []solana.PublicKey{mcmSignerPDA, timelockSignerPDA}, 1, s.SolanaClient)

	validUntil := 2051222400 // 2035-01-01T12:00:00 UTC
	mcmAddress := solanasdk.ContractAddress(s.MCMProgramID, testPDASeedTimelockConverter)
	timelockAddress := solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockConverter)
	converters := map[types.ChainSelector]sdk.TimelockConverter{
		s.ChainSelector: solanasdk.TimelockConverter{},
	}

	// setup cpi-stub calls used as input
	emptyFnInstruction, err := cpistub.NewEmptyInstruction().ValidateAndBuild()
	s.Require().NoError(err)
	emptyFnTransaction, err := solanasdk.NewTransactionFromInstruction(emptyFnInstruction, "CPIStub",
		[]string{"cpi-stub-empty"})
	s.Require().NoError(err)

	u8DataInstruction, err := cpistub.NewU8InstructionDataInstruction(123).ValidateAndBuild()
	s.Require().NoError(err)
	u8DataTransaction, err := solanasdk.NewTransactionFromInstruction(u8DataInstruction, "CPIStub",
		[]string{"cpi-stub-u8data"})
	s.Require().NoError(err)

	u8ValuePDA, _, err := solana.FindProgramAddress([][]byte{[]byte("u8_value")}, s.CPIStubProgramID)
	s.Require().NoError(err)
	accountMutInstruction, err := cpistub.NewAccountMutInstruction(u8ValuePDA, timelockSignerPDA,
		solana.SystemProgramID).ValidateAndBuild()
	s.Require().NoError(err)
	accountMutTransaction, err := solanasdk.NewTransactionFromInstruction(accountMutInstruction, "CPIStub",
		[]string{"cpi-stub-account-mut"})
	s.Require().NoError(err)

	op1Tags := []string{"cpi-stub-empty", "cpi-stub-u8data"}
	op2Tags := []string{"cpi-stub-account-mut"}

	// operation ids and timelock pdas
	configPDA, err := solanasdk.FindTimelockConfigPDA(s.TimelockProgramID, testPDASeedTimelockConverter)
	s.Require().NoError(err)
	operation1ID := common.HexToHash("0x8716a69ad1b666929ef2d88cb978f93ec3d3492f053c0c638800f956f044df4e")
	operation1PDA, err := solanasdk.FindTimelockOperationPDA(s.TimelockProgramID, testPDASeedTimelockConverter, operation1ID)
	s.Require().NoError(err)
	operation2ID := common.HexToHash("0x21573e854c17e4bda196f6e518604c7776d96926b13ef8a211f9b4ba4c83beeb")
	operation2PDA, err := solanasdk.FindTimelockOperationPDA(s.TimelockProgramID, testPDASeedTimelockConverter, operation2ID)
	s.Require().NoError(err)

	chainMetadata := func() types.ChainMetadata {
		address := solanasdk.ContractAddress(s.MCMProgramID, testPDASeedTimelockConverter)
		opCount, err := solanasdk.NewInspector(s.SolanaClient).GetOpCount(ctx, address)
		s.Require().NoError(err)

		metadata, err := solanasdk.NewChainMetadata(opCount, s.MCMProgramID, testPDASeedTimelockConverter,
			s.Roles[timelock.Proposer_Role].AccessController.PublicKey(),
			s.Roles[timelock.Canceller_Role].AccessController.PublicKey(),
			s.Roles[timelock.Bypasser_Role].AccessController.PublicKey())
		s.Require().NoError(err)

		return metadata
	}
	timelockProposalBuilder := func() *mcms.TimelockProposalBuilder {
		return mcms.NewTimelockProposalBuilder().
			SetValidUntil(uint32(validUntil)).
			SetDescription("proposal to test the timelock proposal converter").
			SetOverridePreviousRoot(true).
			SetVersion("v1").
			SetDelay(types.NewDuration(1*time.Second)).
			AddTimelockAddress(s.ChainSelector, timelockAddress).
			AddOperation(types.BatchOperation{ // op1
				ChainSelector: s.ChainSelector,
				Transactions:  []types.Transaction{emptyFnTransaction, u8DataTransaction},
			}).
			AddOperation(types.BatchOperation{ // op2
				ChainSelector: s.ChainSelector,
				Transactions:  []types.Transaction{accountMutTransaction},
			})
	}

	s.Run("schedule", func() {
		cMetadata := chainMetadata()
		timelockProposal, err := timelockProposalBuilder().
			AddChainMetadata(s.ChainSelector, cMetadata).
			SetAction(types.TimelockActionSchedule).
			Build()
		s.Require().NoError(err)

		proposerAC := s.Roles[timelock.Proposer_Role].AccessController.PublicKey()

		// build expected output Proposal
		wantProposal, err := mcms.NewProposalBuilder().
			SetValidUntil(uint32(validUntil)).
			SetDescription("proposal to test the timelock proposal converter").
			SetOverridePreviousRoot(true).
			SetVersion("v1").
			AddChainMetadata(s.ChainSelector, chainMetadata()).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: initialize operation instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "D2DZq3wEcfN0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAIcWpprRtmaSnvLYjLl4+T7D00kvBTwMY4gA+VbwRN9OAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB6QyuAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIAAAA="),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op1Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation1PDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: proposerAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: initialize 1st timelock instruction ("empty" call)
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "w+bVh5CUjlV0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAIcWpprRtmaSnvLYjLl4+T7D00kvBTwMY4gA+VbwRN9OHZsMOrpAUG6C9rfHs8ScgG/m6HCBWCuHg3gToZkUBSoAAAAA"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op1Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation1PDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: proposerAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: append 1st timelock instruction ("empty" call)
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "TE1mg4gMLQV0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAIcWpprRtmaSnvLYjLl4+T7D00kvBTwMY4gA+VbwRN9OAAAAAAgAAADWLAT3DCnZbg=="),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op1Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation1PDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: proposerAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: initialize 2nd timelock instruction ("u8_value" call)
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "w+bVh5CUjlV0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAIcWpprRtmaSnvLYjLl4+T7D00kvBTwMY4gA+VbwRN9OHZsMOrpAUG6C9rfHs8ScgG/m6HCBWCuHg3gToZkUBSoAAAAA"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op1Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation1PDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: proposerAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: append 2nd timelock instruction ("u8_value" call)
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "TE1mg4gMLQV0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAIcWpprRtmaSnvLYjLl4+T7D00kvBTwMY4gA+VbwRN9OAQAAAAkAAAARr5z9W60a5Hs="),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op1Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation1PDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: proposerAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: finalize timelock operation instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "P9AgYlW27Ix0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAIcWpprRtmaSnvLYjLl4+T7D00kvBTwMY4gA+VbwRN9O"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op1Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation1PDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: proposerAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: schedule batch instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "8oxXakfiViB0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAIcWpprRtmaSnvLYjLl4+T7D00kvBTwMY4gA+VbwRN9OAQAAAAAAAAA="),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op1Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation1PDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: proposerAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op2: initialize operation instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "D2DZq3wEcfN0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAACFXPoVMF+S9oZb25RhgTHd22WkmsT74ohH5tLpMg77rhxammtG2ZpKe8tiMuXj5PsPTSS8FPAxjiAD5VvBE3056QyuAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAA="),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op2Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation2PDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: proposerAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op2: initialize 1st timelock instruction ("account_mut" call)
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "w+bVh5CUjlV0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAACFXPoVMF+S9oZb25RhgTHd22WkmsT74ohH5tLpMg77rHZsMOrpAUG6C9rfHs8ScgG/m6HCBWCuHg3gToZkUBSoDAAAAA0JRWm+fXW6dROTGvs1O7lJ17sSZwXdnofi7kQPhnScAAe5KiXXNoYn8wsyXPimFZjXn1bDQqROtngZAnboEB+hqAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op2Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation2PDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: proposerAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op2: append 1st timelock instruction ("account_mut" call)
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "TE1mg4gMLQV0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAACFXPoVMF+S9oZb25RhgTHd22WkmsT74ohH5tLpMg77rAAAAAAgAAAAMAokTFuuQRg=="),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op2Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation2PDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: proposerAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op2: finalize timelock operation instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "P9AgYlW27Ix0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAACFXPoVMF+S9oZb25RhgTHd22WkmsT74ohH5tLpMg77r"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op2Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation2PDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: proposerAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op2: schedule batch instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "8oxXakfiViB0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAACFXPoVMF+S9oZb25RhgTHd22WkmsT74ohH5tLpMg77rAQAAAAAAAAA="),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op2Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation2PDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: proposerAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
					},
				}),
			}}).
			Build()
		s.Require().NoError(err)

		// --- act ---
		gotProposal, gotPredecessors, err := timelockProposal.Convert(ctx, converters)
		s.Require().NoError(err)

		// --- assert ---
		s.Require().Equal([]common.Hash{mcms.ZeroHash, operation1ID}, gotPredecessors)
		s.Require().Empty(cmp.Diff(toJSONString(s.T(), wantProposal), toJSONString(s.T(), &gotProposal)))

		// --- act ---
		// TODO(gustavogama-cll): remove this; should refactor and use as base of a "full workflow" e2e test
		initialValue := readCPIStubU8Value(ctx, s.T(), s.SolanaClient, u8ValuePDA)
		s.executeConvertedProposal(ctx, wallet, gotProposal, mcmAddress) // call ScheduleBatch
		s.waitForOperationToBeReady(ctx, testPDASeedTimelockConverter, operation2ID)
		s.executeTimelockProposal(ctx, wallet, timelockProposal)

		// --- assert ---
		// check that the AccountMut instruction executed successfully
		finalValue := readCPIStubU8Value(ctx, s.T(), s.SolanaClient, u8ValuePDA)
		s.Require().Equal(initialValue+1, finalValue)
	})

	s.Run("cancel", func() {
		metadata := chainMetadata()
		timelockProposal, err := timelockProposalBuilder().
			AddChainMetadata(s.ChainSelector, metadata).
			SetAction(types.TimelockActionCancel).
			Build()
		s.Require().NoError(err)

		// build expected output Proposal
		wantProposal, err := mcms.NewProposalBuilder().
			SetValidUntil(uint32(validUntil)).
			SetDescription("proposal to test the timelock proposal converter").
			SetOverridePreviousRoot(true).
			SetVersion("v1").
			AddChainMetadata(s.ChainSelector, metadata).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: cancel operation instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "6NvfKdvs3L50ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAIcWpprRtmaSnvLYjLl4+T7D00kvBTwMY4gA+VbwRN9O"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op1Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation1PDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: s.Roles[timelock.Canceller_Role].AccessController.PublicKey()},
						{PublicKey: mcmSignerPDA, IsWritable: true},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op2: cancel operation instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "6NvfKdvs3L50ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAACFXPoVMF+S9oZb25RhgTHd22WkmsT74ohH5tLpMg77r"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op2Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation2PDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: s.Roles[timelock.Canceller_Role].AccessController.PublicKey()},
						{PublicKey: mcmSignerPDA, IsWritable: true},
					},
				}),
			}}).
			Build()
		s.Require().NoError(err)

		// --- act ---
		gotProposal, gotPredecessors, err := timelockProposal.Convert(ctx, converters)
		s.Require().NoError(err)

		// --- assert ---
		s.Require().Equal([]common.Hash{mcms.ZeroHash, operation1ID}, gotPredecessors)
		s.Require().Empty(cmp.Diff(toJSONString(s.T(), wantProposal), toJSONString(s.T(), &gotProposal)))
	})

	s.Run("bypass", func() {
		bypassOperation1ID := common.HexToHash("0x8716a69ad1b666929ef2d88cb978f93ec3d3492f053c0c638800f956f044df4e")
		operation1BypasserPDA, err := solanasdk.FindTimelockBypasserOperationPDA(s.TimelockProgramID, testPDASeedTimelockConverter, bypassOperation1ID)
		s.Require().NoError(err)
		bypassOperation2ID := common.HexToHash("0xfff6a19846af52bf2b905db2126ce0953f5a0d463c73cea09505c323c1df35d4")
		operation2BypasserPDA, err := solanasdk.FindTimelockBypasserOperationPDA(s.TimelockProgramID, testPDASeedTimelockConverter, bypassOperation2ID)
		s.Require().NoError(err)

		metadata := chainMetadata()
		timelockProposal, err := timelockProposalBuilder().
			AddChainMetadata(s.ChainSelector, metadata).
			SetAction(types.TimelockActionBypass).
			Build()
		s.Require().NoError(err)

		bypasserAC := s.Roles[timelock.Bypasser_Role].AccessController.PublicKey()

		// build expected output Proposal
		wantProposal, err := mcms.NewProposalBuilder().
			SetValidUntil(uint32(validUntil)).
			SetDescription("proposal to test the timelock proposal converter").
			SetOverridePreviousRoot(true).
			SetVersion("v1").
			AddChainMetadata(s.ChainSelector, metadata).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: initialize operation instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "OhswzBPFPxp0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAIcWpprRtmaSnvLYjLl4+T7D00kvBTwMY4gA+VbwRN9OekMrgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACAAAA"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op1Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation1BypasserPDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: bypasserAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: initialize 1st timelock instruction ("empty" call)
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "MhHNrK+Mwyd0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAIcWpprRtmaSnvLYjLl4+T7D00kvBTwMY4gA+VbwRN9OHZsMOrpAUG6C9rfHs8ScgG/m6HCBWCuHg3gToZkUBSoAAAAA"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op1Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation1BypasserPDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: bypasserAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: append 1st timelock instruction ("empty" call)
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "uOiX3m9118V0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAIcWpprRtmaSnvLYjLl4+T7D00kvBTwMY4gA+VbwRN9OAAAAAAgAAADWLAT3DCnZbg=="),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op1Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation1BypasserPDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: bypasserAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: initialize 2nd timelock instruction ("u8_value" call)
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "MhHNrK+Mwyd0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAIcWpprRtmaSnvLYjLl4+T7D00kvBTwMY4gA+VbwRN9OHZsMOrpAUG6C9rfHs8ScgG/m6HCBWCuHg3gToZkUBSoAAAAA"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op1Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation1BypasserPDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: bypasserAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: append 2nd timelock instruction ("u8_value" call)
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "uOiX3m9118V0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAIcWpprRtmaSnvLYjLl4+T7D00kvBTwMY4gA+VbwRN9OAQAAAAkAAAARr5z9W60a5Hs="),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op1Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation1BypasserPDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: bypasserAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: finalize timelock operation instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "LTfGM3wYqfp0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAIcWpprRtmaSnvLYjLl4+T7D00kvBTwMY4gA+VbwRN9O"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op1Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation1BypasserPDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: bypasserAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: bypass execute batch instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "Wj5CBuOuHsJ0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAIcWpprRtmaSnvLYjLl4+T7D00kvBTwMY4gA+VbwRN9O"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op1Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation1BypasserPDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: timelockSignerPDA},
						{PublicKey: bypasserAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: cpistub.ProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op2: initialize operation instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "OhswzBPFPxp0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAP/2oZhGr1K/K5BdshJs4JU/Wg1GPHPOoJUFwyPB3zXUekMrgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAA"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op2Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation2BypasserPDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: bypasserAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op2: initialize 1st timelock instruction ("account_mut" call)
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "MhHNrK+Mwyd0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAP/2oZhGr1K/K5BdshJs4JU/Wg1GPHPOoJUFwyPB3zXUHZsMOrpAUG6C9rfHs8ScgG/m6HCBWCuHg3gToZkUBSoDAAAAA0JRWm+fXW6dROTGvs1O7lJ17sSZwXdnofi7kQPhnScAAe5KiXXNoYn8wsyXPimFZjXn1bDQqROtngZAnboEB+hqAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op2Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation2BypasserPDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: bypasserAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op2: append 1st timelock instruction ("account_mut" call)
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "uOiX3m9118V0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAP/2oZhGr1K/K5BdshJs4JU/Wg1GPHPOoJUFwyPB3zXUAAAAAAgAAAAMAokTFuuQRg=="),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op2Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation2BypasserPDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: bypasserAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op2: finalize timelock operation instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "LTfGM3wYqfp0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAP/2oZhGr1K/K5BdshJs4JU/Wg1GPHPOoJUFwyPB3zXU"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op2Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation2BypasserPDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: bypasserAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op2: bypass execute batch instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "Wj5CBuOuHsJ0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAP/2oZhGr1K/K5BdshJs4JU/Wg1GPHPOoJUFwyPB3zXU"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op2Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation2BypasserPDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: timelockSignerPDA},
						{PublicKey: bypasserAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
						{PublicKey: cpistub.ProgramID},
						{PublicKey: u8ValuePDA, IsWritable: true},
						{PublicKey: timelockSignerPDA},
						{PublicKey: solana.SystemProgramID},
					},
				}),
			}}).
			Build()
		s.Require().NoError(err)

		// --- act: convert proposal ---
		gotProposal, gotPredecessors, err := timelockProposal.Convert(ctx, converters)
		s.Require().NoError(err)

		// --- assert ---
		s.Require().Equal([]common.Hash{mcms.ZeroHash, bypassOperation1ID}, gotPredecessors)
		s.Require().Empty(cmp.Diff(toJSONString(s.T(), wantProposal), toJSONString(s.T(), &gotProposal)))

		// --- act: executed converted proposal ---
		initialValue := readCPIStubU8Value(ctx, s.T(), s.SolanaClient, u8ValuePDA)
		s.executeConvertedProposal(ctx, wallet, gotProposal, mcmAddress) // call BypassExecuteBatch

		// --- assert ---
		// check that the AccountMut instruction executed successfully
		finalValue := readCPIStubU8Value(ctx, s.T(), s.SolanaClient, u8ValuePDA)
		s.Require().Equal(initialValue+1, finalValue)
	})
}

func (s *TestSuite) executeConvertedProposal(
	ctx context.Context, wallet solana.PrivateKey, gotProposal mcms.Proposal, mcmAddress string,
) {
	// set config
	signerEVMAccount := NewEVMTestAccount(s.T())
	mcmConfig := types.Config{Quorum: 1, Signers: []common.Address{signerEVMAccount.Address}}
	configurer := solanasdk.NewConfigurer(s.SolanaClient, wallet, s.ChainSelector)
	_, err := configurer.SetConfig(ctx, mcmAddress, &mcmConfig, true)
	s.Require().NoError(err)

	// sign
	inspectors := map[types.ChainSelector]sdk.Inspector{s.ChainSelector: solanasdk.NewInspector(s.SolanaClient)}
	signable, err := mcms.NewSignable(&gotProposal, inspectors)
	s.Require().NoError(err)
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(signerEVMAccount.PrivateKey))
	s.Require().NoError(err)

	// set root
	encoders, err := gotProposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.ChainSelector].(*solanasdk.Encoder)
	executors := map[types.ChainSelector]sdk.Executor{s.ChainSelector: solanasdk.NewExecutor(encoder, s.SolanaClient, wallet)}
	executable, err := mcms.NewExecutable(&gotProposal, executors)
	s.Require().NoError(err)

	_, err = executable.SetRoot(ctx, s.ChainSelector)
	s.Require().NoError(err)

	// execute all operations
	for i := range len(gotProposal.Operations) {
		_, err = executable.Execute(ctx, i)
		s.Require().NoError(err)
	}
}

func (s *TestSuite) executeTimelockProposal(
	ctx context.Context, wallet solana.PrivateKey, timelockProposal *mcms.TimelockProposal,
) {
	timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
		s.ChainSelector: solanasdk.NewTimelockExecutor(s.SolanaClient, wallet),
	}
	timelockExecutable, err := mcms.NewTimelockExecutable(ctx, timelockProposal, timelockExecutors)
	s.Require().NoError(err)

	tx, err := timelockExecutable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().Contains(getTransactionLogs(s.T(), ctx, s.SolanaClient, tx.Hash), "Called `empty`")
	s.Require().Contains(getTransactionLogs(s.T(), ctx, s.SolanaClient, tx.Hash), "Called `u8_instruction_data`")

	tx, err = timelockExecutable.Execute(ctx, 1)
	s.Require().NoError(err)
	s.Require().Contains(getTransactionLogs(s.T(), ctx, s.SolanaClient, tx.Hash), "Called `account_mut`")
}

func marshalAdditionalFields(t *testing.T, additionalFields solanasdk.AdditionalFields) []byte {
	t.Helper()
	marshalledBytes, err := json.Marshal(additionalFields)
	require.NoError(t, err)

	return marshalledBytes
}

func base64Decode(t *testing.T, str string) []byte {
	t.Helper()
	decodedBytes, err := base64.StdEncoding.DecodeString(str)
	require.NoError(t, err)

	return decodedBytes
}

func toJSONString(t *testing.T, proposal *mcms.Proposal) string {
	t.Helper()

	var buffer bytes.Buffer
	writer := io.Writer(&buffer)
	err := mcms.WriteProposal(writer, proposal)
	require.NoError(t, err)

	return buffer.String()
}

func getTransactionLogs(t *testing.T, client *rpc.Client, signature string) string {
	t.Helper()
	ctx := context.Background()

	opts := &rpc.GetTransactionOpts{Commitment: rpc.CommitmentConfirmed}
	result, err := client.GetTransaction(ctx, solana.MustSignatureFromBase58(signature), opts)
	require.NoError(t, err)

	return strings.Join(result.Meta.LogMessages, "\n")
}

func readCPIStubU8Value(ctx context.Context, t *testing.T, client *rpc.Client, pda solana.PublicKey) uint8 {
	t.Helper()

	var account cpistub.Value
	err := solanaCommon.GetAccountDataBorshInto(ctx, client, pda, rpc.CommitmentConfirmed, &account)
	require.NoError(t, err)

	return account.Value
}
