// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	sdk "github.com/smartcontractkit/mcms/sdk"
	mock "github.com/stretchr/testify/mock"

	types "github.com/smartcontractkit/mcms/types"
)

// Decoder is an autogenerated mock type for the Decoder type
type Decoder struct {
	mock.Mock
}

type Decoder_Expecter struct {
	mock *mock.Mock
}

func (_m *Decoder) EXPECT() *Decoder_Expecter {
	return &Decoder_Expecter{mock: &_m.Mock}
}

// Decode provides a mock function with given fields: op, contractInterfaces
func (_m *Decoder) Decode(op types.Transaction, contractInterfaces string) (sdk.DecodedOperation, error) {
	ret := _m.Called(op, contractInterfaces)

	if len(ret) == 0 {
		panic("no return value specified for Decode")
	}

	var r0 sdk.DecodedOperation
	var r1 error
	if rf, ok := ret.Get(0).(func(types.Transaction, string) (sdk.DecodedOperation, error)); ok {
		return rf(op, contractInterfaces)
	}
	if rf, ok := ret.Get(0).(func(types.Transaction, string) sdk.DecodedOperation); ok {
		r0 = rf(op, contractInterfaces)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(sdk.DecodedOperation)
		}
	}

	if rf, ok := ret.Get(1).(func(types.Transaction, string) error); ok {
		r1 = rf(op, contractInterfaces)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Decoder_Decode_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Decode'
type Decoder_Decode_Call struct {
	*mock.Call
}

// Decode is a helper method to define mock.On call
//   - op types.Transaction
//   - contractInterfaces string
func (_e *Decoder_Expecter) Decode(op interface{}, contractInterfaces interface{}) *Decoder_Decode_Call {
	return &Decoder_Decode_Call{Call: _e.mock.On("Decode", op, contractInterfaces)}
}

func (_c *Decoder_Decode_Call) Run(run func(op types.Transaction, contractInterfaces string)) *Decoder_Decode_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.Transaction), args[1].(string))
	})
	return _c
}

func (_c *Decoder_Decode_Call) Return(_a0 sdk.DecodedOperation, _a1 error) *Decoder_Decode_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Decoder_Decode_Call) RunAndReturn(run func(types.Transaction, string) (sdk.DecodedOperation, error)) *Decoder_Decode_Call {
	_c.Call.Return(run)
	return _c
}

// NewDecoder creates a new instance of Decoder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDecoder(t interface {
	mock.TestingT
	Cleanup(func())
}) *Decoder {
	mock := &Decoder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
