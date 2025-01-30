package aptos

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"
	"time"

	"github.com/aptos-labs/aptos-go-sdk/bcs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	aptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/types"
)

func (a *AptosTestSuite) Test_Aptos_SetRoot() {

	a.deployMCM()
	a.deployMCMUser()

	fmt.Println("MCMS Contract: ", a.MCMContract.StringLong())
	fmt.Println("MCMSUser Contract: ", a.MCMSUserContract.StringLong())

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
		ser.WriteString("helloworld")
		ser.WriteBytes([]byte{5, 4, 3, 2, 1})
	})
	a.Require().NoError(err)
	callOneAdditionalFields, _ := json.Marshal(additionalFields)
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
		ser.FixedBytes(a.deployerAccount.Address[:])
		ser.U128(*big.NewInt(42))
	})
	a.Require().NoError(err)
	callTwoAdditionalFields, _ := json.Marshal(additionalFields)
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

	// Assert
	tree, _ := proposal.MerkleTree()
	gotHash, gotValidUntil, err := inspector.GetRoot(context.Background(), a.MCMContract.StringLong())
	a.Require().NoError(err)
	a.Require().Equal(uint32(validUntil), gotValidUntil)
	a.Require().Equal(tree.Root, gotHash)
}
