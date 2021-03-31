// Copyright 2021 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package jujuc

import (
	"github.com/juju/cmd"
	"github.com/juju/errors"

	jujucmd "github.com/juju/juju/cmd"
)

// cloudEventGetCommand implements the cloud-event-get command.
type cloudEventGetCommand struct {
	cmd.CommandBase
	ctx          Context
	out          cmd.Output
	resourceType string
	resourceName string
}

// NewCloudEventGetCommand returns a new cloudEventGetCommand with the given context.
func NewCloudEventGetCommand(ctx Context) (cmd.Command, error) {
	return &cloudEventGetCommand{ctx: ctx}, nil
}

// Info is part of the cmd.Command interface.
func (c *cloudEventGetCommand) Info() *cmd.Info {
	doc := `
cloud-event-get returns the cloud event watched by the application.
`
	return jujucmd.Info(&cmd.Info{
		Name:    "cloud-event-get",
		Args:    "",
		Purpose: "get cloud events",
		Doc:     doc,
	})
}

// Init is part of the cmd.Command interface.
func (c *cloudEventGetCommand) Init(args []string) error {
	if len(args) != 2 {
		return errors.New("cloud-event-get requires resource type and resource name")
	}
	c.resourceType = args[0]
	c.resourceName = args[1]
	return nil
}

// Run is part of the cmd.Command interface.
func (c *cloudEventGetCommand) Run(ctx *cmd.Context) error {
	events, err := c.ctx.GetCloudEvent(c.resourceType, c.resourceName)
	if err != nil {
		return errors.Annotatef(err, "cannot access cloud events")
	}
	return c.out.Write(ctx, events)
}
