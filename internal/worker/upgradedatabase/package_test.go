// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package upgradedatabase

import (
	"testing"

	jujutesting "github.com/juju/testing"
	gomock "go.uber.org/mock/gomock"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/core/logger"
	loggertesting "github.com/juju/juju/internal/logger/testing"
)

//go:generate go run go.uber.org/mock/mockgen -typed -package upgradedatabase -destination lock_mock_test.go github.com/juju/juju/internal/worker/gate Lock
//go:generate go run go.uber.org/mock/mockgen -typed -package upgradedatabase -destination agent_mock_test.go github.com/juju/juju/agent Agent,Config,ConfigSetter
//go:generate go run go.uber.org/mock/mockgen -typed -package upgradedatabase -destination servicefactory_mock_test.go github.com/juju/juju/internal/servicefactory ControllerServiceFactory
//go:generate go run go.uber.org/mock/mockgen -typed -package upgradedatabase -destination database_mock_test.go github.com/juju/juju/core/database DBGetter
//go:generate go run go.uber.org/mock/mockgen -typed -package upgradedatabase -destination service_mock_test.go github.com/juju/juju/internal/worker/upgradedatabase UpgradeService,ModelService
//go:generate go run go.uber.org/mock/mockgen -typed -package upgradedatabase -destination worker_mock_test.go github.com/juju/worker/v4 Worker

func TestPackage(t *testing.T) {
	gc.TestingT(t)
}

type baseSuite struct {
	jujutesting.IsolationSuite

	lock           *MockLock
	agent          *MockAgent
	agentConfig    *MockConfig
	serviceFactory *MockControllerServiceFactory

	dbGetter *MockDBGetter

	upgradeService *MockUpgradeService
	modelService   *MockModelService

	logger logger.Logger
}

func (s *baseSuite) setupMocks(c *gc.C) *gomock.Controller {
	ctrl := gomock.NewController(c)

	s.lock = NewMockLock(ctrl)
	s.agent = NewMockAgent(ctrl)
	s.agentConfig = NewMockConfig(ctrl)
	s.serviceFactory = NewMockControllerServiceFactory(ctrl)

	s.dbGetter = NewMockDBGetter(ctrl)

	s.upgradeService = NewMockUpgradeService(ctrl)
	s.modelService = NewMockModelService(ctrl)

	s.logger = loggertesting.WrapCheckLog(c)

	return ctrl
}
