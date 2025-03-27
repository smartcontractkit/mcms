// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// DecodedOperation is an autogenerated mock type for the DecodedOperation type
type DecodedOperation struct {
	mock.Mock
}

type DecodedOperation_Expecter struct {
	mock *mock.Mock
}

func (_m *DecodedOperation) EXPECT() *DecodedOperation_Expecter {
	return &DecodedOperation_Expecter{mock: &_m.Mock}
}

// Args provides a mock function with no fields
func (_m *DecodedOperation) Args() []interface{} {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Args")
	}

	var r0 []interface{}
	if rf, ok := ret.Get(0).(func() []interface{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]interface{})
		}
	}

	return r0
}

// DecodedOperation_Args_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Args'
type DecodedOperation_Args_Call struct {
	*mock.Call
}

// Args is a helper method to define mock.On call
func (_e *DecodedOperation_Expecter) Args() *DecodedOperation_Args_Call {
	return &DecodedOperation_Args_Call{Call: _e.mock.On("Args")}
}

func (_c *DecodedOperation_Args_Call) Run(run func()) *DecodedOperation_Args_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *DecodedOperation_Args_Call) Return(_a0 []interface{}) *DecodedOperation_Args_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DecodedOperation_Args_Call) RunAndReturn(run func() []interface{}) *DecodedOperation_Args_Call {
	_c.Call.Return(run)
	return _c
}

// Keys provides a mock function with no fields
func (_m *DecodedOperation) Keys() []string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Keys")
	}

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// DecodedOperation_Keys_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Keys'
type DecodedOperation_Keys_Call struct {
	*mock.Call
}

// Keys is a helper method to define mock.On call
func (_e *DecodedOperation_Expecter) Keys() *DecodedOperation_Keys_Call {
	return &DecodedOperation_Keys_Call{Call: _e.mock.On("Keys")}
}

func (_c *DecodedOperation_Keys_Call) Run(run func()) *DecodedOperation_Keys_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *DecodedOperation_Keys_Call) Return(_a0 []string) *DecodedOperation_Keys_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DecodedOperation_Keys_Call) RunAndReturn(run func() []string) *DecodedOperation_Keys_Call {
	_c.Call.Return(run)
	return _c
}

// MethodName provides a mock function with no fields
func (_m *DecodedOperation) MethodName() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for MethodName")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// DecodedOperation_MethodName_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'MethodName'
type DecodedOperation_MethodName_Call struct {
	*mock.Call
}

// MethodName is a helper method to define mock.On call
func (_e *DecodedOperation_Expecter) MethodName() *DecodedOperation_MethodName_Call {
	return &DecodedOperation_MethodName_Call{Call: _e.mock.On("MethodName")}
}

func (_c *DecodedOperation_MethodName_Call) Run(run func()) *DecodedOperation_MethodName_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *DecodedOperation_MethodName_Call) Return(_a0 string) *DecodedOperation_MethodName_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DecodedOperation_MethodName_Call) RunAndReturn(run func() string) *DecodedOperation_MethodName_Call {
	_c.Call.Return(run)
	return _c
}

// String provides a mock function with no fields
func (_m *DecodedOperation) String() (string, string, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for String")
	}

	var r0 string
	var r1 string
	var r2 error
	if rf, ok := ret.Get(0).(func() (string, string, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func() string); ok {
		r1 = rf()
	} else {
		r1 = ret.Get(1).(string)
	}

	if rf, ok := ret.Get(2).(func() error); ok {
		r2 = rf()
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// DecodedOperation_String_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'String'
type DecodedOperation_String_Call struct {
	*mock.Call
}

// String is a helper method to define mock.On call
func (_e *DecodedOperation_Expecter) String() *DecodedOperation_String_Call {
	return &DecodedOperation_String_Call{Call: _e.mock.On("String")}
}

func (_c *DecodedOperation_String_Call) Run(run func()) *DecodedOperation_String_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *DecodedOperation_String_Call) Return(_a0 string, _a1 string, _a2 error) *DecodedOperation_String_Call {
	_c.Call.Return(_a0, _a1, _a2)
	return _c
}

func (_c *DecodedOperation_String_Call) RunAndReturn(run func() (string, string, error)) *DecodedOperation_String_Call {
	_c.Call.Return(run)
	return _c
}

// NewDecodedOperation creates a new instance of DecodedOperation. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDecodedOperation(t interface {
	mock.TestingT
	Cleanup(func())
}) *DecodedOperation {
	mock := &DecodedOperation{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
