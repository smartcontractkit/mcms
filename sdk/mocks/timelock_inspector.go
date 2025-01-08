// Code generated by mockery v2.48.0. DO NOT EDIT.

package mocks

import (
	context "context"

	common "github.com/ethereum/go-ethereum/common"

	mock "github.com/stretchr/testify/mock"
)

// TimelockInspector is an autogenerated mock type for the TimelockInspector type
type TimelockInspector struct {
	mock.Mock
}

type TimelockInspector_Expecter struct {
	mock *mock.Mock
}

func (_m *TimelockInspector) EXPECT() *TimelockInspector_Expecter {
	return &TimelockInspector_Expecter{mock: &_m.Mock}
}

// GetBypassers provides a mock function with given fields: ctx, address
func (_m *TimelockInspector) GetBypassers(ctx context.Context, address string) ([]common.Address, error) {
	ret := _m.Called(ctx, address)

	if len(ret) == 0 {
		panic("no return value specified for GetBypassers")
	}

	var r0 []common.Address
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]common.Address, error)); ok {
		return rf(ctx, address)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []common.Address); ok {
		r0 = rf(ctx, address)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]common.Address)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, address)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TimelockInspector_GetBypassers_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetBypassers'
type TimelockInspector_GetBypassers_Call struct {
	*mock.Call
}

// GetBypassers is a helper method to define mock.On call
//   - ctx context.Context
//   - address string
func (_e *TimelockInspector_Expecter) GetBypassers(ctx interface{}, address interface{}) *TimelockInspector_GetBypassers_Call {
	return &TimelockInspector_GetBypassers_Call{Call: _e.mock.On("GetBypassers", ctx, address)}
}

func (_c *TimelockInspector_GetBypassers_Call) Run(run func(ctx context.Context, address string)) *TimelockInspector_GetBypassers_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *TimelockInspector_GetBypassers_Call) Return(_a0 []common.Address, _a1 error) *TimelockInspector_GetBypassers_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *TimelockInspector_GetBypassers_Call) RunAndReturn(run func(context.Context, string) ([]common.Address, error)) *TimelockInspector_GetBypassers_Call {
	_c.Call.Return(run)
	return _c
}

// GetCancellers provides a mock function with given fields: ctx, address
func (_m *TimelockInspector) GetCancellers(ctx context.Context, address string) ([]common.Address, error) {
	ret := _m.Called(ctx, address)

	if len(ret) == 0 {
		panic("no return value specified for GetCancellers")
	}

	var r0 []common.Address
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]common.Address, error)); ok {
		return rf(ctx, address)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []common.Address); ok {
		r0 = rf(ctx, address)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]common.Address)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, address)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TimelockInspector_GetCancellers_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetCancellers'
type TimelockInspector_GetCancellers_Call struct {
	*mock.Call
}

// GetCancellers is a helper method to define mock.On call
//   - ctx context.Context
//   - address string
func (_e *TimelockInspector_Expecter) GetCancellers(ctx interface{}, address interface{}) *TimelockInspector_GetCancellers_Call {
	return &TimelockInspector_GetCancellers_Call{Call: _e.mock.On("GetCancellers", ctx, address)}
}

func (_c *TimelockInspector_GetCancellers_Call) Run(run func(ctx context.Context, address string)) *TimelockInspector_GetCancellers_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *TimelockInspector_GetCancellers_Call) Return(_a0 []common.Address, _a1 error) *TimelockInspector_GetCancellers_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *TimelockInspector_GetCancellers_Call) RunAndReturn(run func(context.Context, string) ([]common.Address, error)) *TimelockInspector_GetCancellers_Call {
	_c.Call.Return(run)
	return _c
}

// GetExecutors provides a mock function with given fields: ctx, address
func (_m *TimelockInspector) GetExecutors(ctx context.Context, address string) ([]common.Address, error) {
	ret := _m.Called(ctx, address)

	if len(ret) == 0 {
		panic("no return value specified for GetExecutors")
	}

	var r0 []common.Address
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]common.Address, error)); ok {
		return rf(ctx, address)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []common.Address); ok {
		r0 = rf(ctx, address)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]common.Address)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, address)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TimelockInspector_GetExecutors_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetExecutors'
type TimelockInspector_GetExecutors_Call struct {
	*mock.Call
}

// GetExecutors is a helper method to define mock.On call
//   - ctx context.Context
//   - address string
func (_e *TimelockInspector_Expecter) GetExecutors(ctx interface{}, address interface{}) *TimelockInspector_GetExecutors_Call {
	return &TimelockInspector_GetExecutors_Call{Call: _e.mock.On("GetExecutors", ctx, address)}
}

func (_c *TimelockInspector_GetExecutors_Call) Run(run func(ctx context.Context, address string)) *TimelockInspector_GetExecutors_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *TimelockInspector_GetExecutors_Call) Return(_a0 []common.Address, _a1 error) *TimelockInspector_GetExecutors_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *TimelockInspector_GetExecutors_Call) RunAndReturn(run func(context.Context, string) ([]common.Address, error)) *TimelockInspector_GetExecutors_Call {
	_c.Call.Return(run)
	return _c
}

// GetProposers provides a mock function with given fields: ctx, address
func (_m *TimelockInspector) GetProposers(ctx context.Context, address string) ([]common.Address, error) {
	ret := _m.Called(ctx, address)

	if len(ret) == 0 {
		panic("no return value specified for GetProposers")
	}

	var r0 []common.Address
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]common.Address, error)); ok {
		return rf(ctx, address)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []common.Address); ok {
		r0 = rf(ctx, address)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]common.Address)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, address)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TimelockInspector_GetProposers_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetProposers'
type TimelockInspector_GetProposers_Call struct {
	*mock.Call
}

// GetProposers is a helper method to define mock.On call
//   - ctx context.Context
//   - address string
func (_e *TimelockInspector_Expecter) GetProposers(ctx interface{}, address interface{}) *TimelockInspector_GetProposers_Call {
	return &TimelockInspector_GetProposers_Call{Call: _e.mock.On("GetProposers", ctx, address)}
}

func (_c *TimelockInspector_GetProposers_Call) Run(run func(ctx context.Context, address string)) *TimelockInspector_GetProposers_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *TimelockInspector_GetProposers_Call) Return(_a0 []common.Address, _a1 error) *TimelockInspector_GetProposers_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *TimelockInspector_GetProposers_Call) RunAndReturn(run func(context.Context, string) ([]common.Address, error)) *TimelockInspector_GetProposers_Call {
	_c.Call.Return(run)
	return _c
}

// IsOperation provides a mock function with given fields: ctx, address, opID
func (_m *TimelockInspector) IsOperation(ctx context.Context, address string, opID [32]byte) (bool, error) {
	ret := _m.Called(ctx, address, opID)

	if len(ret) == 0 {
		panic("no return value specified for IsOperation")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, [32]byte) (bool, error)); ok {
		return rf(ctx, address, opID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, [32]byte) bool); ok {
		r0 = rf(ctx, address, opID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, [32]byte) error); ok {
		r1 = rf(ctx, address, opID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TimelockInspector_IsOperation_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsOperation'
type TimelockInspector_IsOperation_Call struct {
	*mock.Call
}

// IsOperation is a helper method to define mock.On call
//   - ctx context.Context
//   - address string
//   - opID [32]byte
func (_e *TimelockInspector_Expecter) IsOperation(ctx interface{}, address interface{}, opID interface{}) *TimelockInspector_IsOperation_Call {
	return &TimelockInspector_IsOperation_Call{Call: _e.mock.On("IsOperation", ctx, address, opID)}
}

func (_c *TimelockInspector_IsOperation_Call) Run(run func(ctx context.Context, address string, opID [32]byte)) *TimelockInspector_IsOperation_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].([32]byte))
	})
	return _c
}

func (_c *TimelockInspector_IsOperation_Call) Return(_a0 bool, _a1 error) *TimelockInspector_IsOperation_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *TimelockInspector_IsOperation_Call) RunAndReturn(run func(context.Context, string, [32]byte) (bool, error)) *TimelockInspector_IsOperation_Call {
	_c.Call.Return(run)
	return _c
}

// IsOperationDone provides a mock function with given fields: ctx, address, opID
func (_m *TimelockInspector) IsOperationDone(ctx context.Context, address string, opID [32]byte) (bool, error) {
	ret := _m.Called(ctx, address, opID)

	if len(ret) == 0 {
		panic("no return value specified for IsOperationDone")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, [32]byte) (bool, error)); ok {
		return rf(ctx, address, opID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, [32]byte) bool); ok {
		r0 = rf(ctx, address, opID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, [32]byte) error); ok {
		r1 = rf(ctx, address, opID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TimelockInspector_IsOperationDone_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsOperationDone'
type TimelockInspector_IsOperationDone_Call struct {
	*mock.Call
}

// IsOperationDone is a helper method to define mock.On call
//   - ctx context.Context
//   - address string
//   - opID [32]byte
func (_e *TimelockInspector_Expecter) IsOperationDone(ctx interface{}, address interface{}, opID interface{}) *TimelockInspector_IsOperationDone_Call {
	return &TimelockInspector_IsOperationDone_Call{Call: _e.mock.On("IsOperationDone", ctx, address, opID)}
}

func (_c *TimelockInspector_IsOperationDone_Call) Run(run func(ctx context.Context, address string, opID [32]byte)) *TimelockInspector_IsOperationDone_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].([32]byte))
	})
	return _c
}

func (_c *TimelockInspector_IsOperationDone_Call) Return(_a0 bool, _a1 error) *TimelockInspector_IsOperationDone_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *TimelockInspector_IsOperationDone_Call) RunAndReturn(run func(context.Context, string, [32]byte) (bool, error)) *TimelockInspector_IsOperationDone_Call {
	_c.Call.Return(run)
	return _c
}

// IsOperationPending provides a mock function with given fields: ctx, address, opID
func (_m *TimelockInspector) IsOperationPending(ctx context.Context, address string, opID [32]byte) (bool, error) {
	ret := _m.Called(ctx, address, opID)

	if len(ret) == 0 {
		panic("no return value specified for IsOperationPending")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, [32]byte) (bool, error)); ok {
		return rf(ctx, address, opID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, [32]byte) bool); ok {
		r0 = rf(ctx, address, opID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, [32]byte) error); ok {
		r1 = rf(ctx, address, opID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TimelockInspector_IsOperationPending_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsOperationPending'
type TimelockInspector_IsOperationPending_Call struct {
	*mock.Call
}

// IsOperationPending is a helper method to define mock.On call
//   - ctx context.Context
//   - address string
//   - opID [32]byte
func (_e *TimelockInspector_Expecter) IsOperationPending(ctx interface{}, address interface{}, opID interface{}) *TimelockInspector_IsOperationPending_Call {
	return &TimelockInspector_IsOperationPending_Call{Call: _e.mock.On("IsOperationPending", ctx, address, opID)}
}

func (_c *TimelockInspector_IsOperationPending_Call) Run(run func(ctx context.Context, address string, opID [32]byte)) *TimelockInspector_IsOperationPending_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].([32]byte))
	})
	return _c
}

func (_c *TimelockInspector_IsOperationPending_Call) Return(_a0 bool, _a1 error) *TimelockInspector_IsOperationPending_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *TimelockInspector_IsOperationPending_Call) RunAndReturn(run func(context.Context, string, [32]byte) (bool, error)) *TimelockInspector_IsOperationPending_Call {
	_c.Call.Return(run)
	return _c
}

// IsOperationReady provides a mock function with given fields: ctx, address, opID
func (_m *TimelockInspector) IsOperationReady(ctx context.Context, address string, opID [32]byte) (bool, error) {
	ret := _m.Called(ctx, address, opID)

	if len(ret) == 0 {
		panic("no return value specified for IsOperationReady")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, [32]byte) (bool, error)); ok {
		return rf(ctx, address, opID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, [32]byte) bool); ok {
		r0 = rf(ctx, address, opID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, [32]byte) error); ok {
		r1 = rf(ctx, address, opID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TimelockInspector_IsOperationReady_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsOperationReady'
type TimelockInspector_IsOperationReady_Call struct {
	*mock.Call
}

// IsOperationReady is a helper method to define mock.On call
//   - ctx context.Context
//   - address string
//   - opID [32]byte
func (_e *TimelockInspector_Expecter) IsOperationReady(ctx interface{}, address interface{}, opID interface{}) *TimelockInspector_IsOperationReady_Call {
	return &TimelockInspector_IsOperationReady_Call{Call: _e.mock.On("IsOperationReady", ctx, address, opID)}
}

func (_c *TimelockInspector_IsOperationReady_Call) Run(run func(ctx context.Context, address string, opID [32]byte)) *TimelockInspector_IsOperationReady_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].([32]byte))
	})
	return _c
}

func (_c *TimelockInspector_IsOperationReady_Call) Return(_a0 bool, _a1 error) *TimelockInspector_IsOperationReady_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *TimelockInspector_IsOperationReady_Call) RunAndReturn(run func(context.Context, string, [32]byte) (bool, error)) *TimelockInspector_IsOperationReady_Call {
	_c.Call.Return(run)
	return _c
}

// NewTimelockInspector creates a new instance of TimelockInspector. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewTimelockInspector(t interface {
	mock.TestingT
	Cleanup(func())
}) *TimelockInspector {
	mock := &TimelockInspector{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
