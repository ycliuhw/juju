// Copyright 2021 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package jujuc

import (
	"github.com/juju/cmd"
	"github.com/juju/errors"

	jujucmd "github.com/juju/juju/cmd"
)

// unregisterCloudEventCommand implements the unregister-cloud-event command.
type unregisterCloudEventCommand struct {
	cmd.CommandBase
	ctx          Context
	resourceType string
	resourceName string
}

// NewUnregisterCloudEventCommand returns a new unregisterCloudEventCommand with the given context.
func NewUnregisterCloudEventCommand(ctx Context) (cmd.Command, error) {
	return &unregisterCloudEventCommand{ctx: ctx}, nil
}

// Info is part of the cmd.Command interface.
func (c *unregisterCloudEventCommand) Info() *cmd.Info {
	doc := `
unregister-cloud-event registers the cloud resource to watch.
`
	return jujucmd.Info(&cmd.Info{
		Name:    "unregister-cloud-event",
		Args:    "configmap configmap-foo",
		Purpose: "unregisters the cloud resource to watch",
		Doc:     doc,
	})
}

// Init is part of the cmd.Command interface.
func (c *unregisterCloudEventCommand) Init(args []string) error {
	if len(args) != 2 {
		return errors.New("unregister-cloud-event requires resource type and resource name")
	}
	c.resourceType = args[0]
	c.resourceName = args[1]
	return cmd.CheckEmpty(args)
}

// Run is part of the cmd.Command interface.
func (c *unregisterCloudEventCommand) Run(ctx *cmd.Context) error {
	return c.ctx.UnregisterCloudEvent(c.resourceType, c.resourceName)
}
