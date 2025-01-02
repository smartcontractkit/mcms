//go:build e2e
// +build e2e

package e2e_solana

import (
	"github.com/ethereum/go-ethereum/common"
	solana "github.com/gagliardetto/solana-go"
	"github.com/google/go-cmp/cmp"

	mcmsSolana "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

func (s *SolanaTestSuite) Test_Solana_SetConfig() {
	// --- arrange ---
	mcmAddress := s.SolanaChain.SolanaPrograms["mcm"] // FIXME: replace with runtime deployment?
	auth, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

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
				// FIXME: the following does not work
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
	configurer := mcmsSolana.NewConfigurer(s.SolanaClient, auth, s.chainSelector)
	signature, err := configurer.SetConfig(mcmAddress, &config, true)
	s.Require().NoError(err)
    _, err = solana.SignatureFromBase58(signature)
	s.Require().NoError(err)

	// --- assert ---
	gotConfig, err := mcmsSolana.NewInspector(s.SolanaClient).GetConfig(mcmAddress)
	s.Require().NoError(err)
	s.Require().NotNil(gotConfig)
	s.Require().Empty(cmp.Diff(config, *gotConfig))
}
