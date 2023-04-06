// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package deployer

import (
	"github.com/juju/charm/v10"
	"github.com/juju/juju/core/constraints"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

type bundleSuite struct {
}

var _ = gc.Suite(&bundleSuite{})

func (s *bundleSuite) TestCheckExplicitBase(c *gc.C) {
	explicitSeriesErrorUbuntu := "series must be explicitly provided for \"ch:ubuntu\" when image-id constraint is used"
	explicitSeriesError := "series must be explicitly provided for(.)*"

	testCases := []struct {
		title         string
		deployBundle  deployBundle
		bundleData    *charm.BundleData
		expectedError string
	}{
		{
			title: "two apps, no image-id, no series -> no error",
			bundleData: &charm.BundleData{
				Applications: map[string]*charm.ApplicationSpec{
					"prometheus2": {
						Charm:       "ch:prometheus2",
						Constraints: "cpu-cores=2",
					},
					"ubuntu": {
						Charm:       "ch:ubuntu",
						Constraints: "mem=2G",
					},
				},
			},
			deployBundle: deployBundle{},
		},
		{
			title: "two apps, one with image-id, no series -> error",
			bundleData: &charm.BundleData{
				Applications: map[string]*charm.ApplicationSpec{
					"prometheus2": {
						Charm:       "ch:prometheus2",
						Constraints: "image-id=ubuntu-bf2",
					},
					"ubuntu": {
						Charm:       "ch:ubuntu",
						Constraints: "mem=2G",
					},
				},
			},
			deployBundle:  deployBundle{},
			expectedError: explicitSeriesError,
		},
		{
			title: "two apps, model with image-id, no series -> error",
			bundleData: &charm.BundleData{
				Applications: map[string]*charm.ApplicationSpec{
					"prometheus2": {
						Charm:       "ch:prometheus2",
						Constraints: "cpu-cores=2",
					},
					"ubuntu": {
						Charm:       "ch:ubuntu",
						Constraints: "mem=2G",
					},
				},
			},
			deployBundle: deployBundle{
				modelConstraints: constraints.Value{
					ImageID: strptr("ubuntu-bf2"),
				},
			},
			expectedError: explicitSeriesError,
		},
		{
			title: "two apps, model and one app with image-id, no series -> error",
			bundleData: &charm.BundleData{
				Applications: map[string]*charm.ApplicationSpec{
					"prometheus2": {
						Charm:       "ch:prometheus2",
						Constraints: "image-id=ubuntu-bf2",
					},
					"ubuntu": {
						Charm:       "ch:ubuntu",
						Constraints: "mem=2G",
					},
				},
			},
			deployBundle: deployBundle{
				modelConstraints: constraints.Value{
					ImageID: strptr("ubuntu-bf2"),
				},
			},
			expectedError: explicitSeriesError,
		},
		{
			title: "two apps, machine with image-id in (app).To, no series -> error",
			bundleData: &charm.BundleData{
				Applications: map[string]*charm.ApplicationSpec{
					"prometheus2": {
						Charm:       "ch:prometheus2",
						Constraints: "cpu-cores=2",
					},
					"ubuntu": {
						Charm:       "ch:ubuntu",
						Constraints: "mem=2G",
						To:          []string{"0"},
					},
				},
				Machines: map[string]*charm.MachineSpec{
					"0": {
						Constraints: "image-id=ubuntu-bf2",
					},
					"1": {
						Constraints: "mem=2G",
					},
				},
			},
			deployBundle:  deployBundle{},
			expectedError: explicitSeriesErrorUbuntu,
		},
		{
			title: "two apps, machine with image-id not in (app).To, no series -> no error",
			bundleData: &charm.BundleData{
				Applications: map[string]*charm.ApplicationSpec{
					"prometheus2": {
						Charm:       "ch:prometheus2",
						Constraints: "cpu-cores=2",
					},
					"ubuntu": {
						Charm:       "ch:ubuntu",
						Constraints: "mem=2G",
						To:          []string{"1"},
					},
				},
				Machines: map[string]*charm.MachineSpec{
					"0": {
						Constraints: "image-id=ubuntu-bf2",
					},
					"1": {
						Constraints: "mem=2G",
					},
				},
			},
			deployBundle: deployBundle{},
		},
		{
			title: "two apps, one with image-id, series in same app -> no error",
			bundleData: &charm.BundleData{
				Applications: map[string]*charm.ApplicationSpec{
					"prometheus2": {
						Charm:       "ch:prometheus2",
						Series:      "focal",
						Constraints: "image-id=ubuntu-bf2",
					},
					"ubuntu": {
						Charm:       "ch:ubuntu",
						Constraints: "mem=2G",
					},
				},
			},
			deployBundle: deployBundle{},
		},
		{
			title: "two apps, model with image-id, series in one app -> error",
			bundleData: &charm.BundleData{
				Applications: map[string]*charm.ApplicationSpec{
					"prometheus2": {
						Charm:       "ch:prometheus2",
						Series:      "focal",
						Constraints: "cpu-cores=2",
					},
					"ubuntu": {
						Charm:       "ch:ubuntu",
						Constraints: "mem=2G",
					},
				},
			},
			deployBundle: deployBundle{
				modelConstraints: constraints.Value{
					ImageID: strptr("ubuntu-bf2"),
				},
			},
			expectedError: explicitSeriesErrorUbuntu,
		},
		{
			title: "two apps, model with image-id, series in two apps -> no error",
			bundleData: &charm.BundleData{
				Applications: map[string]*charm.ApplicationSpec{
					"prometheus2": {
						Charm:       "ch:prometheus2",
						Series:      "focal",
						Constraints: "cpu-cores=2",
					},
					"ubuntu": {
						Charm:       "ch:ubuntu",
						Series:      "focal",
						Constraints: "mem=2G",
					},
				},
			},
			deployBundle: deployBundle{
				modelConstraints: constraints.Value{
					ImageID: strptr("ubuntu-bf2"),
				},
			},
		},
		{
			title: "two apps, model and one app with image-id, series in one app -> error",

			bundleData: &charm.BundleData{
				Applications: map[string]*charm.ApplicationSpec{
					"prometheus2": {
						Charm:       "ch:prometheus2",
						Series:      "focal",
						Constraints: "image-id=ubuntu-bf2",
					},
					"ubuntu": {
						Charm:       "ch:ubuntu",
						Constraints: "mem=2G",
					},
				},
			},
			deployBundle: deployBundle{
				modelConstraints: constraints.Value{
					ImageID: strptr("ubuntu-bf2"),
				},
			},
			expectedError: explicitSeriesErrorUbuntu,
		},
		{
			title: "two apps, model and one app with image-id, series in two apps -> no error",

			bundleData: &charm.BundleData{
				Applications: map[string]*charm.ApplicationSpec{
					"prometheus2": {
						Charm:       "ch:prometheus2",
						Series:      "focal",
						Constraints: "image-id=ubuntu-bf2",
					},
					"ubuntu": {
						Charm:       "ch:ubuntu",
						Series:      "focal",
						Constraints: "mem=2G",
					},
				},
			},
			deployBundle: deployBundle{
				modelConstraints: constraints.Value{
					ImageID: strptr("ubuntu-bf2"),
				},
			},
		},
		{
			title: "two apps, machine with image-id in (app).To, series in app -> no error",
			bundleData: &charm.BundleData{
				Applications: map[string]*charm.ApplicationSpec{
					"prometheus2": {
						Charm:       "ch:prometheus2",
						Constraints: "cpu-cores=2",
					},
					"ubuntu": {
						Charm:       "ch:ubuntu",
						Series:      "jammy",
						Constraints: "mem=2G",
						To:          []string{"0"},
					},
				},
				Machines: map[string]*charm.MachineSpec{
					"0": {
						Constraints: "image-id=ubuntu-bf2",
					},
					"1": {
						Constraints: "mem=2G",
					},
				},
			},
			deployBundle: deployBundle{},
		},
		{
			title: "two apps, one with image-id, series in bundle -> no error",
			bundleData: &charm.BundleData{
				Series: "focal",
				Applications: map[string]*charm.ApplicationSpec{
					"prometheus2": {
						Charm:       "ch:prometheus2",
						Constraints: "image-id=ubuntu-bf2",
					},
					"ubuntu": {
						Charm:       "ch:ubuntu",
						Constraints: "mem=2G",
					},
				},
			},
			deployBundle: deployBundle{},
		},
		{
			title: "two apps, model with image-id, series in bundle -> no error",
			bundleData: &charm.BundleData{
				Series: "focal",
				Applications: map[string]*charm.ApplicationSpec{
					"prometheus2": {
						Charm:       "ch:prometheus2",
						Constraints: "cpu-cores=2",
					},
					"ubuntu": {
						Charm:       "ch:ubuntu",
						Constraints: "mem=2G",
					},
				},
			},
			deployBundle: deployBundle{
				modelConstraints: constraints.Value{
					ImageID: strptr("ubuntu-bf2"),
				},
			},
		},
		{
			title: "two apps, model with image-id, series in bundle and app -> no error",
			bundleData: &charm.BundleData{
				Series: "focal",
				Applications: map[string]*charm.ApplicationSpec{
					"prometheus2": {
						Charm:       "ch:prometheus2",
						Constraints: "cpu-cores=2",
					},
					"ubuntu": {
						Charm:       "ch:ubuntu",
						Series:      "jammy",
						Constraints: "mem=2G",
					},
				},
			},
			deployBundle: deployBundle{
				modelConstraints: constraints.Value{
					ImageID: strptr("ubuntu-bf2"),
				},
			},
		},
		{
			title: "two apps, machine with image-id in (app).To, series in bundle -> no error",
			bundleData: &charm.BundleData{
				Series: "focal",
				Applications: map[string]*charm.ApplicationSpec{
					"prometheus2": {
						Charm:       "ch:prometheus2",
						Constraints: "cpu-cores=2",
					},
					"ubuntu": {
						Charm:       "ch:ubuntu",
						Constraints: "mem=2G",
						To:          []string{"0"},
					},
				},
				Machines: map[string]*charm.MachineSpec{
					"0": {
						Constraints: "image-id=ubuntu-bf2",
					},
					"1": {
						Constraints: "mem=2G",
					},
				},
			},
			deployBundle: deployBundle{},
		},
	}
	for i, test := range testCases {
		c.Logf("test %d [%s]", i, test.title)

		err := test.deployBundle.checkExplicitSeries(test.bundleData)

		if test.expectedError != "" {
			c.Check(err, gc.ErrorMatches, test.expectedError)
		} else {
			c.Check(err, jc.ErrorIsNil)
		}
	}
}