// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/cmd/juju/application/bundle (interfaces: BundleDataSource)
//
// Generated by this command:
//
//	mockgen -typed -package mocks -destination mocks/bundledatasource_mock.go github.com/juju/juju/cmd/juju/application/bundle BundleDataSource
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	charm "github.com/juju/charm/v13"
	gomock "go.uber.org/mock/gomock"
)

// MockBundleDataSource is a mock of BundleDataSource interface.
type MockBundleDataSource struct {
	ctrl     *gomock.Controller
	recorder *MockBundleDataSourceMockRecorder
}

// MockBundleDataSourceMockRecorder is the mock recorder for MockBundleDataSource.
type MockBundleDataSourceMockRecorder struct {
	mock *MockBundleDataSource
}

// NewMockBundleDataSource creates a new mock instance.
func NewMockBundleDataSource(ctrl *gomock.Controller) *MockBundleDataSource {
	mock := &MockBundleDataSource{ctrl: ctrl}
	mock.recorder = &MockBundleDataSourceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBundleDataSource) EXPECT() *MockBundleDataSourceMockRecorder {
	return m.recorder
}

// BasePath mocks base method.
func (m *MockBundleDataSource) BasePath() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BasePath")
	ret0, _ := ret[0].(string)
	return ret0
}

// BasePath indicates an expected call of BasePath.
func (mr *MockBundleDataSourceMockRecorder) BasePath() *MockBundleDataSourceBasePathCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BasePath", reflect.TypeOf((*MockBundleDataSource)(nil).BasePath))
	return &MockBundleDataSourceBasePathCall{Call: call}
}

// MockBundleDataSourceBasePathCall wrap *gomock.Call
type MockBundleDataSourceBasePathCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockBundleDataSourceBasePathCall) Return(arg0 string) *MockBundleDataSourceBasePathCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockBundleDataSourceBasePathCall) Do(f func() string) *MockBundleDataSourceBasePathCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockBundleDataSourceBasePathCall) DoAndReturn(f func() string) *MockBundleDataSourceBasePathCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// BundleBytes mocks base method.
func (m *MockBundleDataSource) BundleBytes() []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BundleBytes")
	ret0, _ := ret[0].([]byte)
	return ret0
}

// BundleBytes indicates an expected call of BundleBytes.
func (mr *MockBundleDataSourceMockRecorder) BundleBytes() *MockBundleDataSourceBundleBytesCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BundleBytes", reflect.TypeOf((*MockBundleDataSource)(nil).BundleBytes))
	return &MockBundleDataSourceBundleBytesCall{Call: call}
}

// MockBundleDataSourceBundleBytesCall wrap *gomock.Call
type MockBundleDataSourceBundleBytesCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockBundleDataSourceBundleBytesCall) Return(arg0 []byte) *MockBundleDataSourceBundleBytesCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockBundleDataSourceBundleBytesCall) Do(f func() []byte) *MockBundleDataSourceBundleBytesCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockBundleDataSourceBundleBytesCall) DoAndReturn(f func() []byte) *MockBundleDataSourceBundleBytesCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Parts mocks base method.
func (m *MockBundleDataSource) Parts() []*charm.BundleDataPart {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Parts")
	ret0, _ := ret[0].([]*charm.BundleDataPart)
	return ret0
}

// Parts indicates an expected call of Parts.
func (mr *MockBundleDataSourceMockRecorder) Parts() *MockBundleDataSourcePartsCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Parts", reflect.TypeOf((*MockBundleDataSource)(nil).Parts))
	return &MockBundleDataSourcePartsCall{Call: call}
}

// MockBundleDataSourcePartsCall wrap *gomock.Call
type MockBundleDataSourcePartsCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockBundleDataSourcePartsCall) Return(arg0 []*charm.BundleDataPart) *MockBundleDataSourcePartsCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockBundleDataSourcePartsCall) Do(f func() []*charm.BundleDataPart) *MockBundleDataSourcePartsCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockBundleDataSourcePartsCall) DoAndReturn(f func() []*charm.BundleDataPart) *MockBundleDataSourcePartsCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// ResolveInclude mocks base method.
func (m *MockBundleDataSource) ResolveInclude(arg0 string) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResolveInclude", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResolveInclude indicates an expected call of ResolveInclude.
func (mr *MockBundleDataSourceMockRecorder) ResolveInclude(arg0 any) *MockBundleDataSourceResolveIncludeCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResolveInclude", reflect.TypeOf((*MockBundleDataSource)(nil).ResolveInclude), arg0)
	return &MockBundleDataSourceResolveIncludeCall{Call: call}
}

// MockBundleDataSourceResolveIncludeCall wrap *gomock.Call
type MockBundleDataSourceResolveIncludeCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockBundleDataSourceResolveIncludeCall) Return(arg0 []byte, arg1 error) *MockBundleDataSourceResolveIncludeCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockBundleDataSourceResolveIncludeCall) Do(f func(string) ([]byte, error)) *MockBundleDataSourceResolveIncludeCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockBundleDataSourceResolveIncludeCall) DoAndReturn(f func(string) ([]byte, error)) *MockBundleDataSourceResolveIncludeCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
