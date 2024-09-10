// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/apiserver/facades/agent/uniter (interfaces: ModelConfigService,ModelInfoService,NetworkService)
//
// Generated by this command:
//
//	mockgen -package uniter_test -destination service_mock_test.go github.com/juju/juju/apiserver/facades/agent/uniter ModelConfigService,ModelInfoService,NetworkService
//

// Package uniter_test is a generated GoMock package.
package uniter_test

import (
	context "context"
	reflect "reflect"

	model "github.com/juju/juju/core/model"
	network "github.com/juju/juju/core/network"
	watcher "github.com/juju/juju/core/watcher"
	config "github.com/juju/juju/environs/config"
	gomock "go.uber.org/mock/gomock"
)

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

// ModelConfig mocks base method.
func (m *MockModelConfigService) ModelConfig(arg0 context.Context) (*config.Config, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelConfig", arg0)
	ret0, _ := ret[0].(*config.Config)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ModelConfig indicates an expected call of ModelConfig.
func (mr *MockModelConfigServiceMockRecorder) ModelConfig(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelConfig", reflect.TypeOf((*MockModelConfigService)(nil).ModelConfig), arg0)
}

// Watch mocks base method.
func (m *MockModelConfigService) Watch() (watcher.Watcher[[]string], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Watch")
	ret0, _ := ret[0].(watcher.Watcher[[]string])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Watch indicates an expected call of Watch.
func (mr *MockModelConfigServiceMockRecorder) Watch() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockModelConfigService)(nil).Watch))
}

// MockModelInfoService is a mock of ModelInfoService interface.
type MockModelInfoService struct {
	ctrl     *gomock.Controller
	recorder *MockModelInfoServiceMockRecorder
}

// MockModelInfoServiceMockRecorder is the mock recorder for MockModelInfoService.
type MockModelInfoServiceMockRecorder struct {
	mock *MockModelInfoService
}

// NewMockModelInfoService creates a new mock instance.
func NewMockModelInfoService(ctrl *gomock.Controller) *MockModelInfoService {
	mock := &MockModelInfoService{ctrl: ctrl}
	mock.recorder = &MockModelInfoServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModelInfoService) EXPECT() *MockModelInfoServiceMockRecorder {
	return m.recorder
}

// GetModelInfo mocks base method.
func (m *MockModelInfoService) GetModelInfo(arg0 context.Context) (model.ReadOnlyModel, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelInfo", arg0)
	ret0, _ := ret[0].(model.ReadOnlyModel)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelInfo indicates an expected call of GetModelInfo.
func (mr *MockModelInfoServiceMockRecorder) GetModelInfo(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelInfo", reflect.TypeOf((*MockModelInfoService)(nil).GetModelInfo), arg0)
}

// MockNetworkService is a mock of NetworkService interface.
type MockNetworkService struct {
	ctrl     *gomock.Controller
	recorder *MockNetworkServiceMockRecorder
}

// MockNetworkServiceMockRecorder is the mock recorder for MockNetworkService.
type MockNetworkServiceMockRecorder struct {
	mock *MockNetworkService
}

// NewMockNetworkService creates a new mock instance.
func NewMockNetworkService(ctrl *gomock.Controller) *MockNetworkService {
	mock := &MockNetworkService{ctrl: ctrl}
	mock.recorder = &MockNetworkServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNetworkService) EXPECT() *MockNetworkServiceMockRecorder {
	return m.recorder
}

// GetAllSubnets mocks base method.
func (m *MockNetworkService) GetAllSubnets(arg0 context.Context) (network.SubnetInfos, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllSubnets", arg0)
	ret0, _ := ret[0].(network.SubnetInfos)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAllSubnets indicates an expected call of GetAllSubnets.
func (mr *MockNetworkServiceMockRecorder) GetAllSubnets(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllSubnets", reflect.TypeOf((*MockNetworkService)(nil).GetAllSubnets), arg0)
}

// SpaceByName mocks base method.
func (m *MockNetworkService) SpaceByName(arg0 context.Context, arg1 string) (*network.SpaceInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SpaceByName", arg0, arg1)
	ret0, _ := ret[0].(*network.SpaceInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SpaceByName indicates an expected call of SpaceByName.
func (mr *MockNetworkServiceMockRecorder) SpaceByName(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SpaceByName", reflect.TypeOf((*MockNetworkService)(nil).SpaceByName), arg0, arg1)
}
