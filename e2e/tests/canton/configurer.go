//go:build e2e

package canton

import (
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/noders-team/go-daml/pkg/model"

	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"

	"github.com/smartcontractkit/mcms/types"
)

type MCMSConfigurerTestSuite struct {
	TestSuite
}

// SetupSuite runs before the test suite
func (s *MCMSConfigurerTestSuite) SetupSuite() {
	s.TestSuite.SetupSuite()
	s.DeployMCMSContract()
}

func (s *MCMSConfigurerTestSuite) TestSetConfig() {
	// Signers in each group need to be sorted alphabetically
	signers := [30]common.Address{}
	for i := range signers {
		key, _ := crypto.GenerateKey()
		signers[i] = crypto.PubkeyToAddress(key.PublicKey)
	}
	slices.SortFunc(signers[:], func(a, b common.Address) int {
		return a.Cmp(b)
	})

	proposerConfig := &types.Config{
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
	{
		configurer, err := cantonsdk.NewConfigurer(s.client, s.participant.UserName, s.participant.Party)
		s.Require().NoError(err, "creating configurer for Canton mcms contract")
		tx, err := configurer.SetConfig(s.T().Context(), s.mcmsContractID, proposerConfig, true)
		s.Require().NoError(err, "setting config on Canton mcms contract")

		// Verify transaction result
		rawData, ok := tx.RawData.(map[string]any)
		s.Require().True(ok)
		rawTx, ok := rawData["RawTx"]
		s.Require().True(ok)

		submitResp, ok := rawTx.(*model.SubmitAndWaitForTransactionResponse)
		s.Require().True(ok)

		// Verify CompletionOffset exists
		s.Require().NotZero(submitResp.CompletionOffset, "transaction should have CompletionOffset")

		events := submitResp.Transaction.Events
		s.Require().Len(events, 2, "transaction should have exactly 2 events (archived + created)")

		// Verify event[0] is Archived (old contract)
		s.Require().NotNil(events[0].Archived, "first event should be Archived event")
		s.Require().Nil(events[0].Created, "first event should not be Created event")
		s.Require().Equal(s.mcmsContractID, events[0].Archived.ContractID, "archived contract should be the old MCMS contract")

		// Verify event[1] is Created (new contract)
		s.Require().NotNil(events[1].Created, "second event should be Created event")
		s.Require().Nil(events[1].Archived, "second event should not be Archived event")

		// Verify Template ID matches
		rawData, ok = tx.RawData.(map[string]any)
		s.Require().True(ok)
		newMCMSTemplateID, ok := rawData["NewMCMSTemplateID"].(string)
		s.Require().True(ok)
		s.Require().Contains(newMCMSTemplateID, "MCMS.Main:MCMS", "template ID should match MCMS template")
		s.Require().Equal(newMCMSTemplateID, events[1].Created.TemplateID, "created event template ID should match returned template ID")

		// Verify new contract ID is different from old
		newMCMSContractID, ok := rawData["NewMCMSContractID"].(string)
		s.Require().True(ok)
		s.Require().NotEmpty(newMCMSContractID, "new contract ID should not be empty")
		s.Require().NotEqual(s.mcmsContractID, newMCMSContractID, "new contract ID should be different from old contract ID")
		s.Require().Equal(newMCMSContractID, events[1].Created.ContractID, "created event contract ID should match returned contract ID")
	}
}
