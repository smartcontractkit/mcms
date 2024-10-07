package contract

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fakeAddr = "0x1234567890abcdef"
	fakeTx   = `{"data":"0x00001"}`
)

func fakeContractDeployment() DeployFunc[string, json.RawMessage] {
	return func() (string, json.RawMessage, error) {
		return fakeAddr, json.RawMessage(fakeTx), nil
	}
}

func Test_Deploy(t *testing.T) {
	addr, tx, err := Deploy(fakeContractDeployment())
	require.NoError(t, err)

	assert.Equal(t, fakeAddr, addr)
	assert.JSONEq(t, fakeTx, string(tx))
}
