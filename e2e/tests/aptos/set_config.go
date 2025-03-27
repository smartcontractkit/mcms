//go:build e2e

package aptos

import (
	"context"
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	aptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/types"
)

func (a *AptosTestSuite) Test_Aptos_SetConfig() {
	a.deployMCMSContract()
	mcmsAddress := a.MCMSContract.Address()

	// Signers in each group need to be sorted alphabetically
	signers := [14]common.Address{}
	for i := range signers {
		key, _ := crypto.GenerateKey()
		signers[i] = crypto.PubkeyToAddress(key.PublicKey)
	}
	slices.SortFunc(signers[:], func(a, b common.Address) int {
		return strings.Compare(strings.ToLower(a.Hex()), strings.ToLower(b.Hex()))
	})

	config := &types.Config{
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

	// Set config
	configurer := aptossdk.NewConfigurer(a.AptosRPCClient, a.deployerAccount)
	res, err := configurer.SetConfig(context.Background(), mcmsAddress.StringLong(), config, true)
	a.Require().NoError(err, "setting config on Aptos mcms contract")

	data, err := a.AptosRPCClient.WaitForTransaction(res.Hash)
	a.Require().NoError(err)
	a.Require().True(data.Success, data.VmStatus)

	// Assert that config has been set
	inspector := aptossdk.NewInspector(a.AptosRPCClient)
	gotConfig, err := inspector.GetConfig(context.Background(), mcmsAddress.StringLong())
	a.Require().NoError(err)
	a.Require().NotNil(gotConfig)
	a.Require().Equal(config, gotConfig)
}
