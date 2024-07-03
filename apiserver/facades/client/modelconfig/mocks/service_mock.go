// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/apiserver/facades/client/modelconfig (interfaces: SecretBackendService,ModelConfigService)
//
// Generated by this command:
//
//	mockgen -typed -package mocks -destination mocks/service_mock.go github.com/juju/juju/apiserver/facades/client/modelconfig SecretBackendService,ModelConfigService
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	model "github.com/juju/juju/core/model"
	config "github.com/juju/juju/environs/config"
	gomock "go.uber.org/mock/gomock"
)

// MockSecretBackendService is a mock of SecretBackendService interface.
type MockSecretBackendService struct {
	ctrl     *gomock.Controller
	recorder *MockSecretBackendServiceMockRecorder
}

// MockSecretBackendServiceMockRecorder is the mock recorder for MockSecretBackendService.
type MockSecretBackendServiceMockRecorder struct {
	mock *MockSecretBackendService
}

// NewMockSecretBackendService creates a new mock instance.
func NewMockSecretBackendService(ctrl *gomock.Controller) *MockSecretBackendService {
	mock := &MockSecretBackendService{ctrl: ctrl}
	mock.recorder = &MockSecretBackendServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSecretBackendService) EXPECT() *MockSecretBackendServiceMockRecorder {
	return m.recorder
}

// PingSecretBackend mocks base method.
func (m *MockSecretBackendService) PingSecretBackend(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PingSecretBackend", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// PingSecretBackend indicates an expected call of PingSecretBackend.
func (mr *MockSecretBackendServiceMockRecorder) PingSecretBackend(arg0, arg1 any) *MockSecretBackendServicePingSecretBackendCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PingSecretBackend", reflect.TypeOf((*MockSecretBackendService)(nil).PingSecretBackend), arg0, arg1)
	return &MockSecretBackendServicePingSecretBackendCall{Call: call}
}

// MockSecretBackendServicePingSecretBackendCall wrap *gomock.Call
type MockSecretBackendServicePingSecretBackendCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockSecretBackendServicePingSecretBackendCall) Return(arg0 error) *MockSecretBackendServicePingSecretBackendCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockSecretBackendServicePingSecretBackendCall) Do(f func(context.Context, string) error) *MockSecretBackendServicePingSecretBackendCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockSecretBackendServicePingSecretBackendCall) DoAndReturn(f func(context.Context, string) error) *MockSecretBackendServicePingSecretBackendCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// MockModelConfigService is a mock of ModelConfigService interface.
type MockModelConfigService struct {
	ctrl     *gomock.Controller
	recorder *MockModelConfigServiceMockRecorder
}

// MockModelConfigServiceMockRecorder is the mock recorder for MockModelConfigService.
type MockModelConfigServiceMockRecorder struct {
	mock *MockModelConfigService
}

// NewMockModelConfigService creates a new mock instance.
func NewMockModelConfigService(ctrl *gomock.Controller) *MockModelConfigService {
	mock := &MockModelConfigService{ctrl: ctrl}
	mock.recorder = &MockModelConfigServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModelConfigService) EXPECT() *MockModelConfigServiceMockRecorder {
	return m.recorder
}

// GetModelSecretBackend mocks base method.
func (m *MockModelConfigService) GetModelSecretBackend(arg0 context.Context, arg1 model.UUID) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelSecretBackend", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelSecretBackend indicates an expected call of GetModelSecretBackend.
func (mr *MockModelConfigServiceMockRecorder) GetModelSecretBackend(arg0, arg1 any) *MockModelConfigServiceGetModelSecretBackendCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelSecretBackend", reflect.TypeOf((*MockModelConfigService)(nil).GetModelSecretBackend), arg0, arg1)
	return &MockModelConfigServiceGetModelSecretBackendCall{Call: call}
}

// MockModelConfigServiceGetModelSecretBackendCall wrap *gomock.Call
type MockModelConfigServiceGetModelSecretBackendCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockModelConfigServiceGetModelSecretBackendCall) Return(arg0 string, arg1 error) *MockModelConfigServiceGetModelSecretBackendCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockModelConfigServiceGetModelSecretBackendCall) Do(f func(context.Context, model.UUID) (string, error)) *MockModelConfigServiceGetModelSecretBackendCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockModelConfigServiceGetModelSecretBackendCall) DoAndReturn(f func(context.Context, model.UUID) (string, error)) *MockModelConfigServiceGetModelSecretBackendCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// ModelConfigValues mocks base method.
func (m *MockModelConfigService) ModelConfigValues(arg0 context.Context) (config.ConfigValues, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelConfigValues", arg0)
	ret0, _ := ret[0].(config.ConfigValues)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ModelConfigValues indicates an expected call of ModelConfigValues.
func (mr *MockModelConfigServiceMockRecorder) ModelConfigValues(arg0 any) *MockModelConfigServiceModelConfigValuesCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelConfigValues", reflect.TypeOf((*MockModelConfigService)(nil).ModelConfigValues), arg0)
	return &MockModelConfigServiceModelConfigValuesCall{Call: call}
}

// MockModelConfigServiceModelConfigValuesCall wrap *gomock.Call
type MockModelConfigServiceModelConfigValuesCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockModelConfigServiceModelConfigValuesCall) Return(arg0 config.ConfigValues, arg1 error) *MockModelConfigServiceModelConfigValuesCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockModelConfigServiceModelConfigValuesCall) Do(f func(context.Context) (config.ConfigValues, error)) *MockModelConfigServiceModelConfigValuesCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockModelConfigServiceModelConfigValuesCall) DoAndReturn(f func(context.Context) (config.ConfigValues, error)) *MockModelConfigServiceModelConfigValuesCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// SetModelSecretBackend mocks base method.
func (m *MockModelConfigService) SetModelSecretBackend(arg0 context.Context, arg1 model.UUID, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetModelSecretBackend", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetModelSecretBackend indicates an expected call of SetModelSecretBackend.
func (mr *MockModelConfigServiceMockRecorder) SetModelSecretBackend(arg0, arg1, arg2 any) *MockModelConfigServiceSetModelSecretBackendCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetModelSecretBackend", reflect.TypeOf((*MockModelConfigService)(nil).SetModelSecretBackend), arg0, arg1, arg2)
	return &MockModelConfigServiceSetModelSecretBackendCall{Call: call}
}

// MockModelConfigServiceSetModelSecretBackendCall wrap *gomock.Call
type MockModelConfigServiceSetModelSecretBackendCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockModelConfigServiceSetModelSecretBackendCall) Return(arg0 error) *MockModelConfigServiceSetModelSecretBackendCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockModelConfigServiceSetModelSecretBackendCall) Do(f func(context.Context, model.UUID, string) error) *MockModelConfigServiceSetModelSecretBackendCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockModelConfigServiceSetModelSecretBackendCall) DoAndReturn(f func(context.Context, model.UUID, string) error) *MockModelConfigServiceSetModelSecretBackendCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// UpdateModelConfig mocks base method.
func (m *MockModelConfigService) UpdateModelConfig(arg0 context.Context, arg1 map[string]any, arg2 []string, arg3 ...config.Validator) error {
	m.ctrl.T.Helper()
	varargs := []any{arg0, arg1, arg2}
	for _, a := range arg3 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "UpdateModelConfig", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateModelConfig indicates an expected call of UpdateModelConfig.
func (mr *MockModelConfigServiceMockRecorder) UpdateModelConfig(arg0, arg1, arg2 any, arg3 ...any) *MockModelConfigServiceUpdateModelConfigCall {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0, arg1, arg2}, arg3...)
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateModelConfig", reflect.TypeOf((*MockModelConfigService)(nil).UpdateModelConfig), varargs...)
	return &MockModelConfigServiceUpdateModelConfigCall{Call: call}
}

// MockModelConfigServiceUpdateModelConfigCall wrap *gomock.Call
type MockModelConfigServiceUpdateModelConfigCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockModelConfigServiceUpdateModelConfigCall) Return(arg0 error) *MockModelConfigServiceUpdateModelConfigCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockModelConfigServiceUpdateModelConfigCall) Do(f func(context.Context, map[string]any, []string, ...config.Validator) error) *MockModelConfigServiceUpdateModelConfigCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockModelConfigServiceUpdateModelConfigCall) DoAndReturn(f func(context.Context, map[string]any, []string, ...config.Validator) error) *MockModelConfigServiceUpdateModelConfigCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
