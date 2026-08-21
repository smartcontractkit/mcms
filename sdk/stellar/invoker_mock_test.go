package stellar

import (
	"context"
	"sync"

	protocolrpc "github.com/stellar/go-stellar-sdk/protocols/rpc"
	"github.com/stellar/go-stellar-sdk/xdr"
)

type callRecord struct {
	ContractID   string
	FunctionName string
	Args         []xdr.ScVal
}

type mockInvoker struct {
	mu          sync.Mutex
	calls       []callRecord
	stubReturns map[string]*xdr.ScVal
	stubErrors  map[string]error
}

func newMockInvoker() *mockInvoker {
	return &mockInvoker{
		stubReturns: make(map[string]*xdr.ScVal),
		stubErrors:  make(map[string]error),
	}
}

func (m *mockInvoker) stub(fnName string, val *xdr.ScVal, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if val != nil {
		m.stubReturns[fnName] = val
	}
	if err != nil {
		m.stubErrors[fnName] = err
	}
}

func (m *mockInvoker) InvokeContract(_ context.Context, contractID, functionName string, args []xdr.ScVal) (*xdr.ScVal, error) {
	m.mu.Lock()
	m.calls = append(m.calls, callRecord{ContractID: contractID, FunctionName: functionName, Args: args})
	stubRet := m.stubReturns[functionName]
	stubErr := m.stubErrors[functionName]
	m.mu.Unlock()

	return stubRet, stubErr
}

func (m *mockInvoker) SimulateContract(_ context.Context, contractID, functionName string, args []xdr.ScVal) (*xdr.ScVal, error) {
	m.mu.Lock()
	m.calls = append(m.calls, callRecord{ContractID: contractID, FunctionName: functionName, Args: args})
	stubRet := m.stubReturns[functionName]
	stubErr := m.stubErrors[functionName]
	m.mu.Unlock()

	return stubRet, stubErr
}

func (m *mockInvoker) GetEvents(_ context.Context, _ string, _ uint32, _ []string) ([]protocolrpc.EventInfo, error) {
	return nil, nil
}
