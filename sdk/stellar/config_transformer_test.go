package stellar

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	mcmsbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"

	"github.com/smartcontractkit/mcms/types"
)

func TestConfigTransformer_RoundTrip(t *testing.T) {
	t.Parallel()

	signer1 := common.HexToAddress("0x1234")
	signer2 := common.HexToAddress("0xabcd")

	cfg, err := types.NewConfig(
		1,
		[]common.Address{signer1},
		[]types.Config{
			{
				Quorum:       1,
				Signers:      []common.Address{signer2},
				GroupSigners: []types.Config{},
			},
		},
	)
	require.NoError(t, err)

	transformer := NewConfigTransformer()
	chainConfig, err := transformer.ToChainConfig(cfg, nil)
	require.NoError(t, err)

	require.Len(t, chainConfig.Signers, 2)
	require.Equal(t, uint8(1), chainConfig.GroupQuorums[0])
	require.Equal(t, uint8(1), chainConfig.GroupQuorums[1])
	require.Equal(t, uint8(0), chainConfig.GroupParents[1])

	// Convert back to SDK Config
	sdkConfig, err := transformer.ToConfig(chainConfig)
	require.NoError(t, err)

	require.True(t, cfg.Equals(sdkConfig))
}

func TestConfigTransformer_Errors(t *testing.T) {
	t.Parallel()

	transformer := NewConfigTransformer()

	// Test ToConfig nil config error
	_, err := transformer.ToConfig(nil)
	require.Error(t, err)

	// Test ToConfig no signers error
	var quorums [32]byte
	quorums[0] = 1
	_, err = transformer.ToConfig(&mcmsbindings.Config{GroupQuorums: quorums})
	require.ErrorContains(t, err, "no signers")

	// Test ToConfig unpadded EVM address error
	var malformed [32]byte
	malformed[0] = 1
	_, err = transformer.ToConfig(&mcmsbindings.Config{
		GroupQuorums: quorums,
		Signers:      []mcmsbindings.Signer{{Addr: malformed, Group: 0}},
	})
	require.ErrorContains(t, err, "not a padded EVM address")
}
