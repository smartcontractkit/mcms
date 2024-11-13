// Code generated by mockery v2.46.3. DO NOT EDIT.

package mocks

import (
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

// Decode provides a mock function with given fields: op, abi
func (_m *Decoder) Decode(op types.Operation, abi string) (string, string, error) {
	ret := _m.Called(op, abi)

	if len(ret) == 0 {
		panic("no return value specified for Decode")
	}

	var r0 string
	var r1 string
	var r2 error
	if rf, ok := ret.Get(0).(func(types.Operation, string) (string, string, error)); ok {
		return rf(op, abi)
	}
	if rf, ok := ret.Get(0).(func(types.Operation, string) string); ok {
		r0 = rf(op, abi)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(types.Operation, string) string); ok {
		r1 = rf(op, abi)
	} else {
		r1 = ret.Get(1).(string)
	}

	if rf, ok := ret.Get(2).(func(types.Operation, string) error); ok {
		r2 = rf(op, abi)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// Decoder_Decode_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Decode'
type Decoder_Decode_Call struct {
	*mock.Call
}

// Decode is a helper method to define mock.On call
//   - op types.Operation
//   - abi string
func (_e *Decoder_Expecter) Decode(op interface{}, abi interface{}) *Decoder_Decode_Call {
	return &Decoder_Decode_Call{Call: _e.mock.On("Decode", op, abi)}
}

func (_c *Decoder_Decode_Call) Run(run func(op types.Operation, abi string)) *Decoder_Decode_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.Operation), args[1].(string))
	})
	return _c
}

func (_c *Decoder_Decode_Call) Return(methodName string, args string, err error) *Decoder_Decode_Call {
	_c.Call.Return(methodName, args, err)
	return _c
}

func (_c *Decoder_Decode_Call) RunAndReturn(run func(types.Operation, string) (string, string, error)) *Decoder_Decode_Call {
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
