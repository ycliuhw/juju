// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/apiserver/facades/client/modelmanager (interfaces: StatePool,State,Model)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	modelmanager "github.com/juju/juju/apiserver/facades/client/modelmanager"
	state "github.com/juju/juju/state"
)

// MockStatePool is a mock of StatePool interface.
type MockStatePool struct {
	ctrl     *gomock.Controller
	recorder *MockStatePoolMockRecorder
}

// MockStatePoolMockRecorder is the mock recorder for MockStatePool.
type MockStatePoolMockRecorder struct {
	mock *MockStatePool
}

// NewMockStatePool creates a new mock instance.
func NewMockStatePool(ctrl *gomock.Controller) *MockStatePool {
	mock := &MockStatePool{ctrl: ctrl}
	mock.recorder = &MockStatePoolMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStatePool) EXPECT() *MockStatePoolMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockStatePool) Get(arg0 string) (modelmanager.State, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].(modelmanager.State)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockStatePoolMockRecorder) Get(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockStatePool)(nil).Get), arg0)
}

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

// HasUpgradeSeriesLocks mocks base method.
func (m *MockState) HasUpgradeSeriesLocks() (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasUpgradeSeriesLocks")
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// HasUpgradeSeriesLocks indicates an expected call of HasUpgradeSeriesLocks.
func (mr *MockStateMockRecorder) HasUpgradeSeriesLocks() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasUpgradeSeriesLocks", reflect.TypeOf((*MockState)(nil).HasUpgradeSeriesLocks))
}

// Model mocks base method.
func (m *MockState) Model() (modelmanager.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Model")
	ret0, _ := ret[0].(modelmanager.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Model indicates an expected call of Model.
func (mr *MockStateMockRecorder) Model() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Model", reflect.TypeOf((*MockState)(nil).Model))
}

// Release mocks base method.
func (m *MockState) Release() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Release")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Release indicates an expected call of Release.
func (mr *MockStateMockRecorder) Release() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Release", reflect.TypeOf((*MockState)(nil).Release))
}

// MockModel is a mock of Model interface.
type MockModel struct {
	ctrl     *gomock.Controller
	recorder *MockModelMockRecorder
}

// MockModelMockRecorder is the mock recorder for MockModel.
type MockModelMockRecorder struct {
	mock *MockModel
}

// NewMockModel creates a new mock instance.
func NewMockModel(ctrl *gomock.Controller) *MockModel {
	mock := &MockModel{ctrl: ctrl}
	mock.recorder = &MockModelMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModel) EXPECT() *MockModelMockRecorder {
	return m.recorder
}

// IsControllerModel mocks base method.
func (m *MockModel) IsControllerModel() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsControllerModel")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsControllerModel indicates an expected call of IsControllerModel.
func (mr *MockModelMockRecorder) IsControllerModel() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsControllerModel", reflect.TypeOf((*MockModel)(nil).IsControllerModel))
}

// Type mocks base method.
func (m *MockModel) Type() state.ModelType {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Type")
	ret0, _ := ret[0].(state.ModelType)
	return ret0
}

// Type indicates an expected call of Type.
func (mr *MockModelMockRecorder) Type() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Type", reflect.TypeOf((*MockModel)(nil).Type))
}
