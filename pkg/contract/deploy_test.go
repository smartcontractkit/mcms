package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fakeAddr = "0x1234567890abcdef"
)

func fakeContractDeployment() DeployFunc[string] {
	return func() (string, error) {
		return fakeAddr, nil
	}
}

func Test_Deploy(t *testing.T) {
	addr, err := Deploy(fakeContractDeployment())
	require.NoError(t, err)

	assert.Equal(t, fakeAddr, addr)
}
