package solana

import (
	"fmt"

	"github.com/gagliardetto/solana-go"
	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/types"
)

var _ types.ContractID = (*SolanaContractID)(nil)

type SolanaContractID struct {
	ProgramID  solana.PublicKey
	InstanceID [32]byte
}

func NewSolanaContractID(programID solana.PublicKey, instanceID [32]byte) *SolanaContractID {
	return &SolanaContractID{
		ProgramID:  programID,
		InstanceID: instanceID,
	}
}

func (i *SolanaContractID) String() string {
	return fmt.Sprintf("[program_id=%s; instance_id=%s]", i.ProgramID, i.InstanceID)
}

func (i *SolanaContractID) ChainFamily() string {
	return cselectors.FamilySolana
}

func FromContractID(cid types.ContractID) (*SolanaContractID, error) {
	if cid.ChainFamily() != cselectors.FamilySolana {
		return nil, fmt.Errorf("invalid contract id: %w", types.ErrUnsupportedChainFamily)
	}

	return cid.(*SolanaContractID), nil
}
