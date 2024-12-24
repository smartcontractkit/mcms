package solana

import (
	"context"
	"fmt"
	"testing"

	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigurer(t *testing.T) {
	type args struct {
		client        *rpc.Client
		auth          solana.PrivateKey
		chainSelector types.ChainSelector
	}
	tests := []struct {
		name string
		args args
		want *Configurer
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		assert.Equalf(t, tt.want, NewConfigurer(tt.args.client, tt.args.auth, tt.args.chainSelector), "%q. NewConfigurer()", tt.name)
	}
}

func TestConfigurer_SetConfig(t *testing.T) {
	type fields struct {
		chainSelector types.ChainSelector
		client        *rpc.Client
		auth          solana.PrivateKey
	}
	type args struct {
		mcmAddressHex string
		cfg           *types.Config
		clearRoot     bool
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      string
		assertion assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		c := &Configurer{
			chainSelector: tt.fields.chainSelector,
			client:        tt.fields.client,
			auth:          tt.fields.auth,
		}
		got, err := c.SetConfig(tt.args.mcmAddressHex, tt.args.cfg, tt.args.clearRoot)
		tt.assertion(t, err, fmt.Sprintf("%q. Configurer.SetConfig()", tt.name))
		assert.Equalf(t, tt.want, got, "%q. Configurer.SetConfig()", tt.name)
	}
}

func TestConfigurer_preloadSigners(t *testing.T) {
	type fields struct {
		chainSelector types.ChainSelector
		client        *rpc.Client
		auth          solana.PrivateKey
	}
	type args struct {
		ctx              context.Context
		mcmName          [32]byte
		signerAddresses  [][20]uint8
		configPDA        solana.PublicKey
		configSignersPDA solana.PublicKey
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		assertion assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		c := &Configurer{
			chainSelector: tt.fields.chainSelector,
			client:        tt.fields.client,
			auth:          tt.fields.auth,
		}
		tt.assertion(t, c.preloadSigners(tt.args.ctx, tt.args.mcmName, tt.args.signerAddresses, tt.args.configPDA, tt.args.configSignersPDA), fmt.Sprintf("%q. Configurer.preloadSigners()", tt.name))
	}
}

func Test_solanaSignerAddresses(t *testing.T) {
	type args struct {
		evmAddresses []evmCommon.Address
	}
	tests := []struct {
		name string
		args args
		want [][20]uint8
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		assert.Equalf(t, tt.want, solanaSignerAddresses(tt.args.evmAddresses), "%q. solanaSignerAddresses()", tt.name)
	}
}
