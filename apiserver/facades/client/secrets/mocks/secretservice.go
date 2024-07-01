// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/apiserver/facades/client/secrets (interfaces: SecretService,SecretBackendService)
//
// Generated by this command:
//
//	mockgen -typed -package mocks -destination mocks/secretservice.go github.com/juju/juju/apiserver/facades/client/secrets SecretService,SecretBackendService
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	model "github.com/juju/juju/core/model"
	secrets "github.com/juju/juju/core/secrets"
	secret "github.com/juju/juju/domain/secret"
	service "github.com/juju/juju/domain/secret/service"
	provider "github.com/juju/juju/internal/secrets/provider"
	gomock "go.uber.org/mock/gomock"
)

// MockSecretService is a mock of SecretService interface.
type MockSecretService struct {
	ctrl     *gomock.Controller
	recorder *MockSecretServiceMockRecorder
}

// MockSecretServiceMockRecorder is the mock recorder for MockSecretService.
type MockSecretServiceMockRecorder struct {
	mock *MockSecretService
}

// NewMockSecretService creates a new mock instance.
func NewMockSecretService(ctrl *gomock.Controller) *MockSecretService {
	mock := &MockSecretService{ctrl: ctrl}
	mock.recorder = &MockSecretServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSecretService) EXPECT() *MockSecretServiceMockRecorder {
	return m.recorder
}

// CreateUserSecret mocks base method.
func (m *MockSecretService) CreateUserSecret(arg0 context.Context, arg1 *secrets.URI, arg2 service.CreateUserSecretParams) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUserSecret", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateUserSecret indicates an expected call of CreateUserSecret.
func (mr *MockSecretServiceMockRecorder) CreateUserSecret(arg0, arg1, arg2 any) *MockSecretServiceCreateUserSecretCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUserSecret", reflect.TypeOf((*MockSecretService)(nil).CreateUserSecret), arg0, arg1, arg2)
	return &MockSecretServiceCreateUserSecretCall{Call: call}
}

// MockSecretServiceCreateUserSecretCall wrap *gomock.Call
type MockSecretServiceCreateUserSecretCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockSecretServiceCreateUserSecretCall) Return(arg0 error) *MockSecretServiceCreateUserSecretCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockSecretServiceCreateUserSecretCall) Do(f func(context.Context, *secrets.URI, service.CreateUserSecretParams) error) *MockSecretServiceCreateUserSecretCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockSecretServiceCreateUserSecretCall) DoAndReturn(f func(context.Context, *secrets.URI, service.CreateUserSecretParams) error) *MockSecretServiceCreateUserSecretCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// DeleteSecret mocks base method.
func (m *MockSecretService) DeleteSecret(arg0 context.Context, arg1 *secrets.URI, arg2 service.DeleteSecretParams) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteSecret", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteSecret indicates an expected call of DeleteSecret.
func (mr *MockSecretServiceMockRecorder) DeleteSecret(arg0, arg1, arg2 any) *MockSecretServiceDeleteSecretCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteSecret", reflect.TypeOf((*MockSecretService)(nil).DeleteSecret), arg0, arg1, arg2)
	return &MockSecretServiceDeleteSecretCall{Call: call}
}

// MockSecretServiceDeleteSecretCall wrap *gomock.Call
type MockSecretServiceDeleteSecretCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockSecretServiceDeleteSecretCall) Return(arg0 error) *MockSecretServiceDeleteSecretCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockSecretServiceDeleteSecretCall) Do(f func(context.Context, *secrets.URI, service.DeleteSecretParams) error) *MockSecretServiceDeleteSecretCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockSecretServiceDeleteSecretCall) DoAndReturn(f func(context.Context, *secrets.URI, service.DeleteSecretParams) error) *MockSecretServiceDeleteSecretCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// GetSecretContentFromBackend mocks base method.
func (m *MockSecretService) GetSecretContentFromBackend(arg0 context.Context, arg1 *secrets.URI, arg2 int) (secrets.SecretValue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSecretContentFromBackend", arg0, arg1, arg2)
	ret0, _ := ret[0].(secrets.SecretValue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSecretContentFromBackend indicates an expected call of GetSecretContentFromBackend.
func (mr *MockSecretServiceMockRecorder) GetSecretContentFromBackend(arg0, arg1, arg2 any) *MockSecretServiceGetSecretContentFromBackendCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSecretContentFromBackend", reflect.TypeOf((*MockSecretService)(nil).GetSecretContentFromBackend), arg0, arg1, arg2)
	return &MockSecretServiceGetSecretContentFromBackendCall{Call: call}
}

// MockSecretServiceGetSecretContentFromBackendCall wrap *gomock.Call
type MockSecretServiceGetSecretContentFromBackendCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockSecretServiceGetSecretContentFromBackendCall) Return(arg0 secrets.SecretValue, arg1 error) *MockSecretServiceGetSecretContentFromBackendCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockSecretServiceGetSecretContentFromBackendCall) Do(f func(context.Context, *secrets.URI, int) (secrets.SecretValue, error)) *MockSecretServiceGetSecretContentFromBackendCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockSecretServiceGetSecretContentFromBackendCall) DoAndReturn(f func(context.Context, *secrets.URI, int) (secrets.SecretValue, error)) *MockSecretServiceGetSecretContentFromBackendCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// GetSecretGrants mocks base method.
func (m *MockSecretService) GetSecretGrants(arg0 context.Context, arg1 *secrets.URI, arg2 secrets.SecretRole) ([]service.SecretAccess, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSecretGrants", arg0, arg1, arg2)
	ret0, _ := ret[0].([]service.SecretAccess)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSecretGrants indicates an expected call of GetSecretGrants.
func (mr *MockSecretServiceMockRecorder) GetSecretGrants(arg0, arg1, arg2 any) *MockSecretServiceGetSecretGrantsCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSecretGrants", reflect.TypeOf((*MockSecretService)(nil).GetSecretGrants), arg0, arg1, arg2)
	return &MockSecretServiceGetSecretGrantsCall{Call: call}
}

// MockSecretServiceGetSecretGrantsCall wrap *gomock.Call
type MockSecretServiceGetSecretGrantsCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockSecretServiceGetSecretGrantsCall) Return(arg0 []service.SecretAccess, arg1 error) *MockSecretServiceGetSecretGrantsCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockSecretServiceGetSecretGrantsCall) Do(f func(context.Context, *secrets.URI, secrets.SecretRole) ([]service.SecretAccess, error)) *MockSecretServiceGetSecretGrantsCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockSecretServiceGetSecretGrantsCall) DoAndReturn(f func(context.Context, *secrets.URI, secrets.SecretRole) ([]service.SecretAccess, error)) *MockSecretServiceGetSecretGrantsCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// GetUserSecretURIByLabel mocks base method.
func (m *MockSecretService) GetUserSecretURIByLabel(arg0 context.Context, arg1 string) (*secrets.URI, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserSecretURIByLabel", arg0, arg1)
	ret0, _ := ret[0].(*secrets.URI)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserSecretURIByLabel indicates an expected call of GetUserSecretURIByLabel.
func (mr *MockSecretServiceMockRecorder) GetUserSecretURIByLabel(arg0, arg1 any) *MockSecretServiceGetUserSecretURIByLabelCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserSecretURIByLabel", reflect.TypeOf((*MockSecretService)(nil).GetUserSecretURIByLabel), arg0, arg1)
	return &MockSecretServiceGetUserSecretURIByLabelCall{Call: call}
}

// MockSecretServiceGetUserSecretURIByLabelCall wrap *gomock.Call
type MockSecretServiceGetUserSecretURIByLabelCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockSecretServiceGetUserSecretURIByLabelCall) Return(arg0 *secrets.URI, arg1 error) *MockSecretServiceGetUserSecretURIByLabelCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockSecretServiceGetUserSecretURIByLabelCall) Do(f func(context.Context, string) (*secrets.URI, error)) *MockSecretServiceGetUserSecretURIByLabelCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockSecretServiceGetUserSecretURIByLabelCall) DoAndReturn(f func(context.Context, string) (*secrets.URI, error)) *MockSecretServiceGetUserSecretURIByLabelCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// GrantSecretAccess mocks base method.
func (m *MockSecretService) GrantSecretAccess(arg0 context.Context, arg1 *secrets.URI, arg2 service.SecretAccessParams) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GrantSecretAccess", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// GrantSecretAccess indicates an expected call of GrantSecretAccess.
func (mr *MockSecretServiceMockRecorder) GrantSecretAccess(arg0, arg1, arg2 any) *MockSecretServiceGrantSecretAccessCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GrantSecretAccess", reflect.TypeOf((*MockSecretService)(nil).GrantSecretAccess), arg0, arg1, arg2)
	return &MockSecretServiceGrantSecretAccessCall{Call: call}
}

// MockSecretServiceGrantSecretAccessCall wrap *gomock.Call
type MockSecretServiceGrantSecretAccessCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockSecretServiceGrantSecretAccessCall) Return(arg0 error) *MockSecretServiceGrantSecretAccessCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockSecretServiceGrantSecretAccessCall) Do(f func(context.Context, *secrets.URI, service.SecretAccessParams) error) *MockSecretServiceGrantSecretAccessCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockSecretServiceGrantSecretAccessCall) DoAndReturn(f func(context.Context, *secrets.URI, service.SecretAccessParams) error) *MockSecretServiceGrantSecretAccessCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// ListCharmSecrets mocks base method.
func (m *MockSecretService) ListCharmSecrets(arg0 context.Context, arg1 ...service.CharmSecretOwner) ([]*secrets.SecretMetadata, [][]*secrets.SecretRevisionMetadata, error) {
	m.ctrl.T.Helper()
	varargs := []any{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListCharmSecrets", varargs...)
	ret0, _ := ret[0].([]*secrets.SecretMetadata)
	ret1, _ := ret[1].([][]*secrets.SecretRevisionMetadata)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ListCharmSecrets indicates an expected call of ListCharmSecrets.
func (mr *MockSecretServiceMockRecorder) ListCharmSecrets(arg0 any, arg1 ...any) *MockSecretServiceListCharmSecretsCall {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0}, arg1...)
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCharmSecrets", reflect.TypeOf((*MockSecretService)(nil).ListCharmSecrets), varargs...)
	return &MockSecretServiceListCharmSecretsCall{Call: call}
}

// MockSecretServiceListCharmSecretsCall wrap *gomock.Call
type MockSecretServiceListCharmSecretsCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockSecretServiceListCharmSecretsCall) Return(arg0 []*secrets.SecretMetadata, arg1 [][]*secrets.SecretRevisionMetadata, arg2 error) *MockSecretServiceListCharmSecretsCall {
	c.Call = c.Call.Return(arg0, arg1, arg2)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockSecretServiceListCharmSecretsCall) Do(f func(context.Context, ...service.CharmSecretOwner) ([]*secrets.SecretMetadata, [][]*secrets.SecretRevisionMetadata, error)) *MockSecretServiceListCharmSecretsCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockSecretServiceListCharmSecretsCall) DoAndReturn(f func(context.Context, ...service.CharmSecretOwner) ([]*secrets.SecretMetadata, [][]*secrets.SecretRevisionMetadata, error)) *MockSecretServiceListCharmSecretsCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// ListSecrets mocks base method.
func (m *MockSecretService) ListSecrets(arg0 context.Context, arg1 *secrets.URI, arg2 *int, arg3 secret.Labels) ([]*secrets.SecretMetadata, [][]*secrets.SecretRevisionMetadata, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListSecrets", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([]*secrets.SecretMetadata)
	ret1, _ := ret[1].([][]*secrets.SecretRevisionMetadata)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ListSecrets indicates an expected call of ListSecrets.
func (mr *MockSecretServiceMockRecorder) ListSecrets(arg0, arg1, arg2, arg3 any) *MockSecretServiceListSecretsCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListSecrets", reflect.TypeOf((*MockSecretService)(nil).ListSecrets), arg0, arg1, arg2, arg3)
	return &MockSecretServiceListSecretsCall{Call: call}
}

// MockSecretServiceListSecretsCall wrap *gomock.Call
type MockSecretServiceListSecretsCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockSecretServiceListSecretsCall) Return(arg0 []*secrets.SecretMetadata, arg1 [][]*secrets.SecretRevisionMetadata, arg2 error) *MockSecretServiceListSecretsCall {
	c.Call = c.Call.Return(arg0, arg1, arg2)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockSecretServiceListSecretsCall) Do(f func(context.Context, *secrets.URI, *int, secret.Labels) ([]*secrets.SecretMetadata, [][]*secrets.SecretRevisionMetadata, error)) *MockSecretServiceListSecretsCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockSecretServiceListSecretsCall) DoAndReturn(f func(context.Context, *secrets.URI, *int, secret.Labels) ([]*secrets.SecretMetadata, [][]*secrets.SecretRevisionMetadata, error)) *MockSecretServiceListSecretsCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// RevokeSecretAccess mocks base method.
func (m *MockSecretService) RevokeSecretAccess(arg0 context.Context, arg1 *secrets.URI, arg2 service.SecretAccessParams) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RevokeSecretAccess", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// RevokeSecretAccess indicates an expected call of RevokeSecretAccess.
func (mr *MockSecretServiceMockRecorder) RevokeSecretAccess(arg0, arg1, arg2 any) *MockSecretServiceRevokeSecretAccessCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RevokeSecretAccess", reflect.TypeOf((*MockSecretService)(nil).RevokeSecretAccess), arg0, arg1, arg2)
	return &MockSecretServiceRevokeSecretAccessCall{Call: call}
}

// MockSecretServiceRevokeSecretAccessCall wrap *gomock.Call
type MockSecretServiceRevokeSecretAccessCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockSecretServiceRevokeSecretAccessCall) Return(arg0 error) *MockSecretServiceRevokeSecretAccessCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockSecretServiceRevokeSecretAccessCall) Do(f func(context.Context, *secrets.URI, service.SecretAccessParams) error) *MockSecretServiceRevokeSecretAccessCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockSecretServiceRevokeSecretAccessCall) DoAndReturn(f func(context.Context, *secrets.URI, service.SecretAccessParams) error) *MockSecretServiceRevokeSecretAccessCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// UpdateUserSecret mocks base method.
func (m *MockSecretService) UpdateUserSecret(arg0 context.Context, arg1 *secrets.URI, arg2 service.UpdateUserSecretParams) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUserSecret", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateUserSecret indicates an expected call of UpdateUserSecret.
func (mr *MockSecretServiceMockRecorder) UpdateUserSecret(arg0, arg1, arg2 any) *MockSecretServiceUpdateUserSecretCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUserSecret", reflect.TypeOf((*MockSecretService)(nil).UpdateUserSecret), arg0, arg1, arg2)
	return &MockSecretServiceUpdateUserSecretCall{Call: call}
}

// MockSecretServiceUpdateUserSecretCall wrap *gomock.Call
type MockSecretServiceUpdateUserSecretCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockSecretServiceUpdateUserSecretCall) Return(arg0 error) *MockSecretServiceUpdateUserSecretCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockSecretServiceUpdateUserSecretCall) Do(f func(context.Context, *secrets.URI, service.UpdateUserSecretParams) error) *MockSecretServiceUpdateUserSecretCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockSecretServiceUpdateUserSecretCall) DoAndReturn(f func(context.Context, *secrets.URI, service.UpdateUserSecretParams) error) *MockSecretServiceUpdateUserSecretCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

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

// GetModelSecretBackend mocks base method.
func (m *MockSecretBackendService) GetModelSecretBackend(arg0 context.Context, arg1 model.UUID) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelSecretBackend", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelSecretBackend indicates an expected call of GetModelSecretBackend.
func (mr *MockSecretBackendServiceMockRecorder) GetModelSecretBackend(arg0, arg1 any) *MockSecretBackendServiceGetModelSecretBackendCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelSecretBackend", reflect.TypeOf((*MockSecretBackendService)(nil).GetModelSecretBackend), arg0, arg1)
	return &MockSecretBackendServiceGetModelSecretBackendCall{Call: call}
}

// MockSecretBackendServiceGetModelSecretBackendCall wrap *gomock.Call
type MockSecretBackendServiceGetModelSecretBackendCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockSecretBackendServiceGetModelSecretBackendCall) Return(arg0 string, arg1 error) *MockSecretBackendServiceGetModelSecretBackendCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockSecretBackendServiceGetModelSecretBackendCall) Do(f func(context.Context, model.UUID) (string, error)) *MockSecretBackendServiceGetModelSecretBackendCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockSecretBackendServiceGetModelSecretBackendCall) DoAndReturn(f func(context.Context, model.UUID) (string, error)) *MockSecretBackendServiceGetModelSecretBackendCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// GetSecretBackendConfigForAdmin mocks base method.
func (m *MockSecretBackendService) GetSecretBackendConfigForAdmin(arg0 context.Context, arg1 model.UUID) (*provider.ModelBackendConfigInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSecretBackendConfigForAdmin", arg0, arg1)
	ret0, _ := ret[0].(*provider.ModelBackendConfigInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSecretBackendConfigForAdmin indicates an expected call of GetSecretBackendConfigForAdmin.
func (mr *MockSecretBackendServiceMockRecorder) GetSecretBackendConfigForAdmin(arg0, arg1 any) *MockSecretBackendServiceGetSecretBackendConfigForAdminCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSecretBackendConfigForAdmin", reflect.TypeOf((*MockSecretBackendService)(nil).GetSecretBackendConfigForAdmin), arg0, arg1)
	return &MockSecretBackendServiceGetSecretBackendConfigForAdminCall{Call: call}
}

// MockSecretBackendServiceGetSecretBackendConfigForAdminCall wrap *gomock.Call
type MockSecretBackendServiceGetSecretBackendConfigForAdminCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockSecretBackendServiceGetSecretBackendConfigForAdminCall) Return(arg0 *provider.ModelBackendConfigInfo, arg1 error) *MockSecretBackendServiceGetSecretBackendConfigForAdminCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockSecretBackendServiceGetSecretBackendConfigForAdminCall) Do(f func(context.Context, model.UUID) (*provider.ModelBackendConfigInfo, error)) *MockSecretBackendServiceGetSecretBackendConfigForAdminCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockSecretBackendServiceGetSecretBackendConfigForAdminCall) DoAndReturn(f func(context.Context, model.UUID) (*provider.ModelBackendConfigInfo, error)) *MockSecretBackendServiceGetSecretBackendConfigForAdminCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// SetModelSecretBackend mocks base method.
func (m *MockSecretBackendService) SetModelSecretBackend(arg0 context.Context, arg1 model.UUID, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetModelSecretBackend", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetModelSecretBackend indicates an expected call of SetModelSecretBackend.
func (mr *MockSecretBackendServiceMockRecorder) SetModelSecretBackend(arg0, arg1, arg2 any) *MockSecretBackendServiceSetModelSecretBackendCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetModelSecretBackend", reflect.TypeOf((*MockSecretBackendService)(nil).SetModelSecretBackend), arg0, arg1, arg2)
	return &MockSecretBackendServiceSetModelSecretBackendCall{Call: call}
}

// MockSecretBackendServiceSetModelSecretBackendCall wrap *gomock.Call
type MockSecretBackendServiceSetModelSecretBackendCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockSecretBackendServiceSetModelSecretBackendCall) Return(arg0 error) *MockSecretBackendServiceSetModelSecretBackendCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockSecretBackendServiceSetModelSecretBackendCall) Do(f func(context.Context, model.UUID, string) error) *MockSecretBackendServiceSetModelSecretBackendCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockSecretBackendServiceSetModelSecretBackendCall) DoAndReturn(f func(context.Context, model.UUID, string) error) *MockSecretBackendServiceSetModelSecretBackendCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
