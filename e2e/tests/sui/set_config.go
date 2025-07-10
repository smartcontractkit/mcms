//go:build e2e

package sui

import (
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	suisdk "github.com/smartcontractkit/mcms/sdk/sui"

	"github.com/smartcontractkit/mcms/types"
)

const Sui_test_chain_selector = 10

func (s *SuiTestSuite) Test_Sui_SetConfig() {

	// Signers in each group need to be sorted alphabetically
	signers := [30]common.Address{}
	for i := range signers {
		key, _ := crypto.GenerateKey()
		signers[i] = crypto.PubkeyToAddress(key.PublicKey)
	}
	slices.SortFunc(signers[:], func(a, b common.Address) int {
		return strings.Compare(strings.ToLower(s.Hex()), strings.ToLower(b.Hex()))
	})

	bypasserConfig := &types.Config{
		Quorum: 2,
		Signers: []common.Address{
			signers[0],
			signers[1],
			signers[2],
		},
		GroupSigners: []types.Config{
			{
				Quorum: 4,
				Signers: []common.Address{
					signers[3],
					signers[4],
					signers[5],
					signers[6],
					signers[7],
				},
				GroupSigners: []types.Config{
					{
						Quorum: 1,
						Signers: []common.Address{
							signers[8],
							signers[9],
						},
						GroupSigners: []types.Config{},
					},
				},
			},
			{
				Quorum: 3,
				Signers: []common.Address{
					signers[10],
					signers[11],
					signers[12],
					signers[13],
				},
				GroupSigners: []types.Config{},
			},
		},
	}

	cancellerConfig := &types.Config{
		Quorum: 1,
		Signers: []common.Address{
			signers[14],
			signers[15],
		},
		GroupSigners: []types.Config{
			{
				Quorum: 2,
				Signers: []common.Address{
					signers[16],
					signers[17],
					signers[18],
					signers[19],
				},
				GroupSigners: []types.Config{},
			},
		},
	}

	proposerConfig := &types.Config{
		Quorum:  2,
		Signers: []common.Address{},
		GroupSigners: []types.Config{
			{
				Quorum: 2,
				Signers: []common.Address{
					signers[20],
					signers[21],
					signers[22],
					signers[23],
				},
				GroupSigners: []types.Config{},
			}, {
				Quorum: 2,
				Signers: []common.Address{
					signers[24],
					signers[25],
					signers[26],
					signers[27],
				},
				GroupSigners: []types.Config{},
			}, {
				Quorum: 1,
				Signers: []common.Address{
					signers[28],
					signers[29],
				},
				GroupSigners: []types.Config{},
			},
		},
	}

	// Set config
	{
		configurer, err := suisdk.NewConfigurer(s.client, s.signer, suisdk.TimelockRoleBypasser, s.mcms.Address(), s.mcms.ownerCapObj, Sui_test_chain_selector)
		s.Require().NoError(err, "creating configurer for Sui mcms contract")
		_, err = configurer.SetConfig(s.T().Context(), s.mcms.mcmsObject, bypasserConfig, true)
		s.Require().NoError(err, "setting config on Sui mcms contract")
	}
	{
		configurer, err := suisdk.NewConfigurer(s.client, s.signer, suisdk.TimelockRoleCanceller, s.mcms.Address(), s.mcms.ownerCapObj, Sui_test_chain_selector)
		s.Require().NoError(err, "creating configurer for Sui mcms contract")
		_, err = configurer.SetConfig(s.T().Context(), s.mcms.mcmsObject, cancellerConfig, true)
		s.Require().NoError(err, "setting config on Sui mcms contract")
	}
	{
		configurer, err := suisdk.NewConfigurer(s.client, s.signer, suisdk.TimelockRoleProposer, s.mcms.Address(), s.mcms.ownerCapObj, Sui_test_chain_selector)
		s.Require().NoError(err, "creating configurer for Sui mcms contract")
		_, err = configurer.SetConfig(s.T().Context(), s.mcms.mcmsObject, proposerConfig, true)
		s.Require().NoError(err, "setting config on Sui mcms contract")

	}

	// Assert that config has been set
	{
		inspector, err := suisdk.NewInspector(s.client, s.signer, s.mcms.Address(), suisdk.TimelockRoleBypasser)
		s.Require().NoError(err, "creating inspector for Sui mcms contract")

		gotConfig, err := inspector.GetConfig(s.T().Context(), s.mcms.mcmsObject)
		s.Require().NoError(err)
		s.Require().NotNil(gotConfig)
		s.Require().Equal(bypasserConfig, gotConfig)
	}
	{
		inspector, err := suisdk.NewInspector(s.client, s.signer, s.mcms.Address(), suisdk.TimelockRoleCanceller)
		s.Require().NoError(err, "creating inspector for Sui mcms contract")

		gotConfig, err := inspector.GetConfig(s.T().Context(), s.mcms.mcmsObject)
		s.Require().NoError(err)
		s.Require().NotNil(gotConfig)
		s.Require().Equal(cancellerConfig, gotConfig)
	}
	{
		inspector, err := suisdk.NewInspector(s.client, s.signer, s.mcms.Address(), suisdk.TimelockRoleProposer)
		s.Require().NoError(err, "creating inspector for Sui mcms contract")

		gotConfig, err := inspector.GetConfig(s.T().Context(), s.mcms.mcmsObject)
		s.Require().NoError(err)
		s.Require().NotNil(gotConfig)
		s.Require().Equal(proposerConfig, gotConfig)
	}

}
