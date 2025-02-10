//go:build e2e

package aptos

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"slices"
	"time"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/bcs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms"
	aptosutil "github.com/smartcontractkit/mcms/e2e/utils/aptos"
	"github.com/smartcontractkit/mcms/sdk"
	aptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/types"
)

func (a *AptosTestSuite) Test_Aptos_SetRootExecute() {
	a.deployMCM()
	a.deployMCMUser()

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
	config := &types.Config{
		Quorum:  2,
		Signers: []common.Address{signers[0], signers[1]},
	}
	configurer := aptossdk.NewConfigurer(a.AptosRPCClient, a.deployerAccount)
	_, err := configurer.SetConfig(context.Background(), a.MCMContract.StringLong(), config, false)
	a.Require().NoError(err, "setting config on Aptos mcms contract failed")

	// Arguments to call MCMUser contract with
	arg1 := "helloworld"
	arg2 := []byte{5, 4, 3, 2, 1}
	arg3 := a.deployerAccount.AccountAddress()
	arg4 := big.NewInt(42)

	// Build proposal
	validUntil := time.Now().Add(time.Hour * 24).Unix()
	proposalBuilder := mcms.NewProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil)).
		SetDescription("Test proposal with two signers on Aptos").
		SetOverridePreviousRoot(true).
		AddChainMetadata(a.ChainSelector, types.ChainMetadata{
			StartingOpCount: 0,
			MCMAddress:      a.MCMContract.StringLong(),
		})

	// Call 1
	additionalFields := aptossdk.AdditionalFields{
		ModuleName: "mcms_user",
		Function:   "function_one",
	}
	callOneParamBytes, err := bcs.SerializeSingle(func(ser *bcs.Serializer) {
		ser.WriteString(arg1)
		ser.WriteBytes(arg2)
	})
	a.Require().NoError(err)
	callOneAdditionalFields, err := json.Marshal(additionalFields)
	a.Require().NoError(err)
	proposalBuilder.AddOperation(types.Operation{
		ChainSelector: a.ChainSelector,
		Transaction: types.Transaction{
			To:               a.MCMSUserContract.StringLong(),
			Data:             callOneParamBytes,
			AdditionalFields: callOneAdditionalFields,
		},
	})

	// Call 2
	additionalFields = aptossdk.AdditionalFields{
		ModuleName: "mcms_user",
		Function:   "function_two",
	}
	callTwoParamBytes, err := bcs.SerializeSingle(func(ser *bcs.Serializer) {
		ser.FixedBytes(arg3[:])
		ser.U128(*arg4)
	})
	a.Require().NoError(err)
	callTwoAdditionalFields, err := json.Marshal(additionalFields)
	a.Require().NoError(err)
	proposalBuilder.AddOperation(types.Operation{
		ChainSelector: a.ChainSelector,
		Transaction: types.Transaction{
			To:               a.MCMSUserContract.StringLong(),
			Data:             callTwoParamBytes,
			AdditionalFields: callTwoAdditionalFields,
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

	// Check that arguments have been stored in the MCMUser contract
	data, err := a.AptosRPCClient.AccountResource(a.MCMSUserContract, a.MCMSUserContract.StringLong()+"::mcms_user::UserData")
	a.Require().NoError(err)

	userData := struct {
		Invocations int
		A           string
		B           []byte
		C           aptos.AccountAddress
		D           *big.Int
	}{}
	err = aptosutil.DecodeAptosJsonValue([]any{data["data"]}, &userData)
	a.Require().NoError(err)
	a.Require().Equal(2, userData.Invocations)
	a.Require().Equal(arg1, userData.A)
	a.Require().Equal(arg2, userData.B)
	a.Require().Equal(arg3, userData.C)
	a.Require().Equal(arg4, userData.D)
}
