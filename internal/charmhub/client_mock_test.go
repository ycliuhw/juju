// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/internal/charmhub (interfaces: HTTPClient,RESTClient,FileSystem)
//
// Generated by this command:
//
//	mockgen -typed -package charmhub -destination client_mock_test.go github.com/juju/juju/internal/charmhub HTTPClient,RESTClient,FileSystem
//

// Package charmhub is a generated GoMock package.
package charmhub

import (
	context "context"
	http "net/http"
	os "os"
	reflect "reflect"

	path "github.com/juju/juju/internal/charmhub/path"
	gomock "go.uber.org/mock/gomock"
)

// MockHTTPClient is a mock of HTTPClient interface.
type MockHTTPClient struct {
	ctrl     *gomock.Controller
	recorder *MockHTTPClientMockRecorder
}

// MockHTTPClientMockRecorder is the mock recorder for MockHTTPClient.
type MockHTTPClientMockRecorder struct {
	mock *MockHTTPClient
}

// NewMockHTTPClient creates a new mock instance.
func NewMockHTTPClient(ctrl *gomock.Controller) *MockHTTPClient {
	mock := &MockHTTPClient{ctrl: ctrl}
	mock.recorder = &MockHTTPClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockHTTPClient) EXPECT() *MockHTTPClientMockRecorder {
	return m.recorder
}

// Do mocks base method.
func (m *MockHTTPClient) Do(arg0 *http.Request) (*http.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Do", arg0)
	ret0, _ := ret[0].(*http.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Do indicates an expected call of Do.
func (mr *MockHTTPClientMockRecorder) Do(arg0 any) *MockHTTPClientDoCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Do", reflect.TypeOf((*MockHTTPClient)(nil).Do), arg0)
	return &MockHTTPClientDoCall{Call: call}
}

// MockHTTPClientDoCall wrap *gomock.Call
type MockHTTPClientDoCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockHTTPClientDoCall) Return(arg0 *http.Response, arg1 error) *MockHTTPClientDoCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockHTTPClientDoCall) Do(f func(*http.Request) (*http.Response, error)) *MockHTTPClientDoCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockHTTPClientDoCall) DoAndReturn(f func(*http.Request) (*http.Response, error)) *MockHTTPClientDoCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// MockRESTClient is a mock of RESTClient interface.
type MockRESTClient struct {
	ctrl     *gomock.Controller
	recorder *MockRESTClientMockRecorder
}

// MockRESTClientMockRecorder is the mock recorder for MockRESTClient.
type MockRESTClientMockRecorder struct {
	mock *MockRESTClient
}

// NewMockRESTClient creates a new mock instance.
func NewMockRESTClient(ctrl *gomock.Controller) *MockRESTClient {
	mock := &MockRESTClient{ctrl: ctrl}
	mock.recorder = &MockRESTClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRESTClient) EXPECT() *MockRESTClientMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockRESTClient) Get(arg0 context.Context, arg1 path.Path, arg2 any) (restResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1, arg2)
	ret0, _ := ret[0].(restResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockRESTClientMockRecorder) Get(arg0, arg1, arg2 any) *MockRESTClientGetCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockRESTClient)(nil).Get), arg0, arg1, arg2)
	return &MockRESTClientGetCall{Call: call}
}

// MockRESTClientGetCall wrap *gomock.Call
type MockRESTClientGetCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockRESTClientGetCall) Return(arg0 restResponse, arg1 error) *MockRESTClientGetCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockRESTClientGetCall) Do(f func(context.Context, path.Path, any) (restResponse, error)) *MockRESTClientGetCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockRESTClientGetCall) DoAndReturn(f func(context.Context, path.Path, any) (restResponse, error)) *MockRESTClientGetCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Post mocks base method.
func (m *MockRESTClient) Post(arg0 context.Context, arg1 path.Path, arg2 http.Header, arg3, arg4 any) (restResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Post", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(restResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Post indicates an expected call of Post.
func (mr *MockRESTClientMockRecorder) Post(arg0, arg1, arg2, arg3, arg4 any) *MockRESTClientPostCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Post", reflect.TypeOf((*MockRESTClient)(nil).Post), arg0, arg1, arg2, arg3, arg4)
	return &MockRESTClientPostCall{Call: call}
}

// MockRESTClientPostCall wrap *gomock.Call
type MockRESTClientPostCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockRESTClientPostCall) Return(arg0 restResponse, arg1 error) *MockRESTClientPostCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockRESTClientPostCall) Do(f func(context.Context, path.Path, http.Header, any, any) (restResponse, error)) *MockRESTClientPostCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockRESTClientPostCall) DoAndReturn(f func(context.Context, path.Path, http.Header, any, any) (restResponse, error)) *MockRESTClientPostCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// MockFileSystem is a mock of FileSystem interface.
type MockFileSystem struct {
	ctrl     *gomock.Controller
	recorder *MockFileSystemMockRecorder
}

// MockFileSystemMockRecorder is the mock recorder for MockFileSystem.
type MockFileSystemMockRecorder struct {
	mock *MockFileSystem
}

// NewMockFileSystem creates a new mock instance.
func NewMockFileSystem(ctrl *gomock.Controller) *MockFileSystem {
	mock := &MockFileSystem{ctrl: ctrl}
	mock.recorder = &MockFileSystemMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFileSystem) EXPECT() *MockFileSystemMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockFileSystem) Create(arg0 string) (*os.File, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0)
	ret0, _ := ret[0].(*os.File)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockFileSystemMockRecorder) Create(arg0 any) *MockFileSystemCreateCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockFileSystem)(nil).Create), arg0)
	return &MockFileSystemCreateCall{Call: call}
}

// MockFileSystemCreateCall wrap *gomock.Call
type MockFileSystemCreateCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockFileSystemCreateCall) Return(arg0 *os.File, arg1 error) *MockFileSystemCreateCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockFileSystemCreateCall) Do(f func(string) (*os.File, error)) *MockFileSystemCreateCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockFileSystemCreateCall) DoAndReturn(f func(string) (*os.File, error)) *MockFileSystemCreateCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
