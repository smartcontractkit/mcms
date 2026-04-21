package sui

import (
	"context"
	"fmt"
	"math/big"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/block-vision/sui-go-sdk/sui"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	modulemcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
	bindutils "github.com/smartcontractkit/chainlink-sui/bindings/utils"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Configurer = &Configurer{}

type Configurer struct {
	client        sui.ISuiAPI
	signer        bindutils.SuiSigner
	role          TimelockRole
	mcms          modulemcms.IMcms
	ownerCap      string
	chainSelector uint64
}

func NewConfigurer(client sui.ISuiAPI, signer bindutils.SuiSigner, role TimelockRole, mcmsPackageID string, ownerCap string, chainSelector uint64) (*Configurer, error) {
	mcms, err := modulemcms.NewMcms(mcmsPackageID, client)
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
	chainID, err := chainsel.SuiChainIdFromSelector(c.chainSelector)
	if err != nil {
		return types.TransactionResult{}, err
	}
	chainIDBig := new(big.Int).SetUint64(chainID)
	groupQuorum, groupParents, signerAddresses, signerGroups, err := sdk.ExtractSetConfigInputs(cfg)

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
		ChainFamily: chainsel.FamilySui,
		RawData:     tx,
	}, nil
}
