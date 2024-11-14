// Code generated by mockery v2.46.3. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"

	types "github.com/smartcontractkit/mcms/types"
)

// Configurator is an autogenerated mock type for the Configurator type
type Configurator[R any] struct {
	mock.Mock
}

type Configurator_Expecter[R any] struct {
	mock *mock.Mock
}

func (_m *Configurator[R]) EXPECT() *Configurator_Expecter[R] {
	return &Configurator_Expecter[R]{mock: &_m.Mock}
}

// ToChainConfig provides a mock function with given fields: contract, cfg
func (_m *Configurator[R]) ToChainConfig(contract string, cfg types.Config) (R, error) {
	ret := _m.Called(contract, cfg)

	if len(ret) == 0 {
		panic("no return value specified for ToChainConfig")
	}

	var r0 R
	var r1 error
	if rf, ok := ret.Get(0).(func(string, types.Config) (R, error)); ok {
		return rf(contract, cfg)
	}
	if rf, ok := ret.Get(0).(func(string, types.Config) R); ok {
		r0 = rf(contract, cfg)
	} else {
		r0 = ret.Get(0).(R)
	}

	if rf, ok := ret.Get(1).(func(string, types.Config) error); ok {
		r1 = rf(contract, cfg)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Configurator_SetConfigInputs_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ToChainConfig'
type Configurator_SetConfigInputs_Call[R any] struct {
	*mock.Call
}

// SetConfigInputs is a helper method to define mock.On call
//   - contract string
//   - cfg types.Config
func (_e *Configurator_Expecter[R]) SetConfigInputs(contract interface{}, cfg interface{}) *Configurator_SetConfigInputs_Call[R] {
	return &Configurator_SetConfigInputs_Call[R]{Call: _e.mock.On("ToChainConfig", contract, cfg)}
}

func (_c *Configurator_SetConfigInputs_Call[R]) Run(run func(contract string, cfg types.Config)) *Configurator_SetConfigInputs_Call[R] {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(types.Config))
	})
	return _c
}

func (_c *Configurator_SetConfigInputs_Call[R]) Return(_a0 R, _a1 error) *Configurator_SetConfigInputs_Call[R] {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Configurator_SetConfigInputs_Call[R]) RunAndReturn(run func(string, types.Config) (R, error)) *Configurator_SetConfigInputs_Call[R] {
	_c.Call.Return(run)
	return _c
}

// ToConfig provides a mock function with given fields: onchainConfig
func (_m *Configurator[R]) ToConfig(onchainConfig R) (*types.Config, error) {
	ret := _m.Called(onchainConfig)

	if len(ret) == 0 {
		panic("no return value specified for ToConfig")
	}

	var r0 *types.Config
	var r1 error
	if rf, ok := ret.Get(0).(func(R) (*types.Config, error)); ok {
		return rf(onchainConfig)
	}
	if rf, ok := ret.Get(0).(func(R) *types.Config); ok {
		r0 = rf(onchainConfig)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Config)
		}
	}

	if rf, ok := ret.Get(1).(func(R) error); ok {
		r1 = rf(onchainConfig)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Configurator_ToConfig_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ToConfig'
type Configurator_ToConfig_Call[R any] struct {
	*mock.Call
}

// ToConfig is a helper method to define mock.On call
//   - onchainConfig R
func (_e *Configurator_Expecter[R]) ToConfig(onchainConfig interface{}) *Configurator_ToConfig_Call[R] {
	return &Configurator_ToConfig_Call[R]{Call: _e.mock.On("ToConfig", onchainConfig)}
}

func (_c *Configurator_ToConfig_Call[R]) Run(run func(onchainConfig R)) *Configurator_ToConfig_Call[R] {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(R))
	})
	return _c
}

func (_c *Configurator_ToConfig_Call[R]) Return(_a0 *types.Config, _a1 error) *Configurator_ToConfig_Call[R] {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Configurator_ToConfig_Call[R]) RunAndReturn(run func(R) (*types.Config, error)) *Configurator_ToConfig_Call[R] {
	_c.Call.Return(run)
	return _c
}

// NewConfigurator creates a new instance of Configurator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewConfigurator[R any](t interface {
	mock.TestingT
	Cleanup(func())
}) *Configurator[R] {
	mock := &Configurator[R]{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
