//go:build e2e
// +build e2e

package solanae2e

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/google/go-cmp/cmp"
	solanaCommon "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"

	mcmsSolana "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

var testPDASeedSetConfigTest = [32]byte{'t', 'e', 's', 't', '-', 's', 'e', 't', 'c', 'o', 'n', 'f'}
var testPDASeedSetConfigSkipTxTest = [32]byte{'t', 'e', 's', 't', '-', 's', 'e', 't', 'c', 'o', 'n', 'f', '-', 's', 'k', 'i', 'p', 't', 'x'}

// Test_Solana_SetConfig tests the SetConfig functionality by setting a config on the MCM program
func (s *SolanaTestSuite) Test_Solana_SetConfig() {
	// --- arrange ---
	ctx := context.Background()
	s.SetupMCM(testPDASeedSetConfigTest)
	programID, err := solana.PublicKeyFromBase58(s.SolanaChain.SolanaPrograms["mcm"])
	s.Require().NoError(err)
	auth, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)
	mcmAddress := mcmsSolana.ContractAddress(programID, testPDASeedSetConfigTest)

	testEVMAccounts := generateTestEVMAccounts(s.T(), 14)
	config := types.Config{
		Quorum: 2,
		Signers: []common.Address{
			testEVMAccounts[0].Address,
			testEVMAccounts[1].Address,
			testEVMAccounts[2].Address,
		},
		GroupSigners: []types.Config{
			{
				Quorum: 4,
				Signers: []common.Address{
					testEVMAccounts[3].Address,
					testEVMAccounts[4].Address,
					testEVMAccounts[5].Address,
					testEVMAccounts[6].Address,
					testEVMAccounts[7].Address,
				},
				GroupSigners: []types.Config{
					{
						Quorum: 1,
						Signers: []common.Address{
							testEVMAccounts[8].Address,
							testEVMAccounts[9].Address,
						},
						GroupSigners: []types.Config{},
					},
				},
			},
			{
				Quorum: 3,
				Signers: []common.Address{
					testEVMAccounts[10].Address,
					testEVMAccounts[11].Address,
					testEVMAccounts[12].Address,
					testEVMAccounts[13].Address,
				},
				GroupSigners: []types.Config{},
			},
		},
	}

	s.Run("send and confirm instructions to the blockchain", func () {
		// --- act ---
		configurer := mcmsSolana.NewConfigurer(s.SolanaClient, auth, s.ChainSelector)
		result, err := configurer.SetConfig(ctx, mcmAddress, &config, true)
		s.Require().NoError(err)
		_, err = solana.SignatureFromBase58(result.Hash)
		s.Require().NoError(err)

		// --- assert ---
		gotConfig, err := mcmsSolana.NewInspector(s.SolanaClient).GetConfig(ctx, mcmAddress)
		s.Require().NoError(err)
		s.Require().NotNil(gotConfig)
		s.Require().Empty(cmp.Diff(config, *gotConfig))
	})

	s.Run("do NOT send transactions to the blockchain", func () {
		skipTxConfig := types.Config{
			Quorum: 1,
			Signers: []common.Address{
				testEVMAccounts[0].Address,
			},
			GroupSigners: []types.Config{},
		}

		// --- act ---
		configurer := mcmsSolana.NewConfigurer(s.SolanaClient, auth, s.ChainSelector, mcmsSolana.SkipTransaction())
		result, err := configurer.SetConfig(ctx, mcmAddress, &skipTxConfig, true)
		s.Require().NoError(err)

		// --- assert ---
		s.Require().Empty(err, result.Hash)

		gotConfig, err := mcmsSolana.NewInspector(s.SolanaClient).GetConfig(ctx, mcmAddress)
		s.Require().NoError(err)
		s.Require().NotNil(gotConfig)
		s.Require().Empty(cmp.Diff(config, *gotConfig)) // previous config should still be valid

		// manually send instructions from the response and confirm that they modified the config
		instructions := result.RawData.([]solana.Instruction)
		_, err = solanaCommon.SendAndConfirm(ctx, s.SolanaClient, instructions, auth, rpc.CommitmentConfirmed)
		s.Require().NoError(err)

		gotConfig, err = mcmsSolana.NewInspector(s.SolanaClient).GetConfig(ctx, mcmAddress)
		s.Require().NoError(err)
		s.Require().NotNil(gotConfig)
		s.Require().Empty(cmp.Diff(skipTxConfig, *gotConfig))
	})
}
