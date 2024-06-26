// Copyright 2021 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package secretsmanager_test

import (
	"time"

	"github.com/juju/names/v4"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/api/agent/secretsmanager"
	"github.com/juju/juju/api/base/testing"
	"github.com/juju/juju/core/secrets"
	"github.com/juju/juju/rpc/params"
	coretesting "github.com/juju/juju/testing"
)

var _ = gc.Suite(&SecretsSuite{})

type SecretsSuite struct {
	coretesting.BaseSuite
}

func (s *SecretsSuite) TestNewClient(c *gc.C) {
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		return nil
	})
	client := secretsmanager.NewClient(apiCaller)
	c.Assert(client, gc.NotNil)
}

func ptr[T any](v T) *T {
	return &v
}

func (s *SecretsSuite) TestCreateSecret(c *gc.C) {
	data := map[string]string{"foo": "bar"}
	expiry := time.Now()
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		c.Check(objType, gc.Equals, "SecretsManager")
		c.Check(version, gc.Equals, 0)
		c.Check(id, gc.Equals, "")
		c.Check(request, gc.Equals, "CreateSecrets")
		c.Check(arg, jc.DeepEquals, params.CreateSecretArgs{
			Args: []params.CreateSecretArg{{
				OwnerTag: "application-mysql",
				UpsertSecretArg: params.UpsertSecretArg{
					RotatePolicy: ptr(secrets.RotateDaily),
					ExpireTime:   ptr(expiry),
					Description:  ptr("my secret"),
					Label:        ptr("foo"),
					Params: map[string]interface{}{
						"password-length":        10,
						"password-special-chars": true,
					},
					Data: data,
				},
			}},
		})
		c.Assert(result, gc.FitsTypeOf, &params.StringResults{})
		*(result.(*params.StringResults)) = params.StringResults{
			[]params.StringResult{{
				Result: "secret:9m4e2mr0ui3e8a215n4g",
			}},
		}
		return nil
	})
	client := secretsmanager.NewClient(apiCaller)
	value := secrets.NewSecretValue(data)
	cfg := &secrets.SecretConfig{
		RotatePolicy: ptr(secrets.RotateDaily),
		ExpireTime:   ptr(expiry),
		Description:  ptr("my secret"),
		Label:        ptr("foo"),
		Params: map[string]interface{}{
			"password-length":        10,
			"password-special-chars": true,
		},
	}
	result, err := client.Create(cfg, names.NewApplicationTag("mysql"), value)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(result, gc.Equals, "secret:9m4e2mr0ui3e8a215n4g")
}

func (s *SecretsSuite) TestCreateSecretsError(c *gc.C) {
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		*(result.(*params.StringResults)) = params.StringResults{
			[]params.StringResult{{
				Error: &params.Error{Message: "boom"},
			}},
		}
		return nil
	})
	client := secretsmanager.NewClient(apiCaller)
	value := secrets.NewSecretValue(nil)
	result, err := client.Create(&secrets.SecretConfig{}, names.NewApplicationTag("mysql"), value)
	c.Assert(err, gc.ErrorMatches, "boom")
	c.Assert(result, gc.Equals, "")
}

func (s *SecretsSuite) TestUpdateSecret(c *gc.C) {
	data := map[string]string{"foo": "bar"}
	expiry := time.Now()
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		c.Check(objType, gc.Equals, "SecretsManager")
		c.Check(version, gc.Equals, 0)
		c.Check(id, gc.Equals, "")
		c.Check(request, gc.Equals, "UpdateSecrets")
		c.Check(arg, jc.DeepEquals, params.UpdateSecretArgs{
			Args: []params.UpdateSecretArg{{
				URI: "secret:9m4e2mr0ui3e8a215n4g",
				UpsertSecretArg: params.UpsertSecretArg{
					RotatePolicy: ptr(secrets.RotateDaily),
					ExpireTime:   ptr(expiry),
					Description:  ptr("my secret"),
					Label:        ptr("foo"),
					Params: map[string]interface{}{
						"password-length":        10,
						"password-special-chars": true,
					},
					Data: data,
				},
			}},
		})
		c.Assert(result, gc.FitsTypeOf, &params.ErrorResults{})
		*(result.(*params.ErrorResults)) = params.ErrorResults{
			[]params.ErrorResult{{}},
		}
		return nil
	})
	client := secretsmanager.NewClient(apiCaller)
	value := secrets.NewSecretValue(data)
	cfg := &secrets.SecretConfig{
		RotatePolicy: ptr(secrets.RotateDaily),
		ExpireTime:   ptr(expiry),
		Description:  ptr("my secret"),
		Label:        ptr("foo"),
		Params: map[string]interface{}{
			"password-length":        10,
			"password-special-chars": true,
		},
	}
	err := client.Update("secret:9m4e2mr0ui3e8a215n4g", cfg, value)
	c.Assert(err, jc.ErrorIsNil)
}

func (s *SecretsSuite) TestUpdateSecretsError(c *gc.C) {
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		*(result.(*params.ErrorResults)) = params.ErrorResults{
			[]params.ErrorResult{{
				Error: &params.Error{Message: "boom"},
			}},
		}
		return nil
	})
	client := secretsmanager.NewClient(apiCaller)
	value := secrets.NewSecretValue(nil)
	err := client.Update("secret:9m4e2mr0ui3e8a215n4g", &secrets.SecretConfig{}, value)
	c.Assert(err, gc.ErrorMatches, "boom")
}

func (s *SecretsSuite) TestRemoveSecret(c *gc.C) {
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		c.Check(objType, gc.Equals, "SecretsManager")
		c.Check(version, gc.Equals, 0)
		c.Check(id, gc.Equals, "")
		c.Check(request, gc.Equals, "RemoveSecrets")
		c.Check(arg, jc.DeepEquals, params.SecretURIArgs{
			Args: []params.SecretURIArg{{
				URI: "secret:9m4e2mr0ui3e8a215n4g",
			}},
		})
		c.Assert(result, gc.FitsTypeOf, &params.ErrorResults{})
		*(result.(*params.ErrorResults)) = params.ErrorResults{
			[]params.ErrorResult{{}},
		}
		return nil
	})
	client := secretsmanager.NewClient(apiCaller)
	err := client.Remove("secret:9m4e2mr0ui3e8a215n4g")
	c.Assert(err, jc.ErrorIsNil)
}

func (s *SecretsSuite) TestRemoveSecretsError(c *gc.C) {
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		*(result.(*params.ErrorResults)) = params.ErrorResults{
			[]params.ErrorResult{{
				Error: &params.Error{Message: "boom"},
			}},
		}
		return nil
	})
	client := secretsmanager.NewClient(apiCaller)
	err := client.Remove("secret:9m4e2mr0ui3e8a215n4g")
	c.Assert(err, gc.ErrorMatches, "boom")
}

func (s *SecretsSuite) TestGetSecret(c *gc.C) {
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		c.Check(objType, gc.Equals, "SecretsManager")
		c.Check(version, gc.Equals, 0)
		c.Check(id, gc.Equals, "")
		c.Check(request, gc.Equals, "GetSecretValues")
		c.Check(arg, jc.DeepEquals, params.GetSecretValueArgs{
			Args: []params.GetSecretValueArg{{
				URI:    "secret:9m4e2mr0ui3e8a215n4g",
				Label:  "label",
				Update: true,
				Peek:   true,
			}},
		})
		c.Assert(result, gc.FitsTypeOf, &params.SecretValueResults{})
		*(result.(*params.SecretValueResults)) = params.SecretValueResults{
			[]params.SecretValueResult{{
				Data: map[string]string{"foo": "bar"},
			}},
		}
		return nil
	})
	client := secretsmanager.NewClient(apiCaller)
	result, err := client.GetValue("secret:9m4e2mr0ui3e8a215n4g", "label", true, true)
	c.Assert(err, jc.ErrorIsNil)
	value := secrets.NewSecretValue(map[string]string{"foo": "bar"})
	c.Assert(result, jc.DeepEquals, value)
}

func (s *SecretsSuite) TestGetSecretsError(c *gc.C) {
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		*(result.(*params.SecretValueResults)) = params.SecretValueResults{
			[]params.SecretValueResult{{
				Error: &params.Error{Message: "boom"},
			}},
		}
		return nil
	})
	client := secretsmanager.NewClient(apiCaller)
	result, err := client.GetValue("secret:9m4e2mr0ui3e8a215n4g", "", true, true)
	c.Assert(err, gc.ErrorMatches, "boom")
	c.Assert(result, gc.IsNil)
}

func (s *SecretsSuite) TestWatchSecretsChanges(c *gc.C) {
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		c.Check(objType, gc.Equals, "SecretsManager")
		c.Check(version, gc.Equals, 0)
		c.Check(id, gc.Equals, "")
		c.Check(request, gc.Equals, "WatchSecretsChanges")
		c.Check(arg, jc.DeepEquals, params.Entities{
			Entities: []params.Entity{{Tag: "unit-foo-0"}},
		})
		c.Assert(result, gc.FitsTypeOf, &params.StringsWatchResults{})
		*(result.(*params.StringsWatchResults)) = params.StringsWatchResults{
			Results: []params.StringsWatchResult{{
				Error: &params.Error{Message: "FAIL"},
			}},
		}
		return nil
	})
	client := secretsmanager.NewClient(apiCaller)
	_, err := client.WatchSecretsChanges("foo/0")
	c.Assert(err, gc.ErrorMatches, "FAIL")
}

func (s *SecretsSuite) GetLatestSecretsRevisionInfo(c *gc.C) {
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		c.Check(objType, gc.Equals, "SecretsManager")
		c.Check(version, gc.Equals, 0)
		c.Check(id, gc.Equals, "")
		c.Check(request, gc.Equals, "GetLatestSecretsRevisionInfo")
		c.Check(arg, jc.DeepEquals, params.GetSecretConsumerInfoArgs{
			ConsumerTag: "unit-foo-0",
			URIs: []string{
				"secret:9m4e2mr0ui3e8a215n4g", "secret:8n3e2mr0ui3e8a215n5h", "secret:7c5e2mr0ui3e8a2154r2"},
		})
		c.Assert(result, gc.FitsTypeOf, &params.SecretConsumerInfoResults{})
		*(result.(*params.SecretConsumerInfoResults)) = params.SecretConsumerInfoResults{
			Results: []params.SecretConsumerInfoResult{{
				Revision: 666,
				Label:    "foobar",
			}, {
				Error: &params.Error{Code: params.CodeUnauthorized},
			}, {
				Error: &params.Error{Code: params.CodeNotFound},
			}},
		}
		return nil
	})
	var info map[string]secretsmanager.SecretRevisionInfo
	client := secretsmanager.NewClient(apiCaller)
	info, err := client.GetLatestSecretsRevisionInfo("foo-0", []string{
		"secret:9m4e2mr0ui3e8a215n4g", "secret:8n3e2mr0ui3e8a215n5h", "secret:7c5e2mr0ui3e8a2154r2"})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(info, jc.DeepEquals, map[string]secretsmanager.SecretRevisionInfo{})
}

func (s *SecretsSuite) TestWatchSecretsRotationChanges(c *gc.C) {
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		c.Check(objType, gc.Equals, "SecretsManager")
		c.Check(version, gc.Equals, 0)
		c.Check(id, gc.Equals, "")
		c.Check(request, gc.Equals, "WatchSecretsRotationChanges")
		c.Check(arg, jc.DeepEquals, params.Entities{
			Entities: []params.Entity{{Tag: "application-app"}},
		})
		c.Assert(result, gc.FitsTypeOf, &params.SecretRotationWatchResults{})
		*(result.(*params.SecretRotationWatchResults)) = params.SecretRotationWatchResults{
			[]params.SecretRotationWatchResult{{
				Error: &params.Error{Message: "FAIL"},
			}},
		}
		return nil
	})
	client := secretsmanager.NewClient(apiCaller)
	_, err := client.WatchSecretsRotationChanges("application-app")
	c.Assert(err, gc.ErrorMatches, "FAIL")
}

func (s *SecretsSuite) TestSecretRotated(c *gc.C) {
	now := time.Now()
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		c.Check(objType, gc.Equals, "SecretsManager")
		c.Check(version, gc.Equals, 0)
		c.Check(id, gc.Equals, "")
		c.Check(request, gc.Equals, "SecretsRotated")
		c.Check(arg, jc.DeepEquals, params.SecretRotatedArgs{
			Args: []params.SecretRotatedArg{{
				URI:  "secret:9m4e2mr0ui3e8a215n4g",
				When: now,
			}},
		})
		c.Assert(result, gc.FitsTypeOf, &params.ErrorResults{})
		*(result.(*params.ErrorResults)) = params.ErrorResults{
			[]params.ErrorResult{{
				Error: &params.Error{Message: "boom"},
			}},
		}
		return nil
	})
	client := secretsmanager.NewClient(apiCaller)
	err := client.SecretRotated("secret:9m4e2mr0ui3e8a215n4g", now)
	c.Assert(err, gc.ErrorMatches, "boom")
}

func (s *SecretsSuite) TestGrant(c *gc.C) {
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		c.Check(objType, gc.Equals, "SecretsManager")
		c.Check(version, gc.Equals, 0)
		c.Check(id, gc.Equals, "")
		c.Check(request, gc.Equals, "SecretsGrant")
		c.Check(arg, jc.DeepEquals, params.GrantRevokeSecretArgs{
			Args: []params.GrantRevokeSecretArg{{
				URI:         "secret:9m4e2mr0ui3e8a215n4g",
				ScopeTag:    "relation-wordpress.db#mysql.server",
				SubjectTags: []string{"unit-wordpress-0"},
				Role:        "view",
			}},
		})
		c.Assert(result, gc.FitsTypeOf, &params.ErrorResults{})
		*(result.(*params.ErrorResults)) = params.ErrorResults{
			Results: []params.ErrorResult{{
				Error: &params.Error{Message: "FAIL"},
			}},
		}
		return nil
	})
	client := secretsmanager.NewClient(apiCaller)
	err := client.Grant("secret:9m4e2mr0ui3e8a215n4g", &secretsmanager.SecretRevokeGrantArgs{
		UnitName:    ptr("wordpress/0"),
		RelationKey: ptr("wordpress:db mysql:server"),
		Role:        secrets.RoleView,
	})
	c.Assert(err, gc.ErrorMatches, "FAIL")
}

func (s *SecretsSuite) TestRevoke(c *gc.C) {
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		c.Check(objType, gc.Equals, "SecretsManager")
		c.Check(version, gc.Equals, 0)
		c.Check(id, gc.Equals, "")
		c.Check(request, gc.Equals, "SecretsRevoke")
		c.Check(arg, jc.DeepEquals, params.GrantRevokeSecretArgs{
			Args: []params.GrantRevokeSecretArg{{
				URI:         "secret:9m4e2mr0ui3e8a215n4g",
				ScopeTag:    "relation-wordpress.db#mysql.server",
				SubjectTags: []string{"application-wordpress"},
				Role:        "view",
			}},
		})
		c.Assert(result, gc.FitsTypeOf, &params.ErrorResults{})
		*(result.(*params.ErrorResults)) = params.ErrorResults{
			Results: []params.ErrorResult{{
				Error: &params.Error{Message: "FAIL"},
			}},
		}
		return nil
	})
	client := secretsmanager.NewClient(apiCaller)
	err := client.Revoke("secret:9m4e2mr0ui3e8a215n4g", &secretsmanager.SecretRevokeGrantArgs{
		ApplicationName: ptr("wordpress"),
		RelationKey:     ptr("wordpress:db mysql:server"),
		Role:            secrets.RoleView,
	})
	c.Assert(err, gc.ErrorMatches, "FAIL")
}
