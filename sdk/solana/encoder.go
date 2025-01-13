package solana

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Encoder = (*Encoder)(nil)

// Encoder is a struct that encodes ChainOperations and ChainMetadata into the format expected
// by the Solana ManyChainMultiSig contract.
type Encoder struct {
	ChainSelector        types.ChainSelector
	TxCount              uint64
	OverridePreviousRoot bool
}

// NewEncoder returns a new Encoder.
func NewEncoder(
	chainSelector types.ChainSelector,
	txCount uint64,
	overridePreviousRoot bool,
) *Encoder {
	return &Encoder{
		ChainSelector:        chainSelector,
		TxCount:              txCount,
		OverridePreviousRoot: overridePreviousRoot,
	}
}

// HashOperation converts the MCMS Operation into the format expected by the Solana
// ManyChainMultiSig contract, and hashes it.
func (e *Encoder) HashOperation(
	opCount uint32,
	metadata types.ChainMetadata,
	op types.Operation,
) (common.Hash, error) {
	hashBytes := crypto.Keccak256Hash([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP_SOLANA"))

	programID, pdaSeed, err := ParseContractAddress(metadata.MCMAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("unable to parse solana contract address: %w", err)
	}

	configPDA, err := FindConfigPDA(programID, pdaSeed)
	if err != nil {
		return common.Hash{}, err
	}

	toProgramID, _, err := ParseContractAddress(op.Transaction.To)
	if errors.Is(err, ErrInvalidContractAddressFormat) {
		var pkerr error
		toProgramID, pkerr = solana.PublicKeyFromBase58(op.Transaction.To)
		if pkerr != nil {
			return common.Hash{}, fmt.Errorf("unable to get hash from base58 To address: %w", err)
		}
	}
	// Parse Additional fields to get the ix accounts
	var additionalFields AdditionalFields
	if op.Transaction.AdditionalFields != nil {
		if err := json.Unmarshal(op.Transaction.AdditionalFields, &additionalFields); err != nil {
			return common.Hash{}, fmt.Errorf("unable to unmarshal additional fields: %w", err)
		}
	}

	buffers := [][]byte{
		hashBytes[:],
		numToU64LePaddedEncoding(uint64(e.ChainSelector)),
		configPDA.Bytes(),
		numToU64LePaddedEncoding(uint64(opCount)),
		toProgramID.Bytes(),
		numToU64LePaddedEncoding(uint64(len(op.Transaction.Data))),
		op.Transaction.Data,
		numToU64LePaddedEncoding(uint64(len(additionalFields.Accounts))),
	}
	for _, account := range additionalFields.Accounts {
		buffers = append(buffers, serializeAccountMeta(&account))
	}

	return calculateHash(buffers), nil
}

// HashMetadata converts the MCMS ChainMetadata into the format expected by the Solana
// ManyChainMultiSig contract, and hashes it.
func (e *Encoder) HashMetadata(metadata types.ChainMetadata) (common.Hash, error) {
	hashBytes := crypto.Keccak256Hash([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA_SOLANA"))

	programID, pdaSeed, err := ParseContractAddress(metadata.MCMAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("unable to parse solana contract address: %w", err)
	}

	configPDA, err := FindConfigPDA(programID, pdaSeed)
	if err != nil {
		return common.Hash{}, err
	}

	buffers := [][]byte{
		hashBytes[:],
		numToU64LePaddedEncoding(uint64(e.ChainSelector)),
		configPDA[:],
		numToU64LePaddedEncoding(metadata.StartingOpCount),
		numToU64LePaddedEncoding(metadata.StartingOpCount + e.TxCount),
		boolToPaddedEncoding(e.OverridePreviousRoot),
	}

	return calculateHash(buffers), nil
}

func calculateHash(buffers [][]byte) [32]byte {
	hash := crypto.Keccak256Hash(bytes.Join(buffers, nil))
	return common.BytesToHash(hash[:])
}

func numToU64LePaddedEncoding(n uint64) []byte {
	const numBytes = 32
	const offset = 24
	b := make([]byte, numBytes)
	binary.LittleEndian.PutUint64(b[offset:], n)

	return b
}

func boolToPaddedEncoding(b bool) []byte {
	const numBytes = 32
	result := make([]byte, numBytes)
	if b {
		result[numBytes-1] = 1
	}

	return result
}

func serializeAccountMeta(a *solana.AccountMeta) []byte {
	var flags byte
	if a.IsSigner {
		flags |= 0b10
	}
	if a.IsWritable {
		flags |= 0b01
	}
	result := append(a.PublicKey.Bytes(), flags)

	return result
}
