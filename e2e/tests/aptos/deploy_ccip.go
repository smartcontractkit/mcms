//go:build e2e

package aptos

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"slices"
	"time"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/chainlink-internal-integrations/aptos/bindings/bind"
	"github.com/smartcontractkit/chainlink-internal-integrations/aptos/bindings/ccip"
	"github.com/smartcontractkit/chainlink-internal-integrations/aptos/bindings/compile"
	module_mcms "github.com/smartcontractkit/chainlink-internal-integrations/aptos/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-internal-integrations/aptos/bindings/mcms_large_packages"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	aptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/types"
)

func (a *AptosTestSuite) Test_Aptos_DeployCCIP() {
	// a.T().Skip()
	a.deployMCM()
	opts := &bind.TransactOpts{Signer: a.deployerAccount}

	inspector := aptossdk.NewInspector(a.AptosRPCClient)
	inspectors := map[types.ChainSelector]sdk.Inspector{
		a.ChainSelector: inspector,
	}

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
	result, err := configurer.SetConfig(context.Background(), a.MCMContract.Address.StringLong(), config, false)
	a.Require().NoError(err, "setting config on Aptos mcms contract failed")

	_, err = a.AptosRPCClient.WaitForTransaction(result.Hash)
	a.Require().NoError(err)
	a.T().Logf("✅ SetConfig in tx: %s", result.Hash)

	// Initiate ownership transfer
	tx, err := a.MCMContract.MCMS.TransferOwnershipToSelf(opts)
	a.Require().NoError(err)
	_, err = a.AptosRPCClient.WaitForTransaction(tx.Hash)
	a.Require().NoError(err)
	a.T().Logf("🚀 TransferOwnershipToSelf in tx: %s", tx.Hash)

	// Need to store the address of the large packages contract
	var preregisteredObjectAddress aptos.AccountAddress

	// Build first proposal
	{
		validUntil := time.Now().Add(time.Hour * 24).Unix()
		proposalBuilder := mcms.NewProposalBuilder().
			SetVersion("v1").
			SetValidUntil(uint32(validUntil)).
			SetDescription("Test accepting ownership of the contract itself").
			SetOverridePreviousRoot(true).
			AddChainMetadata(a.ChainSelector, types.ChainMetadata{
				StartingOpCount: 0,
				MCMAddress:      a.MCMContract.Address.StringLong(),
			})

		// Call 1 - accept ownership
		module, function, _, args, err := a.MCMContract.MCMS.EncodeAcceptOwnership()
		a.Require().NoError(err)
		additionalFields := aptossdk.AdditionalFields{
			ModuleName: module.Name,
			Function:   function,
		}
		callOneAdditionalFields, err := json.Marshal(additionalFields)
		a.Require().NoError(err)
		proposalBuilder.AddOperation(types.Operation{
			ChainSelector: a.ChainSelector,
			Transaction: types.Transaction{
				To:               a.MCMContract.Address.StringLong(),
				Data:             module_mcms.ArgsToData(args),
				AdditionalFields: callOneAdditionalFields,
			},
		})

		// Call 2 - deploy mcms_large_packages to preregistered object
		var compiledPackage compile.CompiledPackage
		compiledPackage, preregisteredObjectAddress, err = mcms_large_packages.Compile(ccip.DefaultSeed, a.MCMContract.Address)
		a.Require().NoError(err)
		a.T().Logf("Deploying user contract to: %v", preregisteredObjectAddress.StringLong())
		module, function, _, args, err = a.MCMContract.MCMS.EncodeOwnedObjectPublish([]byte(ccip.DefaultSeed), compiledPackage.Metadata, compiledPackage.Bytecode)
		a.Require().NoError(err)
		additionalFields = aptossdk.AdditionalFields{
			ModuleName: module.Name,
			Function:   function,
		}
		deployAdditionalFields, err := json.Marshal(additionalFields)
		proposalBuilder.AddOperation(types.Operation{
			ChainSelector: a.ChainSelector,
			Transaction: types.Transaction{
				To:               a.MCMContract.Address.StringLong(),
				Data:             module_mcms.ArgsToData(args),
				AdditionalFields: deployAdditionalFields,
			},
		})

		proposal, err := proposalBuilder.Build()
		a.Require().NoError(err, "Error building proposal")

		// Sign proposal
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

		data, err := a.AptosRPCClient.WaitForTransaction(txHash.Hash)
		a.Require().NoError(err)
		a.Require().True(data.Success, "SetRoot failed")

		// Assert
		tree, _ := proposal.MerkleTree()
		gotHash, gotValidUntil, err := inspector.GetRoot(context.Background(), a.MCMContract.Address.StringLong())
		a.Require().NoError(err)
		a.Require().Equal(uint32(validUntil), gotValidUntil)
		a.Require().Equal(tree.Root, gotHash)

		// Execute
		for i := range proposal.Operations {
			a.T().Logf("Executing operation: %v...", i)
			var txOutput types.TransactionResult
			txOutput, err = executable.Execute(context.Background(), i)
			a.Require().NoError(err)
			a.T().Logf("✅ Executed Operation in tx: %s", txOutput.Hash)

			data, err = a.AptosRPCClient.WaitForTransaction(txOutput.Hash)
			a.Require().NoError(err)
			a.Require().True(data.Success, "Execution failed")

			// Assert

			// Check that op count has increased on the mcms contract
			var opCount uint64
			opCount, err = inspector.GetOpCount(context.Background(), a.MCMContract.Address.StringLong())
			a.Require().NoError(err)
			a.Require().EqualValues(opCount, i+1)
		}

		// Check that ownership has been transferred
		owner, err := a.MCMContract.MCMS.Owner(nil)
		a.Require().NoError(err)
		a.Require().Equal(a.MCMContract.Address.StringLong(), owner.StringLong())

		a.T().Logf("MCMS contract is owned by itself: %v", owner.StringLong())
	}

	// Second proposal - deploy CCIP

	{
		startingOpCount, err := inspector.GetOpCount(context.Background(), a.MCMContract.Address.StringLong())
		a.Require().NoError(err)
		validUntil := time.Now().Add(time.Hour * 24).Unix()
		proposalBuilder := mcms.NewProposalBuilder().
			SetVersion("v1").
			SetValidUntil(uint32(validUntil)).
			SetDescription("Test deploying CCIP via MCMS").
			SetOverridePreviousRoot(true).
			AddChainMetadata(a.ChainSelector, types.ChainMetadata{
				StartingOpCount: startingOpCount,
				MCMAddress:      a.MCMContract.Address.StringLong(),
			})

		// Compile CCIP

		ccipPayload, err := ccip.Compile(ccip.DefaultSeed, a.MCMContract.Address)
		a.Require().NoError(err)

		// Create chunks
		chunks, err := bind.CreateChunks(ccipPayload, bind.CHUNK_SIZE_IN_BYTES)
		a.Require().NoError(err)

		// Stage chunks with mcms_large_packages and execute with the last one
		mcmsLargePackagesContract := mcms_large_packages.Bind(preregisteredObjectAddress, nil)
		for i, chunk := range chunks {
			if i == len(chunks)-1 {
				// Last chunk stages the remaining data and executes
				module, function, _, args, err := mcmsLargePackagesContract.LargePackages.EncodeStageCodeChunkAndPublishToAccount(chunk.Metadata, chunk.CodeIndices, chunk.Chunks)
				a.Require().NoError(err)
				additionalFields := aptossdk.AdditionalFields{
					ModuleName: module.Name,
					Function:   function,
				}
				afBytes, err := json.Marshal(additionalFields)
				a.Require().NoError(err)
				proposalBuilder.AddOperation(types.Operation{
					ChainSelector: a.ChainSelector,
					Transaction: types.Transaction{
						To:               preregisteredObjectAddress.StringLong(),
						Data:             module_mcms.ArgsToData(args),
						AdditionalFields: afBytes,
					},
				})
				break
			}
			module, function, _, args, err := mcmsLargePackagesContract.LargePackages.EncodeStageCodeChunk(chunk.Metadata, chunk.CodeIndices, chunk.Chunks)
			a.Require().NoError(err)
			additionalFields := aptossdk.AdditionalFields{
				ModuleName: module.Name,
				Function:   function,
			}
			afBytes, err := json.Marshal(additionalFields)
			a.Require().NoError(err)
			proposalBuilder.AddOperation(types.Operation{
				ChainSelector: a.ChainSelector,
				Transaction: types.Transaction{
					To:               preregisteredObjectAddress.StringLong(),
					Data:             module_mcms.ArgsToData(args),
					AdditionalFields: afBytes,
				},
			})
		}

		proposal, err := proposalBuilder.Build()
		a.Require().NoError(err, "Error building proposal")

		// Sign proposal
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

		data, err := a.AptosRPCClient.WaitForTransaction(txHash.Hash)
		a.Require().NoError(err)
		a.Require().True(data.Success, "SetRoot failed")

		// Execute
		for i := range proposal.Operations {
			a.T().Logf("Executing operation: %v...", i)
			var txOutput types.TransactionResult
			txOutput, err = executable.Execute(context.Background(), i)
			a.Require().NoError(err)
			a.T().Logf("✅ Executed Operation in tx: %s", txOutput.Hash)

			data, err := a.AptosRPCClient.WaitForTransaction(txOutput.Hash)
			a.Require().NoError(err)
			a.Require().True(data.Success, "Execution failed")

			// Assert

			// Check that op count has increased on the mcms contract
			var opCount uint64
			opCount, err = inspector.GetOpCount(context.Background(), a.MCMContract.Address.StringLong())
			a.Require().NoError(err)
			a.Require().EqualValues(opCount, startingOpCount+uint64(i)+1)
		}
	}
}
