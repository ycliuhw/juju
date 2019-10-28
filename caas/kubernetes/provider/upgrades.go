// Copyright 2019 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package provider

import (
	"github.com/juju/errors"

	"github.com/juju/juju/environs"
	// "github.com/juju/juju/environs/config"
	"github.com/juju/juju/environs/context"
)

func (k *kubernetesClient) upgradeConfigMapLabels(cmName string, expectedLabels map[string]string) error {
	cm, err := k.getConfigMap(cmName)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return errors.Trace(err)
	}
	cm.SetLabels(expectedLabels)
	return errors.Trace(k.updateConfigMap(cm))
}

func (k *kubernetesClient) upgradeConfigMaps(appName string) error {
	return nil
}

// UpgradeOperations is part of the upgrades.OperationSource interface.
func (env *kubernetesEnvironProvider) UpgradeOperations(
	_ context.ProviderCallContext, args environs.UpgradeOperationsParams,
) []environs.UpgradeOperation {
	return []environs.UpgradeOperation{
		{
			TargetVersion: environProviderVersion1,
			Steps: []environs.UpgradeStep{
				configMapLabelsUpgradeStep{env, args.ControllerUUID},
			},
		},
	}
}

type configMapLabelsUpgradeStep struct {
	env            *kubernetesEnvironProvider
	controllerUUID string
}

// Description is part of the environs.UpgradeStep interface.
func (configMapLabelsUpgradeStep) Description() string {
	return "Upgrade labels for configmap resources"
}

// Run is part of the environs.UpgradeStep interface.
func (storageConfig configMapLabelsUpgradeStep) Run(ctx context.ProviderCallContext) error {

	return nil
}
