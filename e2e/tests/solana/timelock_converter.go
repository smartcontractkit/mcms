//go:build e2e
// +build e2e

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

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/google/go-cmp/cmp"
	cpistub "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/external_program_cpi_stub"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/timelock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	testutils "github.com/smartcontractkit/mcms/e2e/utils/solana"
	"github.com/smartcontractkit/mcms/sdk"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

var testPDASeedTimelockConverter = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'l', 'o', 'c', 'k', 'c', 'o', 'n', 'v', 'e', 'r', 't', 'e', 'r'}

func (s *SolanaTestSuite) Test_TimelockConverter() {
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

	timelockSignerPDA, err := solanasdk.FindTimelockSignerPDA(s.TimelockProgramID, testPDASeedTimelockConverter)
	s.Require().NoError(err)

	testutils.FundAccounts(s.T(), ctx, []solana.PublicKey{mcmSignerPDA, timelockSignerPDA}, 1, s.SolanaClient)

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
	operation1ID := common.HexToHash("0x5177a0f841d284b16378790668accd634692d8f2b0a03170f9625c24b46866b7")
	operation1PDA, err := solanasdk.FindTimelockOperationPDA(s.TimelockProgramID, testPDASeedTimelockConverter, operation1ID)
	s.Require().NoError(err)
	operation2ID := common.HexToHash("0xf3b4a6b4ccfbef30504c7400bd075dc25780c122081b84a6a633dd152c572476")
	operation2PDA, err := solanasdk.FindTimelockOperationPDA(s.TimelockProgramID, testPDASeedTimelockConverter, operation2ID)
	s.Require().NoError(err)

	operation1BypasserPDA, err := solanasdk.FindTimelockBypasserOperationPDA(s.TimelockProgramID, testPDASeedTimelockConverter, operation1ID)
	s.Require().NoError(err)
	operation2BypasserPDA, err := solanasdk.FindTimelockBypasserOperationPDA(s.TimelockProgramID, testPDASeedTimelockConverter, operation2ID)
	s.Require().NoError(err)

	s.Require().NoError(err)
	metadata, err := solanasdk.NewChainMetadata(
		0,
		s.MCMProgramID,
		testPDASeedTimelockConverter,
		s.Roles[timelock.Proposer_Role].AccessController.PublicKey(),
		s.Roles[timelock.Canceller_Role].AccessController.PublicKey(),
		s.Roles[timelock.Bypasser_Role].AccessController.PublicKey())
	s.Require().NoError(err)
	timelockProposalBuilder := func() *mcms.TimelockProposalBuilder {
		return mcms.NewTimelockProposalBuilder().
			SetValidUntil(uint32(validUntil)).
			SetDescription("proposal to test the timelock proposal converter").
			SetOverridePreviousRoot(true).
			SetVersion("v1").
			SetDelay(types.NewDuration(1*time.Second)).
			AddTimelockAddress(s.ChainSelector, timelockAddress).
			AddChainMetadata(s.ChainSelector, metadata).
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
		timelockProposal, err := timelockProposalBuilder().SetAction(types.TimelockActionSchedule).Build()
		s.Require().NoError(err)

		proposerAC := s.Roles[timelock.Proposer_Role].AccessController.PublicKey()

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
				Data:              base64Decode(s.T(), "D2DZq3wEcfN0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAFF3oPhB0oSxY3h5BmiszWNGktjysKAxcPliXCS0aGa3AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB6QyuAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIAAAA="),
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
				Data:              base64Decode(s.T(), "w+bVh5CUjlV0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAFF3oPhB0oSxY3h5BmiszWNGktjysKAxcPliXCS0aGa3MNchUZRm56fGBpH3X0lzSVS+4C/bZwDdfwKhnelyzNsAAAAA"),
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
				Data:              base64Decode(s.T(), "TE1mg4gMLQV0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAFF3oPhB0oSxY3h5BmiszWNGktjysKAxcPliXCS0aGa3AAAAAAgAAADWLAT3DCnZbg=="),
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
				Data:              base64Decode(s.T(), "w+bVh5CUjlV0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAFF3oPhB0oSxY3h5BmiszWNGktjysKAxcPliXCS0aGa3MNchUZRm56fGBpH3X0lzSVS+4C/bZwDdfwKhnelyzNsAAAAA"),
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
				Data:              base64Decode(s.T(), "TE1mg4gMLQV0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAFF3oPhB0oSxY3h5BmiszWNGktjysKAxcPliXCS0aGa3AQAAAAkAAAARr5z9W60a5Hs="),
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
				Data:              base64Decode(s.T(), "P9AgYlW27Ix0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAFF3oPhB0oSxY3h5BmiszWNGktjysKAxcPliXCS0aGa3"),
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
				Data:              base64Decode(s.T(), "8oxXakfiViB0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAFF3oPhB0oSxY3h5BmiszWNGktjysKAxcPliXCS0aGa3AQAAAAAAAAA="),
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
				Data:              base64Decode(s.T(), "D2DZq3wEcfN0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAPO0prTM++8wUEx0AL0HXcJXgMEiCBuEpqYz3RUsVyR2UXeg+EHShLFjeHkGaKzNY0aS2PKwoDFw+WJcJLRoZrd6QyuAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAA="),
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
				Data:              base64Decode(s.T(), "w+bVh5CUjlV0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAPO0prTM++8wUEx0AL0HXcJXgMEiCBuEpqYz3RUsVyR2MNchUZRm56fGBpH3X0lzSVS+4C/bZwDdfwKhnelyzNsDAAAAb4Ezc71PQCcAP8nkYKXix7gp+hT0j5UUJCouW5tBzooAAcSFrcjukyvq73/bE9YO/KFljWLRDY+RGpHMk4NkhqT6AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"),
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
				Data:              base64Decode(s.T(), "TE1mg4gMLQV0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAPO0prTM++8wUEx0AL0HXcJXgMEiCBuEpqYz3RUsVyR2AAAAAAgAAAAMAokTFuuQRg=="),
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
				Data:              base64Decode(s.T(), "P9AgYlW27Ix0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAPO0prTM++8wUEx0AL0HXcJXgMEiCBuEpqYz3RUsVyR2"),
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
				Data:              base64Decode(s.T(), "8oxXakfiViB0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAPO0prTM++8wUEx0AL0HXcJXgMEiCBuEpqYz3RUsVyR2AQAAAAAAAAA="),
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
		s.Require().Equal([]common.Hash{mcms.ZERO_HASH, operation1ID}, gotPredecessors)
		s.Require().Empty(cmp.Diff(toJSONString(s.T(), wantProposal), toJSONString(s.T(), &gotProposal)))

		// TODO(gustavogama-cll): remove this; should refactor and use as base of a "full workflow" e2e test
		s.executeConvertedProposal(ctx, wallet, gotProposal, mcmAddress) // call ScheduleBatch
		s.waitForOperationToBeReady(ctx, testPDASeedTimelockConverter, operation2ID)
		s.executeTimelockProposal(ctx, wallet, timelockProposal)
	})

	s.Run("cancel", func() {
		timelockProposal, err := timelockProposalBuilder().SetAction(types.TimelockActionCancel).Build()
		s.Require().NoError(err)

		// build expected output Proposal
		s.Require().NoError(err)
		wantProposal, err := mcms.NewProposalBuilder().
			SetValidUntil(uint32(validUntil)).
			SetDescription("proposal to test the timelock proposal converter").
			SetOverridePreviousRoot(true).
			SetVersion("v1").
			AddChainMetadata(s.ChainSelector, metadata).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: cancel operation instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "6NvfKdvs3L50ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAFF3oPhB0oSxY3h5BmiszWNGktjysKAxcPliXCS0aGa3"),
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
				Data:              base64Decode(s.T(), "6NvfKdvs3L50ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAPO0prTM++8wUEx0AL0HXcJXgMEiCBuEpqYz3RUsVyR2"),
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
		s.Require().Equal([]common.Hash{mcms.ZERO_HASH, operation1ID}, gotPredecessors)
		s.Require().Empty(cmp.Diff(toJSONString(s.T(), wantProposal), toJSONString(s.T(), &gotProposal)))
	})

	s.Run("bypass", func() {
		timelockProposal, err := timelockProposalBuilder().SetAction(types.TimelockActionBypass).Build()
		s.Require().NoError(err)

		bypasserAC := s.Roles[timelock.Bypasser_Role].AccessController.PublicKey()

		// build expected output Proposal
		s.Require().NoError(err)
		wantProposal, err := mcms.NewProposalBuilder().
			SetValidUntil(uint32(validUntil)).
			SetDescription("proposal to test the timelock proposal converter").
			SetOverridePreviousRoot(true).
			SetVersion("v1").
			AddChainMetadata(s.ChainSelector, metadata).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op1: initialize operation instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "OhswzBPFPxp0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAFF3oPhB0oSxY3h5BmiszWNGktjysKAxcPliXCS0aGa3ekMrgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACAAAA"),
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
				Data:              base64Decode(s.T(), "MhHNrK+Mwyd0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAFF3oPhB0oSxY3h5BmiszWNGktjysKAxcPliXCS0aGa3MNchUZRm56fGBpH3X0lzSVS+4C/bZwDdfwKhnelyzNsAAAAA"),
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
				Data:              base64Decode(s.T(), "uOiX3m9118V0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAFF3oPhB0oSxY3h5BmiszWNGktjysKAxcPliXCS0aGa3AAAAAAgAAADWLAT3DCnZbg=="),
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
				Data:              base64Decode(s.T(), "MhHNrK+Mwyd0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAFF3oPhB0oSxY3h5BmiszWNGktjysKAxcPliXCS0aGa3MNchUZRm56fGBpH3X0lzSVS+4C/bZwDdfwKhnelyzNsAAAAA"),
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
				Data:              base64Decode(s.T(), "uOiX3m9118V0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAFF3oPhB0oSxY3h5BmiszWNGktjysKAxcPliXCS0aGa3AQAAAAkAAAARr5z9W60a5Hs="),
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
				Data:              base64Decode(s.T(), "LTfGM3wYqfp0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAFF3oPhB0oSxY3h5BmiszWNGktjysKAxcPliXCS0aGa3"),
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
				Data:              base64Decode(s.T(), "Wj5CBuOuHsJ0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAFF3oPhB0oSxY3h5BmiszWNGktjysKAxcPliXCS0aGa3"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op1Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation1BypasserPDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: timelockSignerPDA},
						{PublicKey: bypasserAC},
						{PublicKey: mcmSignerPDA, IsWritable: true},
					},
				}),
			}}).
			AddOperation(types.Operation{ChainSelector: s.ChainSelector, Transaction: types.Transaction{
				// op2: initialize operation instruction
				To:                s.TimelockProgramID.String(),
				Data:              base64Decode(s.T(), "OhswzBPFPxp0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAPO0prTM++8wUEx0AL0HXcJXgMEiCBuEpqYz3RUsVyR2ekMrgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAA"),
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
				Data:              base64Decode(s.T(), "MhHNrK+Mwyd0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAPO0prTM++8wUEx0AL0HXcJXgMEiCBuEpqYz3RUsVyR2MNchUZRm56fGBpH3X0lzSVS+4C/bZwDdfwKhnelyzNsDAAAAb4Ezc71PQCcAP8nkYKXix7gp+hT0j5UUJCouW5tBzooAAcSFrcjukyvq73/bE9YO/KFljWLRDY+RGpHMk4NkhqT6AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"),
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
				Data:              base64Decode(s.T(), "uOiX3m9118V0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAPO0prTM++8wUEx0AL0HXcJXgMEiCBuEpqYz3RUsVyR2AAAAAAgAAAAMAokTFuuQRg=="),
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
				Data:              base64Decode(s.T(), "LTfGM3wYqfp0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAPO0prTM++8wUEx0AL0HXcJXgMEiCBuEpqYz3RUsVyR2"),
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
				Data:              base64Decode(s.T(), "Wj5CBuOuHsJ0ZXN0LXRpbWVsb2NrY29udmVydGVyAAAAAAAAAAAAAPO0prTM++8wUEx0AL0HXcJXgMEiCBuEpqYz3RUsVyR2"),
				OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock", Tags: op2Tags},
				AdditionalFields: marshalAdditionalFields(s.T(), solanasdk.AdditionalFields{
					Accounts: []*solana.AccountMeta{
						{PublicKey: operation2BypasserPDA, IsWritable: true},
						{PublicKey: configPDA},
						{PublicKey: timelockSignerPDA},
						{PublicKey: bypasserAC},
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
		s.Require().Equal([]common.Hash{mcms.ZERO_HASH, operation1ID}, gotPredecessors)
		s.Require().Empty(cmp.Diff(toJSONString(s.T(), wantProposal), toJSONString(s.T(), &gotProposal)))
	})
}

func (s *SolanaTestSuite) executeConvertedProposal(
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

func (s *SolanaTestSuite) executeTimelockProposal(
	ctx context.Context, wallet solana.PrivateKey, timelockProposal *mcms.TimelockProposal,
) {
	timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
		s.ChainSelector: solanasdk.NewTimelockExecutor(s.SolanaClient, wallet),
	}
	timelockExecutable, err := mcms.NewTimelockExecutable(timelockProposal, timelockExecutors)
	s.Require().NoError(err)

	tx, err := timelockExecutable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().Contains(getTransactionLogs(s.T(), ctx, s.SolanaClient, tx.Hash), "Called `empty`")
	s.Require().Contains(getTransactionLogs(s.T(), ctx, s.SolanaClient, tx.Hash), "Called `u8_instruction_data`")

	tx, err = timelockExecutable.Execute(ctx, 1)
	s.Require().NoError(err)
	s.Require().Contains(getTransactionLogs(s.T(), ctx, s.SolanaClient, tx.Hash), "Called `account_mut`")
}

func newRandomAccount(t *testing.T) (solana.PrivateKey, solana.PublicKey) { //nolint:unused
	t.Helper()

	privateKey, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	return privateKey, privateKey.PublicKey()
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

func getTransactionLogs(t *testing.T, ctx context.Context, client *rpc.Client, signature string) string {
	t.Helper()

	opts := &rpc.GetTransactionOpts{Commitment: rpc.CommitmentConfirmed}
	result, err := client.GetTransaction(ctx, solana.MustSignatureFromBase58(signature), opts)
	require.NoError(t, err)

	return strings.Join(result.Meta.LogMessages, "\n")
}
