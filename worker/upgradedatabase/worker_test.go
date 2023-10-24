// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package upgradedatabase

import (
	"context"
	"time"

	"github.com/juju/clock"
	"github.com/juju/errors"
	names "github.com/juju/names/v4"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils/v3"
	"github.com/juju/version/v2"
	"github.com/juju/worker/v3"
	"github.com/juju/worker/v3/dependency"
	"github.com/juju/worker/v3/workertest"
	"go.uber.org/mock/gomock"
	gc "gopkg.in/check.v1"

	coredatabase "github.com/juju/juju/core/database"
	"github.com/juju/juju/core/testing"
	upgrade "github.com/juju/juju/core/upgrade"
	watcher "github.com/juju/juju/core/watcher"
	"github.com/juju/juju/core/watcher/watchertest"
	model "github.com/juju/juju/domain/model"
	"github.com/juju/juju/domain/schema"
	domainupgrade "github.com/juju/juju/domain/upgrade"
	upgradeerrors "github.com/juju/juju/domain/upgrade/errors"
	databasetesting "github.com/juju/juju/internal/database/testing"
	jujuversion "github.com/juju/juju/version"
)

// baseSuite is embedded in both the worker and manifold tests.
// Tests should not go on this suite directly.

type workerSuite struct {
	baseSuite
	databasetesting.DqliteSuite

	upgradeUUID domainupgrade.UUID
}

var _ = gc.Suite(&workerSuite{})

func (s *workerSuite) TestNewLockSameVersionUnlocked(c *gc.C) {
	defer s.setupMocks(c).Finish()

	s.agentConfig.EXPECT().UpgradedToVersion().Return(jujuversion.Current)
	c.Assert(NewLock(s.agentConfig).IsUnlocked(), jc.IsTrue)
}

func (s *workerSuite) TestNewLockOldVersionLocked(c *gc.C) {
	defer s.setupMocks(c).Finish()

	s.agentConfig.EXPECT().UpgradedToVersion().Return(version.Number{})
	c.Assert(NewLock(s.agentConfig).IsUnlocked(), jc.IsFalse)
}

func (s *workerSuite) TestLockAlreadyUnlocked(c *gc.C) {
	defer s.setupMocks(c).Finish()

	s.lock.EXPECT().IsUnlocked().Return(true)

	w, err := NewUpgradeDatabaseWorker(s.getConfig())
	c.Assert(err, jc.ErrorIsNil)

	err = workertest.CheckKill(c, w)
	c.Check(err, jc.ErrorIs, dependency.ErrUninstall)
}

func (s *workerSuite) TestLockIsUnlockedIfMatchingVersions(c *gc.C) {
	defer s.setupMocks(c).Finish()

	s.lock.EXPECT().IsUnlocked().Return(false)
	s.lock.EXPECT().Unlock()

	cfg := s.getConfig()
	cfg.FromVersion = jujuversion.Current
	cfg.ToVersion = jujuversion.Current

	w, err := NewUpgradeDatabaseWorker(cfg)
	c.Assert(err, jc.ErrorIsNil)

	err = workertest.CheckKill(c, w)
	c.Check(err, jc.ErrorIs, dependency.ErrUninstall)
}

func (s *workerSuite) TestWatchUpgradeCompleted(c *gc.C) {
	defer s.setupMocks(c).Finish()

	// Ensure that the update hasn't already happened.
	s.lock.EXPECT().IsUnlocked().Return(false)

	cfg := s.getConfig()

	chCompleted := make(chan struct{})
	chFailed := make(chan struct{})

	completedWatcher := watchertest.NewMockNotifyWatcher(chCompleted)
	defer workertest.DirtyKill(c, completedWatcher)

	failedWatcher := watchertest.NewMockNotifyWatcher(chFailed)
	defer workertest.DirtyKill(c, failedWatcher)

	// Walk through the upgrade process:
	//  - Create Upgrade, but it's already started.
	//  - Get the active upgrade.
	//  - Get the upgrade info and ensure it's not in an error state.
	//  - Watch for the upgrade to be completed.
	//  - Watch for the upgrade to be failed, but do not act upon it.

	srv := s.upgradeService.EXPECT()
	srv.CreateUpgrade(gomock.Any(), cfg.FromVersion, cfg.ToVersion).Return(domainupgrade.UUID(""), upgradeerrors.ErrUpgradeAlreadyStarted)
	srv.ActiveUpgrade(gomock.Any()).Return(s.upgradeUUID, nil)
	srv.UpgradeInfo(gomock.Any(), s.upgradeUUID).Return(upgrade.Info{State: upgrade.Created}, nil)
	srv.SetControllerReady(gomock.Any(), s.upgradeUUID, "0").Return(nil)

	srv.WatchForUpgradeState(gomock.Any(), s.upgradeUUID, upgrade.DBCompleted).Return(completedWatcher, nil)
	srv.WatchForUpgradeState(gomock.Any(), s.upgradeUUID, upgrade.Error).Return(failedWatcher, nil)

	done := make(chan struct{})

	// We expect the lock to be unlocked when the upgrade completes.
	s.lock.EXPECT().Unlock().DoAndReturn(func() {
		defer close(done)
	})

	w, err := NewUpgradeDatabaseWorker(cfg)
	c.Assert(err, jc.ErrorIsNil)

	// Dispatch the initial event.
	s.dispatchChange(c, chCompleted)
	s.dispatchChange(c, chFailed)

	s.dispatchChange(c, chCompleted)

	select {
	case <-done:
	case <-time.After(testing.LongWait):
		c.Fatalf("timed out waiting for unlock")
	}

	err = workertest.CheckKill(c, w)
	c.Check(err, jc.ErrorIs, dependency.ErrUninstall)
}

func (s *workerSuite) TestWatchUpgradeCompletedErrorSetControllerReady(c *gc.C) {
	defer s.setupMocks(c).Finish()

	// Ensure that the update hasn't already happened.
	s.lock.EXPECT().IsUnlocked().Return(false)

	cfg := s.getConfig()

	chCompleted := make(chan struct{})
	chFailed := make(chan struct{})

	completedWatcher := watchertest.NewMockNotifyWatcher(chCompleted)
	defer workertest.DirtyKill(c, completedWatcher)

	failedWatcher := watchertest.NewMockNotifyWatcher(chFailed)
	defer workertest.DirtyKill(c, failedWatcher)

	// Walk through the upgrade process:
	//  - Create Upgrade, but it's already started.
	//  - Get the active upgrade.
	//  - Get the upgrade info and ensure it's not in an error state.
	//  - Set controller ready, but fails.
	//  - Set upgrade failed, so it causes everyone else to bounce.

	done := make(chan struct{})

	srv := s.upgradeService.EXPECT()
	srv.CreateUpgrade(gomock.Any(), cfg.FromVersion, cfg.ToVersion).Return(domainupgrade.UUID(""), upgradeerrors.ErrUpgradeAlreadyStarted)
	srv.ActiveUpgrade(gomock.Any()).Return(s.upgradeUUID, nil)
	srv.UpgradeInfo(gomock.Any(), s.upgradeUUID).Return(upgrade.Info{State: upgrade.Created}, nil)
	srv.SetControllerReady(gomock.Any(), s.upgradeUUID, "0").Return(errors.Errorf("boom"))
	srv.SetDBUpgradeFailed(gomock.Any(), s.upgradeUUID).DoAndReturn(func(ctx context.Context, uuid domainupgrade.UUID) error {
		defer close(done)
		return nil
	})

	srv.WatchForUpgradeState(gomock.Any(), s.upgradeUUID, upgrade.DBCompleted).Return(completedWatcher, nil)
	srv.WatchForUpgradeState(gomock.Any(), s.upgradeUUID, upgrade.Error).Return(failedWatcher, nil)

	w, err := NewUpgradeDatabaseWorker(cfg)
	c.Assert(err, jc.ErrorIsNil)

	// Dispatch the initial event.
	s.dispatchChange(c, chCompleted)
	s.dispatchChange(c, chFailed)

	select {
	case <-done:
	case <-time.After(testing.LongWait):
		c.Fatalf("timed out waiting for unlock")
	}

	err = workertest.CheckKill(c, w)
	c.Check(err, jc.ErrorIs, dependency.ErrBounce)
}

func (s *workerSuite) TestWatchUpgradeCompletedErrorSetControllerReadyError(c *gc.C) {
	defer s.setupMocks(c).Finish()

	// Ensure that the update hasn't already happened.
	s.lock.EXPECT().IsUnlocked().Return(false)

	cfg := s.getConfig()

	chCompleted := make(chan struct{})
	chFailed := make(chan struct{})

	completedWatcher := watchertest.NewMockNotifyWatcher(chCompleted)
	defer workertest.DirtyKill(c, completedWatcher)

	failedWatcher := watchertest.NewMockNotifyWatcher(chFailed)
	defer workertest.DirtyKill(c, failedWatcher)

	// Walk through the upgrade process:
	//  - Create Upgrade, but it's already started.
	//  - Get the active upgrade.
	//  - Get the upgrade info and ensure it's not in an error state.
	//  - Set controller ready, but fails.
	//  - Set upgrade failed also fails, which kills the worker causing manual intervention.

	done := make(chan struct{})

	srv := s.upgradeService.EXPECT()
	srv.CreateUpgrade(gomock.Any(), cfg.FromVersion, cfg.ToVersion).Return(domainupgrade.UUID(""), upgradeerrors.ErrUpgradeAlreadyStarted)
	srv.ActiveUpgrade(gomock.Any()).Return(s.upgradeUUID, nil)
	srv.UpgradeInfo(gomock.Any(), s.upgradeUUID).Return(upgrade.Info{State: upgrade.Created}, nil)
	srv.SetControllerReady(gomock.Any(), s.upgradeUUID, "0").Return(errors.Errorf("boom"))
	srv.SetDBUpgradeFailed(gomock.Any(), s.upgradeUUID).DoAndReturn(func(ctx context.Context, uuid domainupgrade.UUID) error {
		defer close(done)
		return errors.Errorf("boom")
	})

	srv.WatchForUpgradeState(gomock.Any(), s.upgradeUUID, upgrade.DBCompleted).Return(completedWatcher, nil)
	srv.WatchForUpgradeState(gomock.Any(), s.upgradeUUID, upgrade.Error).Return(failedWatcher, nil)

	w, err := NewUpgradeDatabaseWorker(cfg)
	c.Assert(err, jc.ErrorIsNil)
	defer workertest.DirtyKill(c, w)

	// Dispatch the initial event.
	s.dispatchChange(c, chCompleted)
	s.dispatchChange(c, chFailed)

	select {
	case <-done:
	case <-time.After(testing.LongWait):
		c.Fatalf("timed out waiting for unlock")
	}

	err = workertest.CheckKill(c, w)
	c.Check(err, jc.ErrorIs, nil)
}

func (s *workerSuite) TestWatchUpgradeCompletedNotFound(c *gc.C) {
	defer s.setupMocks(c).Finish()

	// Ensure that the update hasn't already happened.
	s.lock.EXPECT().IsUnlocked().Return(false)

	cfg := s.getConfig()

	// Walk through the upgrade process:
	//  - Create Upgrade, but it's already started.
	//  - Get the active upgrade.
	//  - Get the upgrade info and returns not found.
	//  - Cause the worker to bounce.

	done := make(chan struct{})

	srv := s.upgradeService.EXPECT()
	srv.CreateUpgrade(gomock.Any(), cfg.FromVersion, cfg.ToVersion).Return(domainupgrade.UUID(""), upgradeerrors.ErrUpgradeAlreadyStarted)
	srv.ActiveUpgrade(gomock.Any()).Return(s.upgradeUUID, nil)
	srv.UpgradeInfo(gomock.Any(), s.upgradeUUID).DoAndReturn(func(ctx context.Context, uuid domainupgrade.UUID) (upgrade.Info, error) {
		defer close(done)
		return upgrade.Info{State: upgrade.Created}, errors.NotFoundf("boom")
	})

	w, err := NewUpgradeDatabaseWorker(cfg)
	c.Assert(err, jc.ErrorIsNil)
	defer workertest.DirtyKill(c, w)

	select {
	case <-done:
	case <-time.After(testing.LongWait):
		c.Fatalf("timed out waiting for unlock")
	}

	err = workertest.CheckKill(c, w)
	c.Check(err, jc.ErrorIs, dependency.ErrBounce)
}

func (s *workerSuite) TestWatchUpgradeCompletedInErrorState(c *gc.C) {
	defer s.setupMocks(c).Finish()

	// Ensure that the update hasn't already happened.
	s.lock.EXPECT().IsUnlocked().Return(false)

	cfg := s.getConfig()

	// Walk through the upgrade process:
	//  - Create Upgrade, but it's already started.
	//  - Get the active upgrade.
	//  - Get the upgrade info and ensure it's not in an error state.
	//  - Stop the worker, requires manual intervention.

	done := make(chan struct{})

	srv := s.upgradeService.EXPECT()
	srv.CreateUpgrade(gomock.Any(), cfg.FromVersion, cfg.ToVersion).Return(domainupgrade.UUID(""), upgradeerrors.ErrUpgradeAlreadyStarted)
	srv.ActiveUpgrade(gomock.Any()).Return(s.upgradeUUID, nil)
	srv.UpgradeInfo(gomock.Any(), s.upgradeUUID).DoAndReturn(func(ctx context.Context, uuid domainupgrade.UUID) (upgrade.Info, error) {
		defer close(done)
		return upgrade.Info{State: upgrade.Error}, nil
	})

	w, err := NewUpgradeDatabaseWorker(cfg)
	c.Assert(err, jc.ErrorIsNil)
	defer workertest.DirtyKill(c, w)

	select {
	case <-done:
	case <-time.After(testing.LongWait):
		c.Fatalf("timed out waiting for unlock")
	}

	err = workertest.CheckKill(c, w)
	c.Check(err, jc.ErrorIs, nil)
}

func (s *workerSuite) TestWatchUpgradeFailed(c *gc.C) {
	defer s.setupMocks(c).Finish()

	// Ensure that the update hasn't already happened.
	s.lock.EXPECT().IsUnlocked().Return(false)

	cfg := s.getConfig()

	chCompleted := make(chan struct{})
	chFailed := make(chan struct{})

	completedWatcher := watchertest.NewMockNotifyWatcher(chCompleted)
	defer workertest.DirtyKill(c, completedWatcher)

	failedWatcher := watchertest.NewMockNotifyWatcher(chFailed)
	defer workertest.DirtyKill(c, failedWatcher)

	// Walk through the upgrade process:
	//  - Create Upgrade, but it's already started.
	//  - Get the active upgrade.
	//  - Get the upgrade info and ensure it's not in an error state.
	//  - Watch for the upgrade to be completed.
	//  - Watch for the upgrade to be failed, but do not act upon it.
	//  - Ensure that we _don't_ unlock the lock.

	sync := make(chan struct{})

	srv := s.upgradeService.EXPECT()
	srv.CreateUpgrade(gomock.Any(), cfg.FromVersion, cfg.ToVersion).Return(domainupgrade.UUID(""), upgradeerrors.ErrUpgradeAlreadyStarted)
	srv.ActiveUpgrade(gomock.Any()).Return(s.upgradeUUID, nil)
	srv.UpgradeInfo(gomock.Any(), s.upgradeUUID).Return(upgrade.Info{State: upgrade.Created}, nil)
	srv.SetControllerReady(gomock.Any(), s.upgradeUUID, "0").Return(nil)

	srv.WatchForUpgradeState(gomock.Any(), s.upgradeUUID, upgrade.DBCompleted).Return(completedWatcher, nil)
	srv.WatchForUpgradeState(gomock.Any(), s.upgradeUUID, upgrade.Error).DoAndReturn(func(ctx context.Context, uuid domainupgrade.UUID, state upgrade.State) (watcher.Watcher[struct{}], error) {
		defer close(sync)
		return failedWatcher, nil
	})

	w, err := NewUpgradeDatabaseWorker(cfg)
	c.Assert(err, jc.ErrorIsNil)
	defer workertest.DirtyKill(c, w)

	// Dispatch the initial event.
	s.dispatchChange(c, chCompleted)
	s.dispatchChange(c, chFailed)

	select {
	case <-sync:
	case <-time.After(testing.LongWait):
		c.Fatalf("timed out waiting for watcher to respond")
	}

	s.dispatchChange(c, chFailed)

	// Wait for the events to be consumed.
	<-time.After(time.Second)

	err = workertest.CheckKill(c, w)
	c.Check(err, jc.ErrorIs, dependency.ErrBounce)
}

func (s *workerSuite) TestWatchUpgradeError(c *gc.C) {
	defer s.setupMocks(c).Finish()

	// Ensure that the update hasn't already happened.
	s.lock.EXPECT().IsUnlocked().Return(false)

	cfg := s.getConfig()

	ch := make(chan struct{})

	watcher := watchertest.NewMockNotifyWatcher(ch)
	defer workertest.DirtyKill(c, watcher)

	// Walk through the upgrade process:
	//  - Create Upgrade, but it's already started.
	//  - Get the active upgrade, but it doesn't exist.

	srv := s.upgradeService.EXPECT()
	srv.CreateUpgrade(gomock.Any(), cfg.FromVersion, cfg.ToVersion).Return(domainupgrade.UUID(""), upgradeerrors.ErrUpgradeAlreadyStarted)
	srv.ActiveUpgrade(gomock.Any()).Return(s.upgradeUUID, errors.NotFoundf("no upgrade"))

	w, err := NewUpgradeDatabaseWorker(cfg)
	c.Assert(err, jc.ErrorIsNil)
	defer workertest.DirtyKill(c, w)

	err = workertest.CheckKill(c, w)
	c.Check(err, jc.ErrorIs, dependency.ErrBounce)
}

func (s *workerSuite) TestUpgradeController(c *gc.C) {
	defer s.setupMocks(c).Finish()

	// Ensure that the update hasn't already happened.
	s.lock.EXPECT().IsUnlocked().Return(false)

	cfg := s.getConfig()

	ch := make(chan struct{})

	watcher := watchertest.NewMockNotifyWatcher(ch)
	defer workertest.DirtyKill(c, watcher)

	// Walk through the upgrade process:
	//  - Create Upgrade.
	//  - Set the controller ready for upgrade.
	//  - Wait for the upgrade to be ready. This means, all the controller nodes
	//    are synced and ready to be upgraded.
	//  - Start the upgrade, we're the leader.
	//  - Upgrade the controller db.
	//  - Set the db upgrade complete.
	//  - Unlock the lock.

	s.expectStartUpgrade(cfg.FromVersion, cfg.ToVersion, watcher)

	// Controller upgrade.
	s.expectControllerDBUpgrade()

	// Model upgrade (there are no models).
	s.expectModelList([]model.UUID{})

	s.expectDBCompleted()
	done := s.expectUnlock()

	w, err := NewUpgradeDatabaseWorker(cfg)
	c.Assert(err, jc.ErrorIsNil)
	defer workertest.DirtyKill(c, w)

	// Dispatch the initial event.
	s.dispatchChange(c, ch)
	s.dispatchChange(c, ch)

	select {
	case <-done:
	case <-time.After(testing.LongWait):
		c.Fatalf("timed out waiting for unlock")
	}

	err = workertest.CheckKill(c, w)
	c.Check(err, jc.ErrorIs, dependency.ErrUninstall)
}

func (s *workerSuite) TestUpgradeControllerThatIsAlreadyUpgraded(c *gc.C) {
	defer s.setupMocks(c).Finish()

	// Ensure that the update hasn't already happened.
	s.lock.EXPECT().IsUnlocked().Return(false)

	cfg := s.getConfig()

	ch := make(chan struct{})

	watcher := watchertest.NewMockNotifyWatcher(ch)
	defer workertest.DirtyKill(c, watcher)

	// Walk through the upgrade process:
	//  - Create Upgrade.
	//  - Set the controller ready for upgrade.
	//  - Wait for the upgrade to be ready. This means, all the controller nodes
	//    are synced and ready to be upgraded.
	//  - Start the upgrade, we're the leader.
	//  - Upgrade the controller db.
	//  - Set the db upgrade complete.
	//  - Unlock the lock.

	s.expectStartUpgrade(cfg.FromVersion, cfg.ToVersion, watcher)

	// Controller upgrade.
	//  - Upgrade the controller db and re-run the upgrades to ensure that they
	//    don't break in the worker.

	schema := schema.ControllerDDL()
	_, err := schema.Ensure(context.Background(), s.TxnRunner())
	c.Assert(err, jc.ErrorIsNil)

	s.expectControllerDBUpgrade()

	// Model upgrade (there are no models).
	s.expectModelList([]model.UUID{})

	s.expectDBCompleted()
	done := s.expectUnlock()

	w, err := NewUpgradeDatabaseWorker(cfg)
	c.Assert(err, jc.ErrorIsNil)
	defer workertest.DirtyKill(c, w)

	// Dispatch the initial event.
	s.dispatchChange(c, ch)
	s.dispatchChange(c, ch)

	select {
	case <-done:
	case <-time.After(testing.LongWait):
		c.Fatalf("timed out waiting for unlock")
	}

	err = workertest.CheckKill(c, w)
	c.Check(err, jc.ErrorIs, dependency.ErrUninstall)
}

func (s *workerSuite) TestUpgradeModels(c *gc.C) {
	defer s.setupMocks(c).Finish()

	// Ensure that the update hasn't already happened.
	s.lock.EXPECT().IsUnlocked().Return(false)

	cfg := s.getConfig()

	ch := make(chan struct{})

	watcher := watchertest.NewMockNotifyWatcher(ch)
	defer workertest.DirtyKill(c, watcher)

	// Walk through the upgrade process:
	//  - Create Upgrade.
	//  - Set the controller ready for upgrade.
	//  - Wait for the upgrade to be ready. This means, all the controller nodes
	//    are synced and ready to be upgraded.
	//  - Start the upgrade, we're the leader.
	//  - Upgrade the controller db.
	//  - Upgrade all the model dbs.
	//  - Set the db upgrade complete.
	//  - Unlock the lock.

	s.expectStartUpgrade(cfg.FromVersion, cfg.ToVersion, watcher)

	// Controller upgrade.
	s.expectControllerDBUpgrade()

	// Model upgrade.
	modelUUID := model.UUID(utils.MustNewUUID().String())
	s.expectModelList([]model.UUID{modelUUID})
	s.expectModelDBUpgrade(c, modelUUID)

	s.expectDBCompleted()
	done := s.expectUnlock()

	w, err := NewUpgradeDatabaseWorker(cfg)
	c.Assert(err, jc.ErrorIsNil)
	defer workertest.DirtyKill(c, w)

	// Dispatch the initial event.
	s.dispatchChange(c, ch)
	s.dispatchChange(c, ch)

	select {
	case <-done:
	case <-time.After(testing.LongWait):
		c.Fatalf("timed out waiting for unlock")
	}

	err = workertest.CheckKill(c, w)
	c.Check(err, jc.ErrorIs, dependency.ErrUninstall)
}

func (s *workerSuite) TestUpgradeModelsThatIsAlreadyUpgraded(c *gc.C) {
	defer s.setupMocks(c).Finish()

	// Ensure that the update hasn't already happened.
	s.lock.EXPECT().IsUnlocked().Return(false)

	cfg := s.getConfig()

	ch := make(chan struct{})

	watcher := watchertest.NewMockNotifyWatcher(ch)
	defer workertest.CheckKill(c, watcher)

	// Walk through the upgrade process:
	//  - Create Upgrade.
	//  - Set the controller ready for upgrade.
	//  - Wait for the upgrade to be ready. This means, all the controller nodes
	//    are synced and ready to be upgraded.
	//  - Start the upgrade, we're the leader.
	//  - Upgrade the controller db.
	//  - Upgrade all the model dbs.
	//  - Set the db upgrade complete.
	//  - Unlock the lock.

	s.expectStartUpgrade(cfg.FromVersion, cfg.ToVersion, watcher)

	// Controller upgrade.
	s.expectControllerDBUpgrade()

	// Model upgrade.
	modelUUID := model.UUID(utils.MustNewUUID().String())
	s.expectModelList([]model.UUID{modelUUID})
	txnRunner := s.expectModelDBUpgrade(c, modelUUID)

	// Run the upgrade steps on the existing model, to ensure it doesn't break
	// in the worker.
	schema := schema.ModelDDL()
	_, err := schema.Ensure(context.Background(), txnRunner)
	c.Assert(err, jc.ErrorIsNil)

	s.expectDBCompleted()
	done := s.expectUnlock()

	w, err := NewUpgradeDatabaseWorker(cfg)
	c.Assert(err, jc.ErrorIsNil)
	defer workertest.DirtyKill(c, w)

	// Dispatch the initial event.
	s.dispatchChange(c, ch)
	s.dispatchChange(c, ch)

	select {
	case <-done:
	case <-time.After(testing.LongWait):
		c.Fatalf("timed out waiting for unlock")
	}

	err = workertest.CheckKill(c, w)
	c.Check(err, jc.ErrorIs, dependency.ErrUninstall)
}

func (s *workerSuite) TestUpgradeFailsWhenKilled(c *gc.C) {
	defer s.setupMocks(c).Finish()

	// Ensure that the update hasn't already happened.
	s.lock.EXPECT().IsUnlocked().Return(false)

	cfg := s.getConfig()

	ch := make(chan struct{})

	watcher := watchertest.NewMockNotifyWatcher(ch)
	defer workertest.CheckKill(c, watcher)

	// Walk through the upgrade process:
	//  - Create Upgrade.
	//  - Watch for the upgrade ready
	//  - Dispatch the initial event.
	//  - Set the controller ready, but kill the worker at the same time.
	//  - Ensure that kill the worker also sets the upgrade to failed.

	done := make(chan struct{})
	kill := make(chan worker.Worker)

	srv := s.upgradeService.EXPECT()
	srv.CreateUpgrade(gomock.Any(), cfg.FromVersion, cfg.ToVersion).Return(s.upgradeUUID, nil)
	srv.WatchForUpgradeReady(gomock.Any(), s.upgradeUUID).Return(watcher, nil)
	srv.SetControllerReady(gomock.Any(), s.upgradeUUID, "0").DoAndReturn(func(ctx context.Context, uuid domainupgrade.UUID, controllerID string) error {
		select {
		case w := <-kill:
			defer close(done)
			w.Kill()
		case <-time.After(testing.LongWait):
			c.Fatalf("timed out waiting for kill")
		}
		return nil
	})
	srv.SetDBUpgradeFailed(gomock.Any(), s.upgradeUUID).Return(nil)

	w, err := NewUpgradeDatabaseWorker(cfg)
	c.Assert(err, jc.ErrorIsNil)
	defer workertest.DirtyKill(c, w)

	// Dispatch the initial event.
	s.dispatchChange(c, ch)

	select {
	case kill <- w:
	case <-time.After(testing.LongWait):
		c.Fatalf("timed out waiting for kill")
	}

	select {
	case <-done:
	case <-time.After(testing.LongWait):
		c.Fatalf("timed out waiting for done")
	}

	err = workertest.CheckKill(c, w)
	c.Assert(err, jc.ErrorIsNil)
}

func (s *workerSuite) getConfig() Config {
	return Config{
		DBUpgradeCompleteLock: s.lock,
		Agent:                 s.agent,
		Logger:                s.logger,
		Clock:                 clock.WallClock,
		UpgradeService:        s.upgradeService,
		ModelManagerService:   s.modelManagerService,
		DBGetter:              s.dbGetter,
		FromVersion:           version.MustParse("3.0.0"),
		ToVersion:             version.MustParse("6.6.6"),
		Tag:                   names.NewMachineTag("0"),
	}
}

func (s *workerSuite) setupMocks(c *gc.C) *gomock.Controller {
	ctrl := s.baseSuite.setupMocks(c)

	s.upgradeUUID = domainupgrade.UUID(utils.MustNewUUID().String())

	return ctrl
}

func (s *workerSuite) expectStartUpgrade(from, to version.Number, watcher worker.Worker) {
	srv := s.upgradeService.EXPECT()
	srv.CreateUpgrade(gomock.Any(), from, to).Return(s.upgradeUUID, nil)
	srv.SetControllerReady(gomock.Any(), s.upgradeUUID, "0").Return(nil)
	srv.WatchForUpgradeReady(gomock.Any(), s.upgradeUUID).Return(watcher, nil)
	srv.StartUpgrade(gomock.Any(), s.upgradeUUID).Return(nil)
}

func (s *workerSuite) expectDBCompleted() {
	srv := s.upgradeService.EXPECT()
	srv.SetDBUpgradeCompleted(gomock.Any(), s.upgradeUUID).Return(nil)
}

func (s *workerSuite) expectControllerDBUpgrade() {
	s.dbGetter.EXPECT().GetDB(coredatabase.ControllerNS).Return(s.TxnRunner(), nil)
}

func (s *workerSuite) expectModelList(models []model.UUID) {
	s.modelManagerService.EXPECT().ModelList(gomock.Any()).Return(models, nil)
}

func (s *workerSuite) expectModelDBUpgrade(c *gc.C, modelUUID model.UUID) coredatabase.TxnRunner {
	txnRunner, _ := s.OpenDB(c)
	s.dbGetter.EXPECT().GetDB(modelUUID.String()).Return(txnRunner, nil)
	return txnRunner
}

func (s *workerSuite) expectUnlock() chan struct{} {
	done := make(chan struct{})
	s.lock.EXPECT().Unlock().DoAndReturn(func() {
		close(done)
	})
	return done
}

func (s *workerSuite) dispatchChange(c *gc.C, ch chan struct{}) {
	// Send initial event.
	select {
	case ch <- struct{}{}:
	case <-time.After(testing.ShortWait):
		c.Fatalf("timed out waiting to enqueue change")
	}
}
