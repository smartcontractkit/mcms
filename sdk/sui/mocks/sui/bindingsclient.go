// Code generated manually for BindingsClient gRPC migration. DO NOT EDIT.

package mock_sui

import (
	"context"
	"math/big"

	suirpcv2 "github.com/block-vision/sui-go-sdk/pb/sui/rpc/v2"
	suigosigner "github.com/block-vision/sui-go-sdk/signer"
	"github.com/block-vision/sui-go-sdk/transaction"
	mock "github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/chainlink-sui/relayer/client"
)

// BindingsClient is a mock type for client.BindingsClient.
type BindingsClient struct {
	mock.Mock
}

type BindingsClient_Expecter struct {
	mock *mock.Mock
}

func (_m *BindingsClient) EXPECT() *BindingsClient_Expecter {
	return &BindingsClient_Expecter{mock: &_m.Mock}
}

func (_m *BindingsClient) ReadObjectId(ctx context.Context, objectId string) (*suirpcv2.Object, error) {
	ret := _m.Called(ctx, objectId)
	if len(ret) == 0 {
		panic("no return value specified for ReadObjectId")
	}
	var r0 *suirpcv2.Object
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*suirpcv2.Object, error)); ok {
		return rf(ctx, objectId)
	}
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*suirpcv2.Object)
	}
	r1 = ret.Error(1)
	return r0, r1
}

func (_m *BindingsClient) QueryCoinsByAddress(ctx context.Context, address string, coinType string) ([]*suirpcv2.Object, error) {
	ret := _m.Called(ctx, address, coinType)
	if len(ret) == 0 {
		panic("no return value specified for QueryCoinsByAddress")
	}
	var r0 []*suirpcv2.Object
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) ([]*suirpcv2.Object, error)); ok {
		return rf(ctx, address, coinType)
	}
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*suirpcv2.Object)
	}
	r1 = ret.Error(1)
	return r0, r1
}

func (_m *BindingsClient) GetReferenceGasPrice(ctx context.Context) (*big.Int, error) {
	ret := _m.Called(ctx)
	if len(ret) == 0 {
		panic("no return value specified for GetReferenceGasPrice")
	}
	var r0 *big.Int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (*big.Int, error)); ok {
		return rf(ctx)
	}
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*big.Int)
	}
	r1 = ret.Error(1)
	return r0, r1
}

func (_m *BindingsClient) SimulatePTB(ctx context.Context, bcsBytes []byte) ([]any, error) {
	ret := _m.Called(ctx, bcsBytes)
	if len(ret) == 0 {
		panic("no return value specified for SimulatePTB")
	}
	var r0 []any
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte) ([]any, error)); ok {
		return rf(ctx, bcsBytes)
	}
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]any)
	}
	r1 = ret.Error(1)
	return r0, r1
}

func (_m *BindingsClient) SendTransaction(ctx context.Context, req *suirpcv2.ExecuteTransactionRequest) (*suirpcv2.ExecuteTransactionResponse, error) {
	ret := _m.Called(ctx, req)
	if len(ret) == 0 {
		panic("no return value specified for SendTransaction")
	}
	var r0 *suirpcv2.ExecuteTransactionResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *suirpcv2.ExecuteTransactionRequest) (*suirpcv2.ExecuteTransactionResponse, error)); ok {
		return rf(ctx, req)
	}
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*suirpcv2.ExecuteTransactionResponse)
	}
	r1 = ret.Error(1)
	return r0, r1
}

func (_m *BindingsClient) GetTransactionStatus(ctx context.Context, digest string) (client.TransactionResult, error) {
	ret := _m.Called(ctx, digest)
	if len(ret) == 0 {
		panic("no return value specified for GetTransactionStatus")
	}
	var r0 client.TransactionResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (client.TransactionResult, error)); ok {
		return rf(ctx, digest)
	}
	r0 = ret.Get(0).(client.TransactionResult)
	r1 = ret.Error(1)
	return r0, r1
}

func (_m *BindingsClient) FinishPTBAndSend(ctx context.Context, txnSigner *suigosigner.Signer, tx *transaction.Transaction, requestType client.TransactionRequestType) (*suirpcv2.ExecuteTransactionResponse, error) {
	ret := _m.Called(ctx, txnSigner, tx, requestType)
	if len(ret) == 0 {
		panic("no return value specified for FinishPTBAndSend")
	}
	var r0 *suirpcv2.ExecuteTransactionResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *suigosigner.Signer, *transaction.Transaction, client.TransactionRequestType) (*suirpcv2.ExecuteTransactionResponse, error)); ok {
		return rf(ctx, txnSigner, tx, requestType)
	}
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*suirpcv2.ExecuteTransactionResponse)
	}
	r1 = ret.Error(1)
	return r0, r1
}

// NewBindingsClient creates a new instance of BindingsClient.
func NewBindingsClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *BindingsClient {
	mockClient := &BindingsClient{}
	mockClient.Mock.Test(t)
	t.Cleanup(func() { mockClient.AssertExpectations(t) })
	return mockClient
}
