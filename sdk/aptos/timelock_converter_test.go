package aptos

import (
	"testing"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/stretchr/testify/require"
)

func TestHashOperationBatch(t *testing.T) {
	t.Parallel()
	// 0x860ab27255ad63f4b1cd56ffeb41953ba4b23d4d1f21e8e821ae7c1d4b0c8001

	targets := []aptos.AccountAddress{aptos.AccountOne}
	moduleNames := []string{"module"}
	functionNames := []string{"function"}
	datas := [][]byte{[]byte("asdf")}
	predecessor := []byte{1, 2, 3, 4}
	salt := []byte{5, 6, 7, 8}

	hash, err := HashOperationBatch(targets, moduleNames, functionNames, datas, predecessor, salt)
	require.NoError(t, err)
	require.Equal(t, "0x860ab27255ad63f4b1cd56ffeb41953ba4b23d4d1f21e8e821ae7c1d4b0c8001", hash.Hex())
}
