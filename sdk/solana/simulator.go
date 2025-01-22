package solana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/smartcontractkit/mcms/types"
)

type Simulator struct {
	client    *rpc.Client
	encoder   *Encoder
	inspector *Inspector
	auth      solana.PrivateKey
}

// NewSimulator creates a new Solana Simulator
func NewSimulator(client *rpc.Client, auth solana.PrivateKey, encoder *Encoder) (*Simulator, error) {
	if client == nil {
		return nil, errors.New("Simulator was created without a Solana RPC client")
	}

	if encoder == nil {
		return nil, errors.New("Simulator was created without an encoder")
	}

	return &Simulator{
		client:    client,
		encoder:   encoder,
		auth:      auth,
		inspector: NewInspector(client),
	}, nil
}

func (s *Simulator) SimulateSetRoot(
	ctx context.Context,
	originCaller solana.PublicKey,
	metadata types.ChainMetadata,
	proof [][]byte,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
) error {
	return fmt.Errorf("not implemented")
}

func (s *Simulator) SimulateOperation(
	ctx context.Context,
	// We don't need the metadata for simulating on solana, since the RPC client already know what chain we are on.
	_ types.ChainMetadata,
	op types.Operation,
) error {
	// Parse the inner instruction from the operation
	var additionalFields AdditionalFields
	if err := json.Unmarshal(op.Transaction.AdditionalFields, &additionalFields); err != nil {
		return fmt.Errorf("unable to unmarshal additional fields: %w", err)
	}

	toProgramID, _, err := ParseContractAddress(op.Transaction.To)
	if errors.Is(err, ErrInvalidContractAddressFormat) {
		var pkerr error
		toProgramID, pkerr = solana.PublicKeyFromBase58(op.Transaction.To)
		if pkerr != nil {
			return fmt.Errorf("unable to parse the 'To' address: %w", err)
		}
	}

	// Build the instruction
	innerInstruction := solana.NewInstruction(
		toProgramID,
		additionalFields.Accounts,
		op.Transaction.Data,
	)
	recentBlockHash, err := s.client.GetRecentBlockhash(ctx, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("unable to get recent blockhash: %w", err)
	}

	// Build the transaction with the inner instruction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{innerInstruction},
		recentBlockHash.Value.Blockhash,
		solana.TransactionPayer(s.auth.PublicKey()),
	)
	if err != nil {
		return fmt.Errorf("unable to create transaction: %w", err)
	}
	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			if s.auth.PublicKey().Equals(key) {
				return &s.auth
			}

			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("unable to sign transaction: %w", err)
	}

	// Simulate the transaction
	_, err = s.client.SimulateTransaction(ctx, tx)
	if err != nil {
		return fmt.Errorf("unable to simulate transaction: %w", err)
	}

	return nil
}
