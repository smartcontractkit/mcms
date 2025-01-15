//go:build e2e
// +build e2e

package solanae2e

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/google/go-cmp/cmp"

	mcmsSolana "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

func (s *SolanaTestSuite) Test_Solana_SetConfig() {
	// --- arrange ---
	ctx := context.Background()
	programID, err := solana.PublicKeyFromBase58(s.SolanaChain.SolanaPrograms["mcm"])
	s.Require().NoError(err)
	auth, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)
	mcmAddress := mcmsSolana.ContractAddress(programID, testPDASeed)

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
				GroupSigners: []types.Config{},
				// FIXME: needs https://github.com/smartcontractkit/mcms/pull/216
				// GroupSigners: []types.Config{
				// 	{
				// 		Quorum: 1,
				// 		Signers: []common.Address{
				// 			testEVMAccounts[8].Address,
				// 			testEVMAccounts[9].Address,
				// 		},
				// 		GroupSigners: []types.Config{},
				// 	},
				// },
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

	// --- act ---
	configurer := mcmsSolana.NewConfigurer(s.SolanaClient, auth, s.ChainSelector)
	signature, err := configurer.SetConfig(ctx, mcmAddress, &config, true)
	s.Require().NoError(err)
	_, err = solana.SignatureFromBase58(signature)
	s.Require().NoError(err)

	// --- assert ---
	gotConfig, err := mcmsSolana.NewInspector(s.SolanaClient).GetConfig(ctx, mcmAddress)
	s.Require().NoError(err)
	s.Require().NotNil(gotConfig)
	s.Require().Empty(cmp.Diff(config, *gotConfig))
}
