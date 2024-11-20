// Code generated by mockery v2.48.0. DO NOT EDIT.

package mocks

import (
	bind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	bindings "github.com/smartcontractkit/mcms/sdk/evm/bindings"

	common "github.com/ethereum/go-ethereum/common"

	event "github.com/ethereum/go-ethereum/event"

	mock "github.com/stretchr/testify/mock"

	types "github.com/ethereum/go-ethereum/core/types"
)

// CallProxyInterface is an autogenerated mock type for the CallProxyInterface type
type CallProxyInterface struct {
	mock.Mock
}

type CallProxyInterface_Expecter struct {
	mock *mock.Mock
}

func (_m *CallProxyInterface) EXPECT() *CallProxyInterface_Expecter {
	return &CallProxyInterface_Expecter{mock: &_m.Mock}
}

// Address provides a mock function with given fields:
func (_m *CallProxyInterface) Address() common.Address {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Address")
	}

	var r0 common.Address
	if rf, ok := ret.Get(0).(func() common.Address); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Address)
		}
	}

	return r0
}

// CallProxyInterface_Address_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Address'
type CallProxyInterface_Address_Call struct {
	*mock.Call
}

// Address is a helper method to define mock.On call
func (_e *CallProxyInterface_Expecter) Address() *CallProxyInterface_Address_Call {
	return &CallProxyInterface_Address_Call{Call: _e.mock.On("Address")}
}

func (_c *CallProxyInterface_Address_Call) Run(run func()) *CallProxyInterface_Address_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *CallProxyInterface_Address_Call) Return(_a0 common.Address) *CallProxyInterface_Address_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *CallProxyInterface_Address_Call) RunAndReturn(run func() common.Address) *CallProxyInterface_Address_Call {
	_c.Call.Return(run)
	return _c
}

// Fallback provides a mock function with given fields: opts, calldata
func (_m *CallProxyInterface) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	ret := _m.Called(opts, calldata)

	if len(ret) == 0 {
		panic("no return value specified for Fallback")
	}

	var r0 *types.Transaction
	var r1 error
	if rf, ok := ret.Get(0).(func(*bind.TransactOpts, []byte) (*types.Transaction, error)); ok {
		return rf(opts, calldata)
	}
	if rf, ok := ret.Get(0).(func(*bind.TransactOpts, []byte) *types.Transaction); ok {
		r0 = rf(opts, calldata)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Transaction)
		}
	}

	if rf, ok := ret.Get(1).(func(*bind.TransactOpts, []byte) error); ok {
		r1 = rf(opts, calldata)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CallProxyInterface_Fallback_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Fallback'
type CallProxyInterface_Fallback_Call struct {
	*mock.Call
}

// Fallback is a helper method to define mock.On call
//   - opts *bind.TransactOpts
//   - calldata []byte
func (_e *CallProxyInterface_Expecter) Fallback(opts interface{}, calldata interface{}) *CallProxyInterface_Fallback_Call {
	return &CallProxyInterface_Fallback_Call{Call: _e.mock.On("Fallback", opts, calldata)}
}

func (_c *CallProxyInterface_Fallback_Call) Run(run func(opts *bind.TransactOpts, calldata []byte)) *CallProxyInterface_Fallback_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*bind.TransactOpts), args[1].([]byte))
	})
	return _c
}

func (_c *CallProxyInterface_Fallback_Call) Return(_a0 *types.Transaction, _a1 error) *CallProxyInterface_Fallback_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *CallProxyInterface_Fallback_Call) RunAndReturn(run func(*bind.TransactOpts, []byte) (*types.Transaction, error)) *CallProxyInterface_Fallback_Call {
	_c.Call.Return(run)
	return _c
}

// FilterTargetSet provides a mock function with given fields: opts
func (_m *CallProxyInterface) FilterTargetSet(opts *bind.FilterOpts) (*bindings.CallProxyTargetSetIterator, error) {
	ret := _m.Called(opts)

	if len(ret) == 0 {
		panic("no return value specified for FilterTargetSet")
	}

	var r0 *bindings.CallProxyTargetSetIterator
	var r1 error
	if rf, ok := ret.Get(0).(func(*bind.FilterOpts) (*bindings.CallProxyTargetSetIterator, error)); ok {
		return rf(opts)
	}
	if rf, ok := ret.Get(0).(func(*bind.FilterOpts) *bindings.CallProxyTargetSetIterator); ok {
		r0 = rf(opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bindings.CallProxyTargetSetIterator)
		}
	}

	if rf, ok := ret.Get(1).(func(*bind.FilterOpts) error); ok {
		r1 = rf(opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CallProxyInterface_FilterTargetSet_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'FilterTargetSet'
type CallProxyInterface_FilterTargetSet_Call struct {
	*mock.Call
}

// FilterTargetSet is a helper method to define mock.On call
//   - opts *bind.FilterOpts
func (_e *CallProxyInterface_Expecter) FilterTargetSet(opts interface{}) *CallProxyInterface_FilterTargetSet_Call {
	return &CallProxyInterface_FilterTargetSet_Call{Call: _e.mock.On("FilterTargetSet", opts)}
}

func (_c *CallProxyInterface_FilterTargetSet_Call) Run(run func(opts *bind.FilterOpts)) *CallProxyInterface_FilterTargetSet_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*bind.FilterOpts))
	})
	return _c
}

func (_c *CallProxyInterface_FilterTargetSet_Call) Return(_a0 *bindings.CallProxyTargetSetIterator, _a1 error) *CallProxyInterface_FilterTargetSet_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *CallProxyInterface_FilterTargetSet_Call) RunAndReturn(run func(*bind.FilterOpts) (*bindings.CallProxyTargetSetIterator, error)) *CallProxyInterface_FilterTargetSet_Call {
	_c.Call.Return(run)
	return _c
}

// ParseLog provides a mock function with given fields: log
func (_m *CallProxyInterface) ParseLog(log types.Log) (bindings.AbigenLog, error) {
	ret := _m.Called(log)

	if len(ret) == 0 {
		panic("no return value specified for ParseLog")
	}

	var r0 bindings.AbigenLog
	var r1 error
	if rf, ok := ret.Get(0).(func(types.Log) (bindings.AbigenLog, error)); ok {
		return rf(log)
	}
	if rf, ok := ret.Get(0).(func(types.Log) bindings.AbigenLog); ok {
		r0 = rf(log)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(bindings.AbigenLog)
		}
	}

	if rf, ok := ret.Get(1).(func(types.Log) error); ok {
		r1 = rf(log)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CallProxyInterface_ParseLog_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ParseLog'
type CallProxyInterface_ParseLog_Call struct {
	*mock.Call
}

// ParseLog is a helper method to define mock.On call
//   - log types.Log
func (_e *CallProxyInterface_Expecter) ParseLog(log interface{}) *CallProxyInterface_ParseLog_Call {
	return &CallProxyInterface_ParseLog_Call{Call: _e.mock.On("ParseLog", log)}
}

func (_c *CallProxyInterface_ParseLog_Call) Run(run func(log types.Log)) *CallProxyInterface_ParseLog_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.Log))
	})
	return _c
}

func (_c *CallProxyInterface_ParseLog_Call) Return(_a0 bindings.AbigenLog, _a1 error) *CallProxyInterface_ParseLog_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *CallProxyInterface_ParseLog_Call) RunAndReturn(run func(types.Log) (bindings.AbigenLog, error)) *CallProxyInterface_ParseLog_Call {
	_c.Call.Return(run)
	return _c
}

// ParseTargetSet provides a mock function with given fields: log
func (_m *CallProxyInterface) ParseTargetSet(log types.Log) (*bindings.CallProxyTargetSet, error) {
	ret := _m.Called(log)

	if len(ret) == 0 {
		panic("no return value specified for ParseTargetSet")
	}

	var r0 *bindings.CallProxyTargetSet
	var r1 error
	if rf, ok := ret.Get(0).(func(types.Log) (*bindings.CallProxyTargetSet, error)); ok {
		return rf(log)
	}
	if rf, ok := ret.Get(0).(func(types.Log) *bindings.CallProxyTargetSet); ok {
		r0 = rf(log)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bindings.CallProxyTargetSet)
		}
	}

	if rf, ok := ret.Get(1).(func(types.Log) error); ok {
		r1 = rf(log)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CallProxyInterface_ParseTargetSet_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ParseTargetSet'
type CallProxyInterface_ParseTargetSet_Call struct {
	*mock.Call
}

// ParseTargetSet is a helper method to define mock.On call
//   - log types.Log
func (_e *CallProxyInterface_Expecter) ParseTargetSet(log interface{}) *CallProxyInterface_ParseTargetSet_Call {
	return &CallProxyInterface_ParseTargetSet_Call{Call: _e.mock.On("ParseTargetSet", log)}
}

func (_c *CallProxyInterface_ParseTargetSet_Call) Run(run func(log types.Log)) *CallProxyInterface_ParseTargetSet_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.Log))
	})
	return _c
}

func (_c *CallProxyInterface_ParseTargetSet_Call) Return(_a0 *bindings.CallProxyTargetSet, _a1 error) *CallProxyInterface_ParseTargetSet_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *CallProxyInterface_ParseTargetSet_Call) RunAndReturn(run func(types.Log) (*bindings.CallProxyTargetSet, error)) *CallProxyInterface_ParseTargetSet_Call {
	_c.Call.Return(run)
	return _c
}

// WatchTargetSet provides a mock function with given fields: opts, sink
func (_m *CallProxyInterface) WatchTargetSet(opts *bind.WatchOpts, sink chan<- *bindings.CallProxyTargetSet) (event.Subscription, error) {
	ret := _m.Called(opts, sink)

	if len(ret) == 0 {
		panic("no return value specified for WatchTargetSet")
	}

	var r0 event.Subscription
	var r1 error
	if rf, ok := ret.Get(0).(func(*bind.WatchOpts, chan<- *bindings.CallProxyTargetSet) (event.Subscription, error)); ok {
		return rf(opts, sink)
	}
	if rf, ok := ret.Get(0).(func(*bind.WatchOpts, chan<- *bindings.CallProxyTargetSet) event.Subscription); ok {
		r0 = rf(opts, sink)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(event.Subscription)
		}
	}

	if rf, ok := ret.Get(1).(func(*bind.WatchOpts, chan<- *bindings.CallProxyTargetSet) error); ok {
		r1 = rf(opts, sink)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CallProxyInterface_WatchTargetSet_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WatchTargetSet'
type CallProxyInterface_WatchTargetSet_Call struct {
	*mock.Call
}

// WatchTargetSet is a helper method to define mock.On call
//   - opts *bind.WatchOpts
//   - sink chan<- *bindings.CallProxyTargetSet
func (_e *CallProxyInterface_Expecter) WatchTargetSet(opts interface{}, sink interface{}) *CallProxyInterface_WatchTargetSet_Call {
	return &CallProxyInterface_WatchTargetSet_Call{Call: _e.mock.On("WatchTargetSet", opts, sink)}
}

func (_c *CallProxyInterface_WatchTargetSet_Call) Run(run func(opts *bind.WatchOpts, sink chan<- *bindings.CallProxyTargetSet)) *CallProxyInterface_WatchTargetSet_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*bind.WatchOpts), args[1].(chan<- *bindings.CallProxyTargetSet))
	})
	return _c
}

func (_c *CallProxyInterface_WatchTargetSet_Call) Return(_a0 event.Subscription, _a1 error) *CallProxyInterface_WatchTargetSet_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *CallProxyInterface_WatchTargetSet_Call) RunAndReturn(run func(*bind.WatchOpts, chan<- *bindings.CallProxyTargetSet) (event.Subscription, error)) *CallProxyInterface_WatchTargetSet_Call {
	_c.Call.Return(run)
	return _c
}

// NewCallProxyInterface creates a new instance of CallProxyInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewCallProxyInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *CallProxyInterface {
	mock := &CallProxyInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
