// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package tracing

import (
	"net/http"
	"net/http/httputil"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/go-autorest/autorest"
	"github.com/juju/loggo"
)

type LoggingPolicy struct {
	Logger loggo.Logger
}

func (p *LoggingPolicy) Do(req *policy.Request) (*http.Response, error) {
	if p.Logger.IsTraceEnabled() {
		dump, err := httputil.DumpRequest(req.Raw(), true)
		if err != nil {
			p.Logger.Tracef("failed to dump request: %v", err)
			p.Logger.Tracef("%+v", req.Raw())
		} else {
			p.Logger.Tracef("%s", dump)
		}
	}
	resp, err := req.Next()
	if err == nil && p.Logger.IsTraceEnabled() {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			p.Logger.Tracef("failed to dump response: %v", err)
			p.Logger.Tracef("%+v", resp)
		} else {
			p.Logger.Tracef("%s", dump)
		}
	}
	return resp, err
}

// PrepareDecorator returns an autorest.PrepareDecorator that
// logs requests at trace level.
func PrepareDecorator(logger loggo.Logger) autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			if logger.IsTraceEnabled() {
				dump, err := httputil.DumpRequest(r, true)
				if err != nil {
					logger.Tracef("failed to dump request: %v", err)
					logger.Tracef("%+v", r)
				} else {
					logger.Tracef("%s", dump)
				}
			}
			return p.Prepare(r)
		})
	}
}

// RespondDecorator returns an autorest.RespondDecorator that
// logs responses at trace level.
func RespondDecorator(logger loggo.Logger) autorest.RespondDecorator {
	return func(r autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(resp *http.Response) error {
			if logger.IsTraceEnabled() {
				dump, err := httputil.DumpResponse(resp, true)
				if err != nil {
					logger.Tracef("failed to dump response: %v", err)
					logger.Tracef("%+v", resp)
				} else {
					logger.Tracef("%s", dump)
				}
			}
			return r.Respond(resp)
		})
	}
}
