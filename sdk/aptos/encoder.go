package aptos

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/types"
)

var (
	mcmDomainSeparatorOp       = crypto.Keccak256([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP_APTOS"))
	mcmDomainSeparatorMetadata = crypto.Keccak256([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA_APTOS"))
)

var _ sdk.Encoder = &Encoder{}

type Encoder struct {
	ChainSelector        types.ChainSelector
	TxCount              uint64
	OverridePreviousRoot bool
}

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

func (e *Encoder) HashOperation(opCount uint32, metadata types.ChainMetadata, op types.Operation) (common.Hash, error) {
	chainID, err := chain_selectors.AptosChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return common.Hash{}, &sdkerrors.InvalidChainIDError{
			ReceivedChainID: e.ChainSelector,
		}
	}
	// TODO Remove this once we've added chainID 4 to chain-selectors
	chainID = 4
	multisigAddress := aptos.AccountAddress{}
	if err := multisigAddress.ParseStringRelaxed(metadata.MCMAddress); err != nil {
		return common.Hash{}, fmt.Errorf("unable to parse Aptos contract address: %w", err)
	}
	toAddress := aptos.AccountAddress{}
	if err := toAddress.ParseStringRelaxed(op.Transaction.To); err != nil {
		return common.Hash{}, fmt.Errorf("unable to parse To address: %w", err)
	}
	additionalFields := AdditionalFields{}
	if err := json.Unmarshal(op.Transaction.AdditionalFields, &additionalFields); err != nil {
		return common.Hash{}, fmt.Errorf("unable to unmarshal additionalFields : %w", err)
	}

	var preImage []byte
	preImage = append(preImage, mcmDomainSeparatorOp...)
	preImage = append(preImage, common.LeftPadBytes(binary.BigEndian.AppendUint64(nil, chainID), 32)...)
	preImage = append(preImage, multisigAddress[:]...)
	preImage = append(preImage, common.LeftPadBytes(binary.BigEndian.AppendUint32(nil, opCount), 32)...)
	preImage = append(preImage, toAddress[:]...)
	preImage = append(preImage, common.LeftPadBytes([]byte(additionalFields.ModuleName), 64)...)
	preImage = append(preImage, common.LeftPadBytes([]byte(additionalFields.Function), 64)...)
	preImage = append(preImage, []byte(append(op.Transaction.Data, bytes.Repeat([]byte{0}, 32-len(op.Transaction.Data)%32)...))...) // Right pad to 32-byte increment

	return crypto.Keccak256Hash(preImage), nil
}

func (e *Encoder) HashMetadata(metadata types.ChainMetadata) (common.Hash, error) {
	chainID, err := chain_selectors.AptosChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return common.Hash{}, &sdkerrors.InvalidChainIDError{
			ReceivedChainID: e.ChainSelector,
		}
	}
	// TODO Remove this once we've added chainID 4 to chain-selectors
	chainID = 4
	multisigAddress := aptos.AccountAddress{}
	if err := multisigAddress.ParseStringRelaxed(metadata.MCMAddress); err != nil {
		return common.Hash{}, fmt.Errorf("unable to parse Aptos contract address: %w", err)
	}

	var preImage []byte
	preImage = append(preImage, mcmDomainSeparatorMetadata...)
	preImage = append(preImage, common.LeftPadBytes(binary.BigEndian.AppendUint64(nil, chainID), 32)...)
	preImage = append(preImage, multisigAddress[:]...)
	preImage = append(preImage, common.LeftPadBytes(binary.BigEndian.AppendUint64(nil, metadata.StartingOpCount), 32)...)
	preImage = append(preImage, common.LeftPadBytes(binary.BigEndian.AppendUint64(nil, metadata.StartingOpCount+e.TxCount), 32)...)
	if e.OverridePreviousRoot {
		preImage = append(preImage, common.LeftPadBytes([]byte{1}, 32)...)
	} else {
		preImage = append(preImage, common.LeftPadBytes([]byte{0}, 32)...)
	}

	return crypto.Keccak256Hash(preImage), nil
}
