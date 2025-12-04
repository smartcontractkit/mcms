//go:build e2e

package solanae2e

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"

	"github.com/smartcontractkit/mcms"
	e2eutils "github.com/smartcontractkit/mcms/e2e/utils/solana"
	"github.com/smartcontractkit/mcms/sdk"
	mcmsSolana "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"
)

func (s *TestSuite) TestChunkLargeInstructions() {
	s.T().Setenv("MCMS_SOLANA_MAX_RETRIES", "20")

	mcmPDASeed := [32]byte([]byte("hEjRE08jHA2ilqk12fgjE9OIjRJRd7m8"[:]))
	timelockPDASeed := [32]byte([]byte("BG7wilBWT4mc6p9yFnmfcu3yX7r9dazl")[:])

	ctx := s.T().Context()
	s.SetupMCM(mcmPDASeed)
	s.SetupTimelock(timelockPDASeed, 1*time.Second)

	mcmSignerPDA, err := mcmsSolana.FindSignerPDA(s.MCMProgramID, mcmPDASeed)
	s.Require().NoError(err)
	timelockSignerPDA, err := mcmsSolana.FindTimelockSignerPDA(s.TimelockProgramID, timelockPDASeed)
	s.Require().NoError(err)

	// Fund the signer PDA account
	auth, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	accounts := []solana.PublicKey{auth.PublicKey(), mcmSignerPDA, timelockSignerPDA}
	e2eutils.FundAccounts(s.T(), accounts, 1, s.SolanaClient)

	s.AssignRoleToAccounts(ctx, timelockPDASeed, auth, []solana.PublicKey{mcmSignerPDA}, timelock.Proposer_Role)
	s.AssignRoleToAccounts(ctx, timelockPDASeed, auth, []solana.PublicKey{mcmSignerPDA}, timelock.Executor_Role)

	marshaledTimelockProposal := largeTimelockProposal(uint64(s.ChainSelector), mcmPDASeed, timelockPDASeed, s.Roles)
	var tp mcms.TimelockProposal
	err = json.Unmarshal([]byte(marshaledTimelockProposal), &tp)
	s.Require().NoError(err)

	timelockProposal, err := mcms.NewTimelockProposal(strings.NewReader(marshaledTimelockProposal))
	s.Require().NoError(err)

	converters := map[types.ChainSelector]sdk.TimelockConverter{s.ChainSelector: &mcmsSolana.TimelockConverter{}}
	proposal, _, err := timelockProposal.Convert(ctx, converters)
	s.Require().NoError(err)

	// Get required programs and accounts
	mcmID := mcmsSolana.ContractAddress(s.MCMProgramID, mcmPDASeed)
	signerEVMAccount := NewEVMTestAccount(s.T())

	// Build a simple 1 signer config
	mcmConfig := types.Config{Quorum: 1, Signers: []common.Address{signerEVMAccount.Address}}

	// build encoders, executors and inspectors.
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.ChainSelector].(*mcmsSolana.Encoder)
	executor := mcmsSolana.NewExecutor(encoder, s.SolanaClient, auth)
	executors := map[types.ChainSelector]sdk.Executor{s.ChainSelector: executor}
	inspectors := map[types.ChainSelector]sdk.Inspector{s.ChainSelector: mcmsSolana.NewInspector(s.SolanaClient)}

	// sign proposal
	signable, err := mcms.NewSignable(&proposal, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(signerEVMAccount.PrivateKey))
	s.Require().NoError(err)

	// set config
	configurer := mcmsSolana.NewConfigurer(s.SolanaClient, auth, s.ChainSelector)
	_, err = configurer.SetConfig(ctx, mcmID, &mcmConfig, true)
	s.Require().NoError(err)

	// call SetRoot
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err)
	signature, err := executable.SetRoot(ctx, s.ChainSelector)
	s.Require().NoError(err)
	_, err = solana.SignatureFromBase58(signature.Hash)
	s.Require().NoError(err)

	// --- act + assert; 3rd and 4th instructions are "AppendData" ---
	s.Require().Len(proposal.Operations, 6)
	for i := range proposal.Operations {
		signature, err = executable.Execute(ctx, i)
		s.Require().NoError(err)
		_, err = solana.SignatureFromBase58(signature.Hash)
		s.Require().NoError(err)
		s.T().Logf("executed operation %d", i)
	}
}

var largeTimelockProposal = func(
	cSelector uint64, mcmSeed, timelockSeed mcmsSolana.PDASeed, roles RoleMap,
) string {
	proposerAC := roles[timelock.Proposer_Role].AccessController.PublicKey()
	cancellerAC := roles[timelock.Canceller_Role].AccessController.PublicKey()
	bypasserAC := roles[timelock.Bypasser_Role].AccessController.PublicKey()

	return fmt.Sprintf(`
		{
			"version": "v1",
			"kind": "TimelockProposal",
			"validUntil": 1999999999,
			"signatures": [],
			"overridePreviousRoot": false,
			"chainMetadata": {
				"%d": {
					"startingOpCount": 0,
					"mcmAddress": "5vNJx78mz7KVMjhuipyr9jKBKcMrKYGdjGkgE4LUmjKk.%s",
					"additionalFields": {
						"proposerRoleAccessController": "%s",
						"cancellerRoleAccessController": "%s",
						"bypasserRoleAccessController": "%s"
					}
				}
			},
			"description": "Test proposal with instruction that needs multiple timelock.AppendData instructions",
			"action": "schedule",
			"delay": "3h0m0s",
			"timelockAddresses": {
				"%d": "DoajfR5tK24xVw51fWcawUZWhAXD8yrBJVacc13neVQA.%s"
			},
			"operations": [
				{
					"chainSelector": %d,
					"transactions": [
						{
							"contractType": "Router",
							"tags": [],
							"to": "Ccip842gzYHhvdDkSyi2YVCoAWPbYJoApMFzSxQroE9C",
							"data": "13pRFr462w0Ve5z8lJmERQAPAAAA0E0+oWp6CPttB4RpTZBctwQnJ1KyV5j0H0vqEFybnaeTUE9IPI/pT4a3L34iUQDZ/orve0s/HPta3A3est3uQ95HdOJTMoiy4IoYyjLiJ56z1wG7g2l9uHSDNa+vA+rpgGIhtd1qVjvNwaOeIFObuLnaWEnWdREzy0TVcC/tHxR8p6U89t8BxC3QMNJRzkUmYv7q6JYo2a2/qDIOEDDhhNvAcgdRiviVb3J27RuY8IQ8fxXDS7ZmRGcFFjExl9JkxuoPIhYKyCDOfNmDWHrNB1V8fZx5zBXdgY9Q0sVU6EjSbL2kJfLLcQRbLZBt4JFIo1mx3ammCz5YaQUa6/1DqegNnSlzLByEYl4be1qGz/C/e3ALbG3pz5WxsD7uUa7Fm5SxS9D4Iapiz95LfqvFAcXKaxozdMm7sbUm3CL7zI/GxbJp2i2xTWXp+ujpcPffnWdvfq+qYuJl5yFy2MLnzAfMwYZK2XIawRWcwCNTLDFYo50dxUEJlPZXbS4lQeMHwPrVUPhdyecxNH9kOlVjPKeJcJJjw1siwbBpIlQ5lWtJ4v/hVUPu/jKRvSl8+yy4rf7UZjiM01DTiPwgSG0mhLeFrpRV1xYchifLtsZQ9aYlpa1lRbIe44dmY2pPzIRLAQ==",
							"additionalFields": {
								"accounts": [
									{
										"PublicKey": "CowGG7G1FsfedN4jx66Gw7co1xEsiKQfE1f8BXSnSK9w",
										"IsWritable": true,
										"IsSigner": false
									},
									{
										"PublicKey": "3Yrg9E4ySAeRezgQY99NNarAmFLtixapga9MZb6y2dt3",
										"IsWritable": false,
										"IsSigner": false
									},
									{
										"PublicKey": "GoFoFfEDgALWRTw5dSY3VZQSwbFpvBEhDv1sFAWzYpbf",
										"IsWritable": true,
										"IsSigner": false
									},
									{
										"PublicKey": "11111111111111111111111111111111",
										"IsWritable": false,
										"IsSigner": false
									}
								],
								"value": 0
							}
						}
					]
				}
			]
		}`, cSelector, string(mcmSeed[:]), proposerAC, cancellerAC, bypasserAC, cSelector, string(timelockSeed[:]), cSelector)
}
