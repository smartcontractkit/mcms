//go:build e2e

package aptos

import (
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	aptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/types"
)

func (a *TestSuite) Test_Aptos_SetConfig() {
	/*
		This tests setting and retrieving the config for all three timelock roles:
		- Bypasser
		- Canceller
		- Proposer
	*/
	a.deployMCMSContract()
	mcmsAddress := a.MCMSContract.Address()

	// Signers in each group need to be sorted alphabetically
	signers := [30]common.Address{}
	for i := range signers {
		key, _ := crypto.GenerateKey()
		signers[i] = crypto.PubkeyToAddress(key.PublicKey)
	}
	slices.SortFunc(signers[:], func(a, b common.Address) int {
		return strings.Compare(strings.ToLower(a.Hex()), strings.ToLower(b.Hex()))
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
		configurer := aptossdk.NewConfigurer(a.AptosRPCClient, a.deployerAccount, aptossdk.TimelockRoleBypasser)
		res, err := configurer.SetConfig(a.T().Context(), mcmsAddress.StringLong(), bypasserConfig, true)
		a.Require().NoError(err, "setting config on Aptos mcms contract")

		data, err := a.AptosRPCClient.WaitForTransaction(res.Hash)
		a.Require().NoError(err)
		a.Require().True(data.Success, data.VmStatus)
	}
	{
		configurer := aptossdk.NewConfigurer(a.AptosRPCClient, a.deployerAccount, aptossdk.TimelockRoleCanceller)
		res, err := configurer.SetConfig(a.T().Context(), mcmsAddress.StringLong(), cancellerConfig, true)
		a.Require().NoError(err, "setting config on Aptos mcms contract")

		data, err := a.AptosRPCClient.WaitForTransaction(res.Hash)
		a.Require().NoError(err)
		a.Require().True(data.Success, data.VmStatus)
	}
	{
		configurer := aptossdk.NewConfigurer(a.AptosRPCClient, a.deployerAccount, aptossdk.TimelockRoleProposer)
		res, err := configurer.SetConfig(a.T().Context(), mcmsAddress.StringLong(), proposerConfig, true)
		a.Require().NoError(err, "setting config on Aptos mcms contract")

		data, err := a.AptosRPCClient.WaitForTransaction(res.Hash)
		a.Require().NoError(err)
		a.Require().True(data.Success, data.VmStatus)
	}

	// Assert that config has been set
	{
		inspector := aptossdk.NewInspector(a.AptosRPCClient, aptossdk.TimelockRoleBypasser)
		gotConfig, err := inspector.GetConfig(a.T().Context(), mcmsAddress.StringLong())
		a.Require().NoError(err)
		a.Require().NotNil(gotConfig)
		a.Require().Equal(bypasserConfig, gotConfig)
	}
	{
		inspector := aptossdk.NewInspector(a.AptosRPCClient, aptossdk.TimelockRoleCanceller)
		gotConfig, err := inspector.GetConfig(a.T().Context(), mcmsAddress.StringLong())
		a.Require().NoError(err)
		a.Require().NotNil(gotConfig)
		a.Require().Equal(cancellerConfig, gotConfig)
	}
	{
		inspector := aptossdk.NewInspector(a.AptosRPCClient, aptossdk.TimelockRoleProposer)
		gotConfig, err := inspector.GetConfig(a.T().Context(), mcmsAddress.StringLong())
		a.Require().NoError(err)
		a.Require().NotNil(gotConfig)
		a.Require().Equal(proposerConfig, gotConfig)
	}
}
