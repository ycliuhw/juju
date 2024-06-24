// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/domain/machine/service (interfaces: State)
//
// Generated by this command:
//
//	mockgen -typed -package service -destination package_mock_test.go github.com/juju/juju/domain/machine/service State
//

// Package service is a generated GoMock package.
package service

import (
	context "context"
	reflect "reflect"

	instance "github.com/juju/juju/core/instance"
	life "github.com/juju/juju/domain/life"
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

// AllMachines mocks base method.
func (m *MockState) AllMachines(arg0 context.Context) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AllMachines", arg0)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AllMachines indicates an expected call of AllMachines.
func (mr *MockStateMockRecorder) AllMachines(arg0 any) *MockStateAllMachinesCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AllMachines", reflect.TypeOf((*MockState)(nil).AllMachines), arg0)
	return &MockStateAllMachinesCall{Call: call}
}

// MockStateAllMachinesCall wrap *gomock.Call
type MockStateAllMachinesCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateAllMachinesCall) Return(arg0 []string, arg1 error) *MockStateAllMachinesCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateAllMachinesCall) Do(f func(context.Context) ([]string, error)) *MockStateAllMachinesCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateAllMachinesCall) DoAndReturn(f func(context.Context) ([]string, error)) *MockStateAllMachinesCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// CreateMachine mocks base method.
func (m *MockState) CreateMachine(arg0 context.Context, arg1, arg2, arg3 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateMachine", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateMachine indicates an expected call of CreateMachine.
func (mr *MockStateMockRecorder) CreateMachine(arg0, arg1, arg2, arg3 any) *MockStateCreateMachineCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateMachine", reflect.TypeOf((*MockState)(nil).CreateMachine), arg0, arg1, arg2, arg3)
	return &MockStateCreateMachineCall{Call: call}
}

// MockStateCreateMachineCall wrap *gomock.Call
type MockStateCreateMachineCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateCreateMachineCall) Return(arg0 error) *MockStateCreateMachineCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateCreateMachineCall) Do(f func(context.Context, string, string, string) error) *MockStateCreateMachineCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateCreateMachineCall) DoAndReturn(f func(context.Context, string, string, string) error) *MockStateCreateMachineCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// DeleteMachine mocks base method.
func (m *MockState) DeleteMachine(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteMachine", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteMachine indicates an expected call of DeleteMachine.
func (mr *MockStateMockRecorder) DeleteMachine(arg0, arg1 any) *MockStateDeleteMachineCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteMachine", reflect.TypeOf((*MockState)(nil).DeleteMachine), arg0, arg1)
	return &MockStateDeleteMachineCall{Call: call}
}

// MockStateDeleteMachineCall wrap *gomock.Call
type MockStateDeleteMachineCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateDeleteMachineCall) Return(arg0 error) *MockStateDeleteMachineCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateDeleteMachineCall) Do(f func(context.Context, string) error) *MockStateDeleteMachineCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateDeleteMachineCall) DoAndReturn(f func(context.Context, string) error) *MockStateDeleteMachineCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// DeleteMachineCloudInstance mocks base method.
func (m *MockState) DeleteMachineCloudInstance(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteMachineCloudInstance", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteMachineCloudInstance indicates an expected call of DeleteMachineCloudInstance.
func (mr *MockStateMockRecorder) DeleteMachineCloudInstance(arg0, arg1 any) *MockStateDeleteMachineCloudInstanceCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteMachineCloudInstance", reflect.TypeOf((*MockState)(nil).DeleteMachineCloudInstance), arg0, arg1)
	return &MockStateDeleteMachineCloudInstanceCall{Call: call}
}

// MockStateDeleteMachineCloudInstanceCall wrap *gomock.Call
type MockStateDeleteMachineCloudInstanceCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateDeleteMachineCloudInstanceCall) Return(arg0 error) *MockStateDeleteMachineCloudInstanceCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateDeleteMachineCloudInstanceCall) Do(f func(context.Context, string) error) *MockStateDeleteMachineCloudInstanceCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateDeleteMachineCloudInstanceCall) DoAndReturn(f func(context.Context, string) error) *MockStateDeleteMachineCloudInstanceCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// GetMachineLife mocks base method.
func (m *MockState) GetMachineLife(arg0 context.Context, arg1 string) (*life.Life, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMachineLife", arg0, arg1)
	ret0, _ := ret[0].(*life.Life)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMachineLife indicates an expected call of GetMachineLife.
func (mr *MockStateMockRecorder) GetMachineLife(arg0, arg1 any) *MockStateGetMachineLifeCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMachineLife", reflect.TypeOf((*MockState)(nil).GetMachineLife), arg0, arg1)
	return &MockStateGetMachineLifeCall{Call: call}
}

// MockStateGetMachineLifeCall wrap *gomock.Call
type MockStateGetMachineLifeCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateGetMachineLifeCall) Return(arg0 *life.Life, arg1 error) *MockStateGetMachineLifeCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateGetMachineLifeCall) Do(f func(context.Context, string) (*life.Life, error)) *MockStateGetMachineLifeCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateGetMachineLifeCall) DoAndReturn(f func(context.Context, string) (*life.Life, error)) *MockStateGetMachineLifeCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// HardwareCharacteristics mocks base method.
func (m *MockState) HardwareCharacteristics(arg0 context.Context, arg1 string) (*instance.HardwareCharacteristics, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HardwareCharacteristics", arg0, arg1)
	ret0, _ := ret[0].(*instance.HardwareCharacteristics)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// HardwareCharacteristics indicates an expected call of HardwareCharacteristics.
func (mr *MockStateMockRecorder) HardwareCharacteristics(arg0, arg1 any) *MockStateHardwareCharacteristicsCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HardwareCharacteristics", reflect.TypeOf((*MockState)(nil).HardwareCharacteristics), arg0, arg1)
	return &MockStateHardwareCharacteristicsCall{Call: call}
}

// MockStateHardwareCharacteristicsCall wrap *gomock.Call
type MockStateHardwareCharacteristicsCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateHardwareCharacteristicsCall) Return(arg0 *instance.HardwareCharacteristics, arg1 error) *MockStateHardwareCharacteristicsCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateHardwareCharacteristicsCall) Do(f func(context.Context, string) (*instance.HardwareCharacteristics, error)) *MockStateHardwareCharacteristicsCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateHardwareCharacteristicsCall) DoAndReturn(f func(context.Context, string) (*instance.HardwareCharacteristics, error)) *MockStateHardwareCharacteristicsCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// InitialWatchStatement mocks base method.
func (m *MockState) InitialWatchStatement() (string, string) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InitialWatchStatement")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(string)
	return ret0, ret1
}

// InitialWatchStatement indicates an expected call of InitialWatchStatement.
func (mr *MockStateMockRecorder) InitialWatchStatement() *MockStateInitialWatchStatementCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InitialWatchStatement", reflect.TypeOf((*MockState)(nil).InitialWatchStatement))
	return &MockStateInitialWatchStatementCall{Call: call}
}

// MockStateInitialWatchStatementCall wrap *gomock.Call
type MockStateInitialWatchStatementCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateInitialWatchStatementCall) Return(arg0, arg1 string) *MockStateInitialWatchStatementCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateInitialWatchStatementCall) Do(f func() (string, string)) *MockStateInitialWatchStatementCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateInitialWatchStatementCall) DoAndReturn(f func() (string, string)) *MockStateInitialWatchStatementCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// InstanceId mocks base method.
func (m *MockState) InstanceId(arg0 context.Context, arg1 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InstanceId", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// InstanceId indicates an expected call of InstanceId.
func (mr *MockStateMockRecorder) InstanceId(arg0, arg1 any) *MockStateInstanceIdCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InstanceId", reflect.TypeOf((*MockState)(nil).InstanceId), arg0, arg1)
	return &MockStateInstanceIdCall{Call: call}
}

// MockStateInstanceIdCall wrap *gomock.Call
type MockStateInstanceIdCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateInstanceIdCall) Return(arg0 string, arg1 error) *MockStateInstanceIdCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateInstanceIdCall) Do(f func(context.Context, string) (string, error)) *MockStateInstanceIdCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateInstanceIdCall) DoAndReturn(f func(context.Context, string) (string, error)) *MockStateInstanceIdCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// SetMachineCloudInstance mocks base method.
func (m *MockState) SetMachineCloudInstance(arg0 context.Context, arg1 string, arg2 instance.Id, arg3 instance.HardwareCharacteristics) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetMachineCloudInstance", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetMachineCloudInstance indicates an expected call of SetMachineCloudInstance.
func (mr *MockStateMockRecorder) SetMachineCloudInstance(arg0, arg1, arg2, arg3 any) *MockStateSetMachineCloudInstanceCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetMachineCloudInstance", reflect.TypeOf((*MockState)(nil).SetMachineCloudInstance), arg0, arg1, arg2, arg3)
	return &MockStateSetMachineCloudInstanceCall{Call: call}
}

// MockStateSetMachineCloudInstanceCall wrap *gomock.Call
type MockStateSetMachineCloudInstanceCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateSetMachineCloudInstanceCall) Return(arg0 error) *MockStateSetMachineCloudInstanceCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateSetMachineCloudInstanceCall) Do(f func(context.Context, string, instance.Id, instance.HardwareCharacteristics) error) *MockStateSetMachineCloudInstanceCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateSetMachineCloudInstanceCall) DoAndReturn(f func(context.Context, string, instance.Id, instance.HardwareCharacteristics) error) *MockStateSetMachineCloudInstanceCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
