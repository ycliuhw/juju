// Copyright 2021 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package jujuc

import (
	"github.com/juju/cmd"
	"github.com/juju/errors"

	jujucmd "github.com/juju/juju/cmd"
)

// registerCloudEventCommand implements the register-cloud-event command.
type registerCloudEventCommand struct {
	cmd.CommandBase
	ctx          Context
	resourceType string
	resourceName string
}

// NewRegisterCloudEventCommand returns a new registerCloudEventCommand with the given context.
func NewRegisterCloudEventCommand(ctx Context) (cmd.Command, error) {
	return &registerCloudEventCommand{ctx: ctx}, nil
}

// Info is part of the cmd.Command interface.
func (c *registerCloudEventCommand) Info() *cmd.Info {
	doc := `
register-cloud-event registers the cloud resource to watch.
`
	return jujucmd.Info(&cmd.Info{
		Name:    "register-cloud-event",
		Args:    "configmap configmap-foo",
		Purpose: "registers the cloud resource to watch",
		Doc:     doc,
	})
}

// Init is part of the cmd.Command interface.
func (c *registerCloudEventCommand) Init(args []string) error {
	if len(args) != 2 {
		return errors.New("register-cloud-event requires resource type and resource name")
	}
	c.resourceType = args[0]
	c.resourceName = args[1]
	return nil
}

// Run is part of the cmd.Command interface.
func (c *registerCloudEventCommand) Run(ctx *cmd.Context) error {
	return c.ctx.RegisterCloudEvent(c.resourceType, c.resourceName)
}
