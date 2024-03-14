// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/domain/secretbackend/service (interfaces: State)
//
// Generated by this command:
//
//	mockgen -package service -destination state_mock_test.go github.com/juju/juju/domain/secretbackend/service State
//

// Package service is a generated GoMock package.
package service

import (
	context "context"
	reflect "reflect"
	time "time"

	secrets "github.com/juju/juju/core/secrets"
	secretbackend "github.com/juju/juju/domain/secretbackend"
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

// CreateSecretBackend mocks base method.
func (m *MockState) CreateSecretBackend(arg0 context.Context, arg1 secretbackend.CreateSecretBackendParams) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateSecretBackend", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateSecretBackend indicates an expected call of CreateSecretBackend.
func (mr *MockStateMockRecorder) CreateSecretBackend(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSecretBackend", reflect.TypeOf((*MockState)(nil).CreateSecretBackend), arg0, arg1)
}

// DeleteSecretBackend mocks base method.
func (m *MockState) DeleteSecretBackend(arg0 context.Context, arg1 string, arg2 bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteSecretBackend", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteSecretBackend indicates an expected call of DeleteSecretBackend.
func (mr *MockStateMockRecorder) DeleteSecretBackend(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteSecretBackend", reflect.TypeOf((*MockState)(nil).DeleteSecretBackend), arg0, arg1, arg2)
}

// GetSecretBackend mocks base method.
func (m *MockState) GetSecretBackend(arg0 context.Context, arg1 string) (*secrets.SecretBackend, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSecretBackend", arg0, arg1)
	ret0, _ := ret[0].(*secrets.SecretBackend)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSecretBackend indicates an expected call of GetSecretBackend.
func (mr *MockStateMockRecorder) GetSecretBackend(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSecretBackend", reflect.TypeOf((*MockState)(nil).GetSecretBackend), arg0, arg1)
}

// GetSecretBackendByName mocks base method.
func (m *MockState) GetSecretBackendByName(arg0 context.Context, arg1 string) (*secrets.SecretBackend, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSecretBackendByName", arg0, arg1)
	ret0, _ := ret[0].(*secrets.SecretBackend)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSecretBackendByName indicates an expected call of GetSecretBackendByName.
func (mr *MockStateMockRecorder) GetSecretBackendByName(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSecretBackendByName", reflect.TypeOf((*MockState)(nil).GetSecretBackendByName), arg0, arg1)
}

// ListSecretBackends mocks base method.
func (m *MockState) ListSecretBackends(arg0 context.Context) ([]*secretbackend.SecretBackendInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListSecretBackends", arg0)
	ret0, _ := ret[0].([]*secretbackend.SecretBackendInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListSecretBackends indicates an expected call of ListSecretBackends.
func (mr *MockStateMockRecorder) ListSecretBackends(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListSecretBackends", reflect.TypeOf((*MockState)(nil).ListSecretBackends), arg0)
}

// SecretBackendRotated mocks base method.
func (m *MockState) SecretBackendRotated(arg0 context.Context, arg1 string, arg2 time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SecretBackendRotated", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// SecretBackendRotated indicates an expected call of SecretBackendRotated.
func (mr *MockStateMockRecorder) SecretBackendRotated(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SecretBackendRotated", reflect.TypeOf((*MockState)(nil).SecretBackendRotated), arg0, arg1, arg2)
}

// UpdateSecretBackend mocks base method.
func (m *MockState) UpdateSecretBackend(arg0 context.Context, arg1 secretbackend.UpdateSecretBackendParams) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateSecretBackend", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateSecretBackend indicates an expected call of UpdateSecretBackend.
func (mr *MockStateMockRecorder) UpdateSecretBackend(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateSecretBackend", reflect.TypeOf((*MockState)(nil).UpdateSecretBackend), arg0, arg1)
}

// WatchSecretBackendRotationChanges mocks base method.
func (m *MockState) WatchSecretBackendRotationChanges(arg0 secretbackend.WatcherFactory) (secretbackend.SecretBackendRotateWatcher, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchSecretBackendRotationChanges", arg0)
	ret0, _ := ret[0].(secretbackend.SecretBackendRotateWatcher)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchSecretBackendRotationChanges indicates an expected call of WatchSecretBackendRotationChanges.
func (mr *MockStateMockRecorder) WatchSecretBackendRotationChanges(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchSecretBackendRotationChanges", reflect.TypeOf((*MockState)(nil).WatchSecretBackendRotationChanges), arg0)
}
