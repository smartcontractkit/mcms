package sui

import (
	"context"
	"fmt"
	"math/big"

	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	moduleMcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"

	"github.com/block-vision/sui-go-sdk/sui"
	bindutils "github.com/smartcontractkit/chainlink-sui/bindings/utils"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Configurer = &Configurer{}

type Configurer struct {
	client        sui.ISuiAPI
	signer        bindutils.SuiSigner
	role          TimelockRole
	mcms          moduleMcms.IMcms
	ownerCap      string
	chainSelector uint64
}

func NewConfigurer(client sui.ISuiAPI, signer bindutils.SuiSigner, role TimelockRole, mcmsPackageId string, ownerCap string, chainSelector uint64) (*Configurer, error) {
	mcms, err := moduleMcms.NewMcms(mcmsPackageId, client)
	if err != nil {
		return nil, err
	}

	return &Configurer{
		client:        client,
		signer:        signer,
		role:          role,
		mcms:          mcms,
		ownerCap:      ownerCap,
		chainSelector: chainSelector,
	}, nil
}

func (c Configurer) SetConfig(ctx context.Context, mcmsAddr string, cfg *types.Config, clearRoot bool) (types.TransactionResult, error) {
	chainID, err := cselectors.SuiChainIdFromSelector(c.chainSelector)
	if err != nil {
		return types.TransactionResult{}, err
	}
	chainIDBig := new(big.Int).SetUint64(chainID)
	groupQuorum, groupParents, signerAddresses, signerGroups, err := evm.ExtractSetConfigInputs(cfg)

	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to extract set config inputs: %w", err)
	}
	signers := make([][]byte, len(signerAddresses))
	for i, addr := range signerAddresses {
		signers[i] = addr.Bytes()
	}
	opts := bind.CallOpts{
		Signer:           c.signer,
		WaitForExecution: true,
	}
	tx, err := c.mcms.SetConfig(
		ctx,
		&opts,
		bind.Object{Id: c.ownerCap},
		bind.Object{Id: mcmsAddr},
		c.role.Byte(),
		chainIDBig,
		signers,
		signerGroups,
		groupQuorum[:],
		groupParents[:],
		clearRoot,
	)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to set config: %w", err)
	}

	return types.TransactionResult{
		Hash:        tx.Digest,
		ChainFamily: cselectors.FamilySui,
		RawData:     tx,
	}, nil
}
