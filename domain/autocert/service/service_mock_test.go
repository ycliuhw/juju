// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/domain/autocert/service (interfaces: State)
//
// Generated by this command:
//
//	mockgen -typed -package service -destination service_mock_test.go github.com/juju/juju/domain/autocert/service State
//

// Package service is a generated GoMock package.
package service

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockState is a mock of State interface.
type MockState struct {
	ctrl     *gomock.Controller
	recorder *MockStateMockRecorder
}

// MockStateMockRecorder is the mock recorder for MockState.
type MockStateMockRecorder struct {
	mock *MockState
}

// NewMockState creates a new mock instance.
func NewMockState(ctrl *gomock.Controller) *MockState {
	mock := &MockState{ctrl: ctrl}
	mock.recorder = &MockStateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockState) EXPECT() *MockStateMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockState) Delete(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockStateMockRecorder) Delete(arg0, arg1 any) *MockStateDeleteCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockState)(nil).Delete), arg0, arg1)
	return &MockStateDeleteCall{Call: call}
}

// MockStateDeleteCall wrap *gomock.Call
type MockStateDeleteCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateDeleteCall) Return(arg0 error) *MockStateDeleteCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateDeleteCall) Do(f func(context.Context, string) error) *MockStateDeleteCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateDeleteCall) DoAndReturn(f func(context.Context, string) error) *MockStateDeleteCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Get mocks base method.
func (m *MockState) Get(arg0 context.Context, arg1 string) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockStateMockRecorder) Get(arg0, arg1 any) *MockStateGetCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockState)(nil).Get), arg0, arg1)
	return &MockStateGetCall{Call: call}
}

// MockStateGetCall wrap *gomock.Call
type MockStateGetCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateGetCall) Return(arg0 []byte, arg1 error) *MockStateGetCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateGetCall) Do(f func(context.Context, string) ([]byte, error)) *MockStateGetCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateGetCall) DoAndReturn(f func(context.Context, string) ([]byte, error)) *MockStateGetCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Put mocks base method.
func (m *MockState) Put(arg0 context.Context, arg1 string, arg2 []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Put", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Put indicates an expected call of Put.
func (mr *MockStateMockRecorder) Put(arg0, arg1, arg2 any) *MockStatePutCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Put", reflect.TypeOf((*MockState)(nil).Put), arg0, arg1, arg2)
	return &MockStatePutCall{Call: call}
}

// MockStatePutCall wrap *gomock.Call
type MockStatePutCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStatePutCall) Return(arg0 error) *MockStatePutCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStatePutCall) Do(f func(context.Context, string, []byte) error) *MockStatePutCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStatePutCall) DoAndReturn(f func(context.Context, string, []byte) error) *MockStatePutCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
