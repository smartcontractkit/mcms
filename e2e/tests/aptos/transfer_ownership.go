//go:build e2e

package aptos

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms"
	aptosutil "github.com/smartcontractkit/mcms/e2e/utils/aptos"
	"github.com/smartcontractkit/mcms/sdk"
	aptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/types"
)

func (a *AptosTestSuite) Test_Aptos_TransferOwnership() {
	a.deployMCM()

	// Set config on contract
	signers := [2]common.Address{}
	signerKeys := [2]*ecdsa.PrivateKey{}
	for i := range signers {
		signerKeys[i], _ = crypto.GenerateKey()
		signers[i] = crypto.PubkeyToAddress(signerKeys[i].PublicKey)
	}
	slices.SortFunc(signers[:], func(a, b common.Address) int {
		return a.Cmp(b)
	})
	a.T().Logf("Signers: %v", signers)
	config := &types.Config{
		Quorum:  2,
		Signers: []common.Address{signers[0], signers[1]},
	}
	configurer := aptossdk.NewConfigurer(a.AptosRPCClient, a.deployerAccount)
	_, err := configurer.SetConfig(context.Background(), a.MCMContract.StringLong(), config, false)
	a.Require().NoError(err, "setting config on Aptos mcms contract failed")

	// Initiate ownership transfer
	payload, err := aptosutil.BuildTransactionPayload(
		a.MCMContract.StringLong()+"::mcms::transfer_ownership_to_self",
		nil,
		nil,
		nil,
	)
	a.Require().NoError(err)
	_, err = aptosutil.BuildSignSubmitAndWaitForTransaction(a.AptosRPCClient, a.deployerAccount, payload)
	a.Require().NoError(err)

	// Build proposal
	validUntil := time.Now().Add(time.Hour * 24).Unix()
	proposalBuilder := mcms.NewProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil)).
		SetDescription("Test accepting ownership of the contract itself").
		SetOverridePreviousRoot(true).
		AddChainMetadata(a.ChainSelector, types.ChainMetadata{
			StartingOpCount: 0,
			MCMAddress:      a.MCMContract.StringLong(),
		})

	// Call 1
	additionalFields := aptossdk.AdditionalFields{
		ModuleName: "mcms",
		Function:   "accept_ownership",
	}
	callOneAdditionalFields, err := json.Marshal(additionalFields)
	a.Require().NoError(err)
	proposalBuilder.AddOperation(types.Operation{
		ChainSelector: a.ChainSelector,
		Transaction: types.Transaction{
			To:               a.MCMContract.StringLong(),
			Data:             []byte{},
			AdditionalFields: callOneAdditionalFields,
		},
	})

	proposal, err := proposalBuilder.Build()
	a.Require().NoError(err, "Error building proposal")

	// Sign proposal
	inspector := aptossdk.NewInspector(a.AptosRPCClient)
	inspectors := map[types.ChainSelector]sdk.Inspector{
		a.ChainSelector: inspector,
	}
	signable, err := mcms.NewSignable(proposal, inspectors)
	a.Require().NoError(err, "Error creating signable")
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(signerKeys[0]))
	a.Require().NoError(err, "Error signing with key 0")
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(signerKeys[1]))
	a.Require().NoError(err, "Error signing with key 1")

	// Validate signatures
	quorumMet, err := signable.ValidateSignatures(context.Background())
	a.Require().NoError(err, "Error validating signatures")
	a.Require().True(quorumMet, "Quorum not met")

	// Set Root
	encoders, err := proposal.GetEncoders()
	a.Require().NoError(err)
	aptosEncoder := encoders[a.ChainSelector].(*aptossdk.Encoder)
	executors := map[types.ChainSelector]sdk.Executor{
		a.ChainSelector: aptossdk.NewExecutor(a.AptosRPCClient, a.deployerAccount, aptosEncoder),
	}
	executable, err := mcms.NewExecutable(proposal, executors)
	a.Require().NoError(err, "Error creating executable")

	txHash, err := executable.SetRoot(context.Background(), a.ChainSelector)
	a.Require().NoError(err)
	a.T().Logf("✅ SetRoot in tx: %s", txHash.Hash)

	_, err = a.AptosRPCClient.WaitForTransaction(txHash.Hash)
	a.Require().NoError(err)

	// Assert
	tree, _ := proposal.MerkleTree()
	gotHash, gotValidUntil, err := inspector.GetRoot(context.Background(), a.MCMContract.StringLong())
	a.Require().NoError(err)
	a.Require().Equal(uint32(validUntil), gotValidUntil)
	a.Require().Equal(tree.Root, gotHash)

	// Execute
	for i := range proposal.Operations {
		a.T().Logf("Executing operation: %v", i)
		var txOutput types.TransactionResult
		txOutput, err = executable.Execute(context.Background(), i)
		a.Require().NoError(err)
		a.T().Logf("✅ Executed Operation in tx: %s", txOutput.Hash)

		_, err = a.AptosRPCClient.WaitForTransaction(txOutput.Hash)
		a.Require().NoError(err)

		// Assert

		// Check that op count has increased on the mcms contract
		var opCount uint64
		opCount, err = inspector.GetOpCount(context.Background(), a.MCMContract.StringLong())
		a.Require().NoError(err)
		a.Require().EqualValues(opCount, i+1)
	}

	// Check that ownership has been transferred
	viewPayload, err := aptosutil.BuildViewPayload(a.MCMContract.StringLong()+"::mcms::owner", nil, nil, nil)
	a.Require().NoError(err)
	data, err := a.AptosRPCClient.View(viewPayload)
	a.Require().NoError(err)

	var owner aptos.AccountAddress
	err = aptosutil.DecodeAptosJsonValue(data, &owner)

	fmt.Println("MCMS contract owner:", owner.StringLong())
}
