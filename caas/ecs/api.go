// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package ecs

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/juju/errors"

	cloudspec "github.com/juju/juju/environs/cloudspec"
)

type awsLogger struct {
	session *session.Session
}

func (l awsLogger) Log(args ...interface{}) {
	logger.Tracef("awsLogger %p: %s", l.session, fmt.Sprint(args...))
}

func newECSClient(cloud cloudspec.CloudSpec) (*ecs.ECS, error) {
	if err := validateCloudSpec(cloud); err != nil {
		return nil, errors.Annotate(err, "validating cloud spec")
	}

	credentialAttrs := cloud.Credential.Attributes()
	accessKey := credentialAttrs["access-key"]
	secretKey := credentialAttrs["secret-key"]

	s := session.Must(session.NewSession())
	config := &aws.Config{
		Retryer: client.DefaultRetryer{
			NumMaxRetries:    10,
			MinRetryDelay:    time.Second,
			MinThrottleDelay: time.Second,
			MaxRetryDelay:    time.Minute,
			MaxThrottleDelay: time.Minute,
		},
		Region: aws.String(cloud.Region),
		Credentials: credentials.NewStaticCredentialsFromCreds(credentials.Value{
			AccessKeyID:     accessKey,
			SecretAccessKey: secretKey,
		}),
	}

	// Enable request and response logging, but only if TRACE is enabled (as
	// they're probably fairly expensive to produce).
	if logger.IsTraceEnabled() {
		config.Logger = awsLogger{s}
		config.LogLevel = aws.LogLevel(aws.LogDebug | aws.LogDebugWithRequestErrors | aws.LogDebugWithRequestRetries)
	}

	return ecs.New(s, config), nil
}
