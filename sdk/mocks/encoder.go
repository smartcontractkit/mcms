// Code generated by mockery v2.46.3. DO NOT EDIT.

package mocks

import (
	common "github.com/ethereum/go-ethereum/common"
	mock "github.com/stretchr/testify/mock"

	types "github.com/smartcontractkit/mcms/types"
)

// Encoder is an autogenerated mock type for the Encoder type
type Encoder struct {
	mock.Mock
}

type Encoder_Expecter struct {
	mock *mock.Mock
}

func (_m *Encoder) EXPECT() *Encoder_Expecter {
	return &Encoder_Expecter{mock: &_m.Mock}
}

// HashMetadata provides a mock function with given fields: metadata
func (_m *Encoder) HashMetadata(metadata types.ChainMetadata) (common.Hash, error) {
	ret := _m.Called(metadata)

	if len(ret) == 0 {
		panic("no return value specified for HashMetadata")
	}

	var r0 common.Hash
	var r1 error
	if rf, ok := ret.Get(0).(func(types.ChainMetadata) (common.Hash, error)); ok {
		return rf(metadata)
	}
	if rf, ok := ret.Get(0).(func(types.ChainMetadata) common.Hash); ok {
		r0 = rf(metadata)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Hash)
		}
	}

	if rf, ok := ret.Get(1).(func(types.ChainMetadata) error); ok {
		r1 = rf(metadata)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Encoder_HashMetadata_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'HashMetadata'
type Encoder_HashMetadata_Call struct {
	*mock.Call
}

// HashMetadata is a helper method to define mock.On call
//   - metadata types.ChainMetadata
func (_e *Encoder_Expecter) HashMetadata(metadata interface{}) *Encoder_HashMetadata_Call {
	return &Encoder_HashMetadata_Call{Call: _e.mock.On("HashMetadata", metadata)}
}

func (_c *Encoder_HashMetadata_Call) Run(run func(metadata types.ChainMetadata)) *Encoder_HashMetadata_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.ChainMetadata))
	})
	return _c
}

func (_c *Encoder_HashMetadata_Call) Return(_a0 common.Hash, _a1 error) *Encoder_HashMetadata_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Encoder_HashMetadata_Call) RunAndReturn(run func(types.ChainMetadata) (common.Hash, error)) *Encoder_HashMetadata_Call {
	_c.Call.Return(run)
	return _c
}

// HashOperation provides a mock function with given fields: opCount, metadata, op
func (_m *Encoder) HashOperation(opCount uint32, metadata types.ChainMetadata, op types.ChainOperation) (common.Hash, error) {
	ret := _m.Called(opCount, metadata, op)

	if len(ret) == 0 {
		panic("no return value specified for HashOperation")
	}

	var r0 common.Hash
	var r1 error
	if rf, ok := ret.Get(0).(func(uint32, types.ChainMetadata, types.ChainOperation) (common.Hash, error)); ok {
		return rf(opCount, metadata, op)
	}
	if rf, ok := ret.Get(0).(func(uint32, types.ChainMetadata, types.ChainOperation) common.Hash); ok {
		r0 = rf(opCount, metadata, op)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Hash)
		}
	}

	if rf, ok := ret.Get(1).(func(uint32, types.ChainMetadata, types.ChainOperation) error); ok {
		r1 = rf(opCount, metadata, op)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Encoder_HashOperation_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'HashOperation'
type Encoder_HashOperation_Call struct {
	*mock.Call
}

// HashOperation is a helper method to define mock.On call
//   - opCount uint32
//   - metadata types.ChainMetadata
//   - op types.ChainOperation
func (_e *Encoder_Expecter) HashOperation(opCount interface{}, metadata interface{}, op interface{}) *Encoder_HashOperation_Call {
	return &Encoder_HashOperation_Call{Call: _e.mock.On("HashOperation", opCount, metadata, op)}
}

func (_c *Encoder_HashOperation_Call) Run(run func(opCount uint32, metadata types.ChainMetadata, op types.ChainOperation)) *Encoder_HashOperation_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint32), args[1].(types.ChainMetadata), args[2].(types.ChainOperation))
	})
	return _c
}

func (_c *Encoder_HashOperation_Call) Return(_a0 common.Hash, _a1 error) *Encoder_HashOperation_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Encoder_HashOperation_Call) RunAndReturn(run func(uint32, types.ChainMetadata, types.ChainOperation) (common.Hash, error)) *Encoder_HashOperation_Call {
	_c.Call.Return(run)
	return _c
}

// NewEncoder creates a new instance of Encoder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewEncoder(t interface {
	mock.TestingT
	Cleanup(func())
}) *Encoder {
	mock := &Encoder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
