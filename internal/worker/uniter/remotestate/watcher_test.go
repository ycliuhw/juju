// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package remotestate_test

import (
	"time"

	"github.com/juju/clock/testclock"
	"github.com/juju/names/v5"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/worker/v4"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/core/life"
	"github.com/juju/juju/core/model"
	"github.com/juju/juju/core/secrets"
	"github.com/juju/juju/core/watcher"
	loggertesting "github.com/juju/juju/internal/logger/testing"
	"github.com/juju/juju/internal/worker/uniter/remotestate"
	"github.com/juju/juju/rpc/params"
	"github.com/juju/juju/testing"
	coretesting "github.com/juju/juju/testing"
)

type WatcherSuite struct {
	coretesting.BaseSuite

	modelType                    model.ModelType
	sidecar                      bool
	enforcedCharmModifiedVersion int
	uniterClient                 *mockUniterClient
	leadership                   *mockLeadershipTracker
	watcher                      *remotestate.RemoteStateWatcher
	clock                        *testclock.Clock

	secretsClient              *mockSecretsClient
	rotateSecretWatcherEvent   chan string
	expireRevisionWatcherEvent chan string

	applicationWatcher   *mockNotifyWatcher
	runningStatusWatcher *mockNotifyWatcher
	running              *remotestate.ContainerRunningStatus

	workloadEventChannel chan string
	shutdownChannel      chan bool
}

type WatcherSuiteIAAS struct {
	WatcherSuite
}

type WatcherSuiteCAAS struct {
	WatcherSuite
}

type WatcherSuiteSidecar struct {
	WatcherSuite
}

type WatcherSuiteSidecarCharmModVer struct {
	WatcherSuiteSidecar
}

var _ = gc.Suite(&WatcherSuiteIAAS{
	WatcherSuite{modelType: model.IAAS},
})
var _ = gc.Suite(&WatcherSuiteCAAS{
	WatcherSuite{modelType: model.CAAS},
})

var _ = gc.Suite(&WatcherSuiteSidecar{
	WatcherSuite{
		modelType:                    model.CAAS,
		sidecar:                      true,
		enforcedCharmModifiedVersion: 5,
	},
})

var _ = gc.Suite(&WatcherSuiteSidecarCharmModVer{
	WatcherSuiteSidecar{
		WatcherSuite{
			modelType: model.CAAS,
			sidecar:   true,
			// Use a different version than the base tests
			enforcedCharmModifiedVersion: 4,
		},
	},
})

func (s *WatcherSuite) SetUpTest(c *gc.C) {
	s.BaseSuite.SetUpTest(c)
	s.uniterClient = &mockUniterClient{
		modelType: s.modelType,
		unit: mockUnit{
			tag:  names.NewUnitTag("mysql/0"),
			life: life.Alive,
			application: mockApplication{
				tag:                   names.NewApplicationTag("mysql"),
				life:                  life.Alive,
				curl:                  "ch:trusty/mysql",
				charmModifiedVersion:  5,
				leaderSettingsWatcher: newMockNotifyWatcher(),
			},
			unitWatcher:                      newMockNotifyWatcher(),
			addressesWatcher:                 newMockStringsWatcher(),
			configSettingsWatcher:            newMockStringsWatcher(),
			applicationConfigSettingsWatcher: newMockStringsWatcher(),
			storageWatcher:                   newMockStringsWatcher(),
			actionWatcher:                    newMockStringsWatcher(),
			relationsWatcher:                 newMockStringsWatcher(),
			instanceDataWatcher:              newMockNotifyWatcher(),
		},
		relations:                   make(map[names.RelationTag]*mockRelation),
		storageAttachment:           make(map[params.StorageAttachmentId]params.StorageAttachment),
		relationUnitsWatchers:       make(map[names.RelationTag]*mockRelationUnitsWatcher),
		relationAppWatchers:         make(map[names.RelationTag]map[string]*mockNotifyWatcher),
		storageAttachmentWatchers:   make(map[names.StorageTag]*mockNotifyWatcher),
		updateStatusInterval:        5 * time.Minute,
		updateStatusIntervalWatcher: newMockNotifyWatcher(),
	}

	s.leadership = &mockLeadershipTracker{
		claimTicket:  mockTicket{make(chan struct{}, 1), true},
		leaderTicket: mockTicket{make(chan struct{}, 1), true},
		minionTicket: mockTicket{make(chan struct{}, 1), true},
	}

	s.rotateSecretWatcherEvent = make(chan string)
	s.secretsClient = &mockSecretsClient{
		secretsWatcher:          newMockStringsWatcher(),
		secretsRevisionsWatcher: newMockStringsWatcher(),
	}

	s.clock = testclock.NewClock(time.Now())

	s.workloadEventChannel = make(chan string)
	s.shutdownChannel = make(chan bool)
}

func (s *WatcherSuiteIAAS) SetUpTest(c *gc.C) {
	s.WatcherSuite.SetUpTest(c)

	s.uniterClient.unit.application.applicationWatcher = newMockNotifyWatcher()
	s.applicationWatcher = s.uniterClient.unit.application.applicationWatcher
	s.uniterClient.unit.upgradeSeriesWatcher = newMockNotifyWatcher()
	s.uniterClient.unit.instanceDataWatcher = newMockNotifyWatcher()

	w, err := remotestate.NewWatcher(s.setupWatcherConfig(c))
	c.Assert(err, jc.ErrorIsNil)

	s.watcher = w
}

func (s *WatcherSuiteCAAS) SetUpTest(c *gc.C) {
	s.WatcherSuite.SetUpTest(c)
	s.runningStatusWatcher = newMockNotifyWatcher()
	s.uniterClient.unit.application.applicationWatcher = newMockNotifyWatcher()
	s.applicationWatcher = s.uniterClient.unit.application.applicationWatcher

	cfg := s.setupWatcherConfig(c)
	cfg.ContainerRunningStatusChannel = s.runningStatusWatcher.Changes()
	cfg.ContainerRunningStatusFunc = func(providerID string) (*remotestate.ContainerRunningStatus, error) {
		return s.running, nil
	}

	w, err := remotestate.NewWatcher(cfg)
	c.Assert(err, jc.ErrorIsNil)

	s.watcher = w
}

func (s *WatcherSuiteSidecar) SetUpTest(c *gc.C) {
	s.WatcherSuite.SetUpTest(c)

	s.uniterClient.unit.application.applicationWatcher = newMockNotifyWatcher()
	s.applicationWatcher = s.uniterClient.unit.application.applicationWatcher

	w, err := remotestate.NewWatcher(s.setupWatcherConfig(c))
	c.Assert(err, jc.ErrorIsNil)

	s.watcher = w
}

func (s *WatcherSuite) setupWatcherConfig(c *gc.C) remotestate.WatcherConfig {
	statusTicker := func(wait time.Duration) remotestate.Waiter {
		return dummyWaiter{s.clock.After(wait)}
	}
	return remotestate.WatcherConfig{
		Logger:                       loggertesting.WrapCheckLog(c),
		UniterClient:                 s.uniterClient,
		ModelType:                    s.modelType,
		Sidecar:                      s.sidecar,
		EnforcedCharmModifiedVersion: s.enforcedCharmModifiedVersion,
		LeadershipTracker:            s.leadership,
		SecretsClient:                s.secretsClient,
		SecretRotateWatcherFunc: func(u names.UnitTag, isLeader bool, rotateCh chan []string) (worker.Worker, error) {
			select {
			case s.rotateSecretWatcherEvent <- u.Id():
			default:
			}
			rotateSecretWatcher := &mockSecretTriggerWatcher{
				ch:     rotateCh,
				stopCh: make(chan struct{}),
			}
			return rotateSecretWatcher, nil
		},
		SecretExpiryWatcherFunc: func(u names.UnitTag, isLeader bool, expireCh chan []string) (worker.Worker, error) {
			select {
			case s.expireRevisionWatcherEvent <- u.Id():
			default:
			}
			expireRevisionWatcher := &mockSecretTriggerWatcher{
				ch:     expireCh,
				stopCh: make(chan struct{}),
			}
			return expireRevisionWatcher, nil
		},
		UnitTag:              s.uniterClient.unit.tag,
		UpdateStatusChannel:  statusTicker,
		CanApplyCharmProfile: s.modelType == model.IAAS,
		WorkloadEventChannel: s.workloadEventChannel,
		ShutdownChannel:      s.shutdownChannel,
	}
}

type dummyWaiter struct {
	c <-chan time.Time
}

func (w dummyWaiter) After() <-chan time.Time {
	return w.c
}

func (s *WatcherSuite) TearDownTest(c *gc.C) {
	if s.watcher != nil {
		s.watcher.Kill()
		err := s.watcher.Wait()
		c.Assert(err, jc.ErrorIsNil)
	}
}

func (s *WatcherSuiteIAAS) TestInitialSnapshot(c *gc.C) {
	snap := s.watcher.Snapshot()
	c.Assert(snap, jc.DeepEquals, remotestate.Snapshot{
		Relations:               map[int]remotestate.RelationSnapshot{},
		Storage:                 map[names.StorageTag]remotestate.StorageSnapshot{},
		ActionChanged:           map[string]int{},
		UpgradeMachineStatus:    model.UpgradeSeriesNotStarted,
		ConsumedSecretInfo:      map[string]secrets.SecretRevisionInfo{},
		ObsoleteSecretRevisions: map[string][]int{},
	})
}

func (s *WatcherSuiteCAAS) TestInitialSnapshot(c *gc.C) {
	snap := s.watcher.Snapshot()
	c.Assert(snap, jc.DeepEquals, remotestate.Snapshot{
		Relations:               map[int]remotestate.RelationSnapshot{},
		Storage:                 map[names.StorageTag]remotestate.StorageSnapshot{},
		ActionChanged:           map[string]int{},
		ActionsBlocked:          true,
		UpgradeMachineStatus:    model.UpgradeSeriesNotStarted,
		ConsumedSecretInfo:      map[string]secrets.SecretRevisionInfo{},
		ObsoleteSecretRevisions: map[string][]int{},
	})
}

func (s *WatcherSuiteSidecar) TestInitialSnapshot(c *gc.C) {
	snap := s.watcher.Snapshot()
	c.Assert(snap, jc.DeepEquals, remotestate.Snapshot{
		Relations:               map[int]remotestate.RelationSnapshot{},
		Storage:                 map[names.StorageTag]remotestate.StorageSnapshot{},
		ActionChanged:           map[string]int{},
		UpgradeMachineStatus:    model.UpgradeSeriesNotStarted,
		ConsumedSecretInfo:      map[string]secrets.SecretRevisionInfo{},
		ObsoleteSecretRevisions: map[string][]int{},
	})
}

func (s *WatcherSuite) TestInitialSignal(c *gc.C) {
	// There should not be a remote state change until
	// we've seen all of the top-level notifications.
	s.uniterClient.unit.unitWatcher.changes <- struct{}{}
	assertNoNotifyEvent(c, s.watcher.RemoteStateChanged(), "remote state change")
	s.uniterClient.unit.addressesWatcher.changes <- []string{"addresseshash"}
	s.uniterClient.unit.configSettingsWatcher.changes <- []string{"confighash"}
	s.uniterClient.unit.applicationConfigSettingsWatcher.changes <- []string{"trusthash"}
	if s.uniterClient.unit.upgradeSeriesWatcher != nil {
		s.uniterClient.unit.upgradeSeriesWatcher.changes <- struct{}{}
	}
	if s.uniterClient.unit.instanceDataWatcher != nil {
		s.uniterClient.unit.instanceDataWatcher.changes <- struct{}{}
	}
	s.uniterClient.unit.storageWatcher.changes <- []string{}
	s.uniterClient.unit.actionWatcher.changes <- []string{}
	if s.uniterClient.unit.application.applicationWatcher != nil {
		s.uniterClient.unit.application.applicationWatcher.changes <- struct{}{}
	}
	s.uniterClient.unit.application.leaderSettingsWatcher.changes <- struct{}{}
	s.uniterClient.unit.relationsWatcher.changes <- []string{}
	s.uniterClient.updateStatusIntervalWatcher.changes <- struct{}{}
	s.leadership.claimTicket.ch <- struct{}{}
	s.secretsClient.secretsWatcher.changes <- []string{}
	s.secretsClient.secretsRevisionsWatcher.changes <- []string{}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
}

func (s *WatcherSuite) signalAll() {
	s.uniterClient.unit.unitWatcher.changes <- struct{}{}
	s.uniterClient.unit.configSettingsWatcher.changes <- []string{"confighash"}
	s.uniterClient.unit.applicationConfigSettingsWatcher.changes <- []string{"trusthash"}
	s.uniterClient.unit.actionWatcher.changes <- []string{}
	s.uniterClient.unit.application.leaderSettingsWatcher.changes <- struct{}{}
	s.uniterClient.unit.relationsWatcher.changes <- []string{}
	s.uniterClient.unit.addressesWatcher.changes <- []string{"addresseshash"}
	s.uniterClient.updateStatusIntervalWatcher.changes <- struct{}{}
	s.leadership.claimTicket.ch <- struct{}{}
	s.uniterClient.unit.storageWatcher.changes <- []string{}
	s.applicationWatcher.changes <- struct{}{}
	s.secretsClient.secretsWatcher.changes <- []string{}
	if s.uniterClient.modelType == model.IAAS {
		s.uniterClient.unit.upgradeSeriesWatcher.changes <- struct{}{}
		s.uniterClient.unit.instanceDataWatcher.changes <- struct{}{}
	}
}

func (s *WatcherSuite) TestSnapshot(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	snap := s.watcher.Snapshot()
	c.Assert(snap, jc.DeepEquals, remotestate.Snapshot{
		Life:                    s.uniterClient.unit.life,
		Relations:               map[int]remotestate.RelationSnapshot{},
		Storage:                 map[names.StorageTag]remotestate.StorageSnapshot{},
		ActionChanged:           map[string]int{},
		CharmModifiedVersion:    s.uniterClient.unit.application.charmModifiedVersion,
		CharmURL:                s.uniterClient.unit.application.curl,
		ForceCharmUpgrade:       s.uniterClient.unit.application.forceUpgrade,
		ResolvedMode:            s.uniterClient.unit.resolved,
		ConfigHash:              "confighash",
		TrustHash:               "trusthash",
		AddressesHash:           "addresseshash",
		LeaderSettingsVersion:   1,
		Leader:                  true,
		UpgradeMachineStatus:    model.UpgradeSeriesPrepareStarted,
		UpgradeMachineTarget:    "ubuntu@20.04",
		ConsumedSecretInfo:      map[string]secrets.SecretRevisionInfo{},
		ObsoleteSecretRevisions: map[string][]int{},
	})
}

func (s *WatcherSuiteSidecar) TestSnapshot(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	snap := s.watcher.Snapshot()
	c.Assert(snap, jc.DeepEquals, remotestate.Snapshot{
		Life:                    s.uniterClient.unit.life,
		Relations:               map[int]remotestate.RelationSnapshot{},
		Storage:                 map[names.StorageTag]remotestate.StorageSnapshot{},
		ActionChanged:           map[string]int{},
		CharmModifiedVersion:    s.uniterClient.unit.application.charmModifiedVersion,
		CharmURL:                s.uniterClient.unit.application.curl,
		ForceCharmUpgrade:       s.uniterClient.unit.application.forceUpgrade,
		ResolvedMode:            s.uniterClient.unit.resolved,
		ConfigHash:              "confighash",
		TrustHash:               "trusthash",
		AddressesHash:           "addresseshash",
		LeaderSettingsVersion:   1,
		Leader:                  true,
		UpgradeMachineStatus:    model.UpgradeSeriesNotStarted,
		ConsumedSecretInfo:      map[string]secrets.SecretRevisionInfo{},
		ObsoleteSecretRevisions: map[string][]int{},
	})
}

func (s *WatcherSuiteCAAS) TestSnapshot(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	snap := s.watcher.Snapshot()
	c.Assert(snap, jc.DeepEquals, remotestate.Snapshot{
		Life:                    s.uniterClient.unit.life,
		Relations:               map[int]remotestate.RelationSnapshot{},
		Storage:                 map[names.StorageTag]remotestate.StorageSnapshot{},
		CharmModifiedVersion:    s.uniterClient.unit.application.charmModifiedVersion,
		CharmURL:                s.uniterClient.unit.application.curl,
		ForceCharmUpgrade:       s.uniterClient.unit.application.forceUpgrade,
		ResolvedMode:            s.uniterClient.unit.resolved,
		ConfigHash:              "confighash",
		TrustHash:               "trusthash",
		AddressesHash:           "addresseshash",
		LeaderSettingsVersion:   1,
		Leader:                  true,
		UpgradeMachineStatus:    model.UpgradeSeriesNotStarted,
		ActionsBlocked:          true,
		ActionChanged:           map[string]int{},
		ContainerRunningStatus:  nil,
		ConsumedSecretInfo:      map[string]secrets.SecretRevisionInfo{},
		ObsoleteSecretRevisions: map[string][]int{},
	})

	t := time.Now()
	s.uniterClient.unit.providerID = "provider-id"
	s.running = &remotestate.ContainerRunningStatus{
		Initialising:     true,
		InitialisingTime: t,
		PodName:          "wow",
		Running:          false,
	}
	select {
	case s.runningStatusWatcher.changes <- struct{}{}:
	case <-time.After(coretesting.LongWait):
		c.Fatal("timeout waiting to post running status change")
	}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	snap = s.watcher.Snapshot()
	c.Assert(snap, jc.DeepEquals, remotestate.Snapshot{
		Life:                    s.uniterClient.unit.life,
		Relations:               map[int]remotestate.RelationSnapshot{},
		Storage:                 map[names.StorageTag]remotestate.StorageSnapshot{},
		CharmModifiedVersion:    s.uniterClient.unit.application.charmModifiedVersion,
		CharmURL:                s.uniterClient.unit.application.curl,
		ForceCharmUpgrade:       s.uniterClient.unit.application.forceUpgrade,
		ResolvedMode:            s.uniterClient.unit.resolved,
		ConfigHash:              "confighash",
		TrustHash:               "trusthash",
		AddressesHash:           "addresseshash",
		LeaderSettingsVersion:   1,
		Leader:                  true,
		UpgradeMachineStatus:    model.UpgradeSeriesNotStarted,
		ActionsBlocked:          true,
		ActionChanged:           map[string]int{},
		ProviderID:              s.uniterClient.unit.providerID,
		ContainerRunningStatus:  s.running,
		ConsumedSecretInfo:      map[string]secrets.SecretRevisionInfo{},
		ObsoleteSecretRevisions: map[string][]int{},
	})

	s.running = &remotestate.ContainerRunningStatus{
		Initialising:     false,
		InitialisingTime: t,
		PodName:          "wow",
		Running:          true,
	}
	select {
	case s.runningStatusWatcher.changes <- struct{}{}:
	case <-time.After(coretesting.LongWait):
		c.Fatal("timeout waiting to post running status change")
	}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	snap = s.watcher.Snapshot()
	c.Assert(snap, jc.DeepEquals, remotestate.Snapshot{
		Life:                    s.uniterClient.unit.life,
		Relations:               map[int]remotestate.RelationSnapshot{},
		Storage:                 map[names.StorageTag]remotestate.StorageSnapshot{},
		CharmModifiedVersion:    s.uniterClient.unit.application.charmModifiedVersion,
		CharmURL:                s.uniterClient.unit.application.curl,
		ForceCharmUpgrade:       s.uniterClient.unit.application.forceUpgrade,
		ResolvedMode:            s.uniterClient.unit.resolved,
		ConfigHash:              "confighash",
		TrustHash:               "trusthash",
		AddressesHash:           "addresseshash",
		LeaderSettingsVersion:   1,
		Leader:                  true,
		UpgradeMachineStatus:    model.UpgradeSeriesNotStarted,
		ActionsBlocked:          false,
		ActionChanged:           map[string]int{},
		ProviderID:              s.uniterClient.unit.providerID,
		ContainerRunningStatus:  s.running,
		ConsumedSecretInfo:      map[string]secrets.SecretRevisionInfo{},
		ObsoleteSecretRevisions: map[string][]int{},
	})
}

func (s *WatcherSuite) TestRemoteStateChanged(c *gc.C) {
	assertOneChange := func() {
		assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
		assertNoNotifyEvent(c, s.watcher.RemoteStateChanged(), "remote state change")
	}

	s.signalAll()
	assertOneChange()
	initial := s.watcher.Snapshot()

	s.uniterClient.unit.life = life.Dying
	s.uniterClient.unit.unitWatcher.changes <- struct{}{}
	assertOneChange()
	c.Assert(s.watcher.Snapshot().Life, gc.Equals, life.Dying)

	s.uniterClient.unit.resolved = params.ResolvedRetryHooks
	s.uniterClient.unit.unitWatcher.changes <- struct{}{}
	assertOneChange()
	c.Assert(s.watcher.Snapshot().ResolvedMode, gc.Equals, params.ResolvedRetryHooks)

	s.uniterClient.unit.addressesWatcher.changes <- []string{"addresseshash2"}
	assertOneChange()
	c.Assert(s.watcher.Snapshot().AddressesHash, gc.Equals, "addresseshash2")

	s.uniterClient.unit.storageWatcher.changes <- []string{}
	assertOneChange()

	rotateWatcher := remotestate.SecretRotateWatcher(s.watcher).(*mockSecretTriggerWatcher)
	secretURIs := []string{"secret:999e2mr0ui3e8a215n4g", "secret:9m4e2mr0ui3e8a215n4g", "secret:8b4e2mr1wi3e8a215n5h"}
	rotateWatcher.ch <- secretURIs
	assertOneChange()
	c.Assert(s.watcher.Snapshot().SecretRotations, jc.DeepEquals, secretURIs)

	expireWatcher := remotestate.SecretExpiryWatcherFunc(s.watcher).(*mockSecretTriggerWatcher)
	secretRevisions := []string{"secret:999e2mr0ui3e8a215n4g/666", "secret:9m4e2mr0ui3e8a215n4g/667", "secret:8b4e2mr1wi3e8a215n5h/668"}
	expireWatcher.ch <- secretRevisions
	assertOneChange()
	c.Assert(s.watcher.Snapshot().ExpiredSecretRevisions, jc.DeepEquals, secretRevisions)

	s.secretsClient.secretsWatcher.changes <- secretURIs
	assertOneChange()
	c.Assert(s.watcher.Snapshot().ConsumedSecretInfo, jc.DeepEquals, map[string]secrets.SecretRevisionInfo{
		"secret:9m4e2mr0ui3e8a215n4g": {
			LatestRevision: 666,
			Label:          "label-secret:9m4e2mr0ui3e8a215n4g",
		},
		"secret:8b4e2mr1wi3e8a215n5h": {
			LatestRevision: 667,
			Label:          "label-secret:8b4e2mr1wi3e8a215n5h",
		},
	})
	c.Assert(s.watcher.Snapshot().DeletedSecrets, jc.DeepEquals, []string{"secret:999e2mr0ui3e8a215n4g"})

	s.secretsClient.secretsRevisionsWatcher.changes <- []string{"secret:9m4e2mr0ui3e8a215n4g/666", "secret:9m4e2mr0ui3e8a215n4g/668", "secret:666e2mr0ui3e8a215n4g"}
	assertOneChange()
	c.Assert(s.watcher.Snapshot().ObsoleteSecretRevisions, jc.DeepEquals, map[string][]int{
		"secret:9m4e2mr0ui3e8a215n4g": {666, 668},
	})
	c.Assert(s.watcher.Snapshot().DeletedSecrets, jc.DeepEquals, []string{"secret:666e2mr0ui3e8a215n4g", "secret:999e2mr0ui3e8a215n4g"})

	s.uniterClient.unit.configSettingsWatcher.changes <- []string{"confighash2"}
	assertOneChange()
	c.Assert(s.watcher.Snapshot().ConfigHash, gc.Equals, "confighash2")

	s.uniterClient.unit.applicationConfigSettingsWatcher.changes <- []string{"trusthash2"}
	assertOneChange()
	c.Assert(s.watcher.Snapshot().TrustHash, gc.Equals, "trusthash2")

	s.uniterClient.unit.application.leaderSettingsWatcher.changes <- struct{}{}
	assertOneChange()
	c.Assert(s.watcher.Snapshot().LeaderSettingsVersion, gc.Equals, initial.LeaderSettingsVersion+1)

	s.uniterClient.unit.relationsWatcher.changes <- []string{}
	assertOneChange()

	if s.modelType == model.IAAS {
		s.uniterClient.unit.upgradeSeriesWatcher.changes <- struct{}{}
		assertOneChange()
		s.uniterClient.unit.instanceDataWatcher.changes <- struct{}{}
		assertOneChange()
	}
	s.uniterClient.unit.application.forceUpgrade = true
	s.applicationWatcher.changes <- struct{}{}
	assertOneChange()
	c.Assert(s.watcher.Snapshot().ForceCharmUpgrade, jc.IsTrue)

	s.clock.Advance(5 * time.Minute)
	assertOneChange()
}

func (s *WatcherSuite) TestActionsReceived(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	s.uniterClient.unit.actionWatcher.changes <- []string{"an-action"}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	snapshot := s.watcher.Snapshot()
	c.Assert(snapshot.ActionsPending, gc.DeepEquals, []string{"an-action"})
	c.Assert(snapshot.ActionChanged["an-action"], gc.NotNil)
}

func (s *WatcherSuite) TestActionsReceivedWithChanges(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	s.uniterClient.unit.actionWatcher.changes <- []string{"an-action"}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	snapshot := s.watcher.Snapshot()
	c.Assert(snapshot.ActionsPending, gc.DeepEquals, []string{"an-action"})
	c.Assert(snapshot.ActionChanged["an-action"], gc.Equals, 0)

	s.uniterClient.unit.actionWatcher.changes <- []string{"an-action"}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	snapshot = s.watcher.Snapshot()
	c.Assert(snapshot.ActionsPending, gc.DeepEquals, []string{"an-action"})
	c.Assert(snapshot.ActionChanged["an-action"], gc.Equals, 1)
}

func (s *WatcherSuite) TestClearResolvedMode(c *gc.C) {
	s.uniterClient.unit.resolved = params.ResolvedRetryHooks
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	snap := s.watcher.Snapshot()
	c.Assert(snap.ResolvedMode, gc.Equals, params.ResolvedRetryHooks)

	s.watcher.ClearResolvedMode()
	snap = s.watcher.Snapshot()
	c.Assert(snap.ResolvedMode, gc.Equals, params.ResolvedNone)
}

func (s *WatcherSuite) TestLeadershipChanged(c *gc.C) {
	s.leadership.claimTicket.result = false
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().Leader, jc.IsFalse)

	s.leadership.leaderTicket.ch <- struct{}{}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().Leader, jc.IsTrue)

	s.leadership.minionTicket.ch <- struct{}{}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().Leader, jc.IsFalse)
}

func (s *WatcherSuite) TestLeadershipMinionUnchanged(c *gc.C) {
	s.leadership.claimTicket.result = false
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	// Initially minion, so triggering minion should have no effect.
	s.leadership.minionTicket.ch <- struct{}{}
	assertNoNotifyEvent(c, s.watcher.RemoteStateChanged(), "remote state change")
}

func (s *WatcherSuite) TestLeadershipLeaderUnchanged(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	// Initially leader, so triggering leader should have no effect.
	s.leadership.leaderTicket.ch <- struct{}{}
	assertNoNotifyEvent(c, s.watcher.RemoteStateChanged(), "remote state change")
}

func (s *WatcherSuite) TestStorageChanged(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	storageTag0 := names.NewStorageTag("blob/0")
	storageAttachmentId0 := params.StorageAttachmentId{
		UnitTag:    s.uniterClient.unit.tag.String(),
		StorageTag: storageTag0.String(),
	}
	storageTag0Watcher := newMockNotifyWatcher()
	s.uniterClient.storageAttachmentWatchers[storageTag0] = storageTag0Watcher
	s.uniterClient.storageAttachment[storageAttachmentId0] = params.StorageAttachment{
		UnitTag:    storageAttachmentId0.UnitTag,
		StorageTag: storageAttachmentId0.StorageTag,
		Life:       life.Alive,
		Kind:       params.StorageKindUnknown, // unprovisioned
		Location:   "nowhere",
	}

	storageTag1 := names.NewStorageTag("blob/1")
	storageAttachmentId1 := params.StorageAttachmentId{
		UnitTag:    s.uniterClient.unit.tag.String(),
		StorageTag: storageTag1.String(),
	}
	storageTag1Watcher := newMockNotifyWatcher()
	s.uniterClient.storageAttachmentWatchers[storageTag1] = storageTag1Watcher
	s.uniterClient.storageAttachment[storageAttachmentId1] = params.StorageAttachment{
		UnitTag:    storageAttachmentId1.UnitTag,
		StorageTag: storageAttachmentId1.StorageTag,
		Life:       life.Dying,
		Kind:       params.StorageKindBlock,
		Location:   "malta",
	}

	// We should not see any event until the storage attachment watchers
	// return their initial events.
	s.uniterClient.unit.storageWatcher.changes <- []string{"blob/0", "blob/1"}
	assertNoNotifyEvent(c, s.watcher.RemoteStateChanged(), "remote state change")
	storageTag0Watcher.changes <- struct{}{}
	storageTag1Watcher.changes <- struct{}{}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	c.Assert(s.watcher.Snapshot().Storage, jc.DeepEquals, map[names.StorageTag]remotestate.StorageSnapshot{
		storageTag0: {
			Life: life.Alive,
		},
		storageTag1: {
			Life:     life.Dying,
			Kind:     params.StorageKindBlock,
			Attached: true,
			Location: "malta",
		},
	})

	s.uniterClient.storageAttachment[storageAttachmentId0] = params.StorageAttachment{
		UnitTag:    storageAttachmentId0.UnitTag,
		StorageTag: storageAttachmentId0.StorageTag,
		Life:       life.Dying,
		Kind:       params.StorageKindFilesystem,
		Location:   "somewhere",
	}
	delete(s.uniterClient.storageAttachment, storageAttachmentId1)
	storageTag0Watcher.changes <- struct{}{}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	s.uniterClient.unit.storageWatcher.changes <- []string{"blob/1"}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().Storage, jc.DeepEquals, map[names.StorageTag]remotestate.StorageSnapshot{
		storageTag0: {
			Life:     life.Dying,
			Attached: true,
			Kind:     params.StorageKindFilesystem,
			Location: "somewhere",
		},
	})
}

func (s *WatcherSuite) TestStorageUnattachedChanged(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	storageTag0 := names.NewStorageTag("blob/0")
	storageAttachmentId0 := params.StorageAttachmentId{
		UnitTag:    s.uniterClient.unit.tag.String(),
		StorageTag: storageTag0.String(),
	}
	storageTag0Watcher := newMockNotifyWatcher()
	s.uniterClient.storageAttachmentWatchers[storageTag0] = storageTag0Watcher
	s.uniterClient.storageAttachment[storageAttachmentId0] = params.StorageAttachment{
		UnitTag:    storageAttachmentId0.UnitTag,
		StorageTag: storageAttachmentId0.StorageTag,
		Life:       life.Alive,
		Kind:       params.StorageKindUnknown, // unprovisioned
	}

	s.uniterClient.unit.storageWatcher.changes <- []string{"blob/0"}
	storageTag0Watcher.changes <- struct{}{}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	c.Assert(s.watcher.Snapshot().Storage, jc.DeepEquals, map[names.StorageTag]remotestate.StorageSnapshot{
		storageTag0: {
			Life: life.Alive,
		},
	})

	s.uniterClient.storageAttachment[storageAttachmentId0] = params.StorageAttachment{
		UnitTag:    storageAttachmentId0.UnitTag,
		StorageTag: storageAttachmentId0.StorageTag,
		Life:       life.Dying,
	}
	// The storage is still unattached; triggering the storage-specific
	// watcher should not cause any event to be emitted.
	storageTag0Watcher.changes <- struct{}{}
	assertNoNotifyEvent(c, s.watcher.RemoteStateChanged(), "remote state change")
	s.uniterClient.unit.storageWatcher.changes <- []string{"blob/0"}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().Storage, jc.DeepEquals, map[names.StorageTag]remotestate.StorageSnapshot{
		storageTag0: {
			Life: life.Dying,
		},
	})
}

func (s *WatcherSuite) TestStorageAttachmentRemoved(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	storageTag0 := names.NewStorageTag("blob/0")
	storageAttachmentId0 := params.StorageAttachmentId{
		UnitTag:    s.uniterClient.unit.tag.String(),
		StorageTag: storageTag0.String(),
	}
	storageTag0Watcher := newMockNotifyWatcher()
	s.uniterClient.storageAttachmentWatchers[storageTag0] = storageTag0Watcher
	s.uniterClient.storageAttachment[storageAttachmentId0] = params.StorageAttachment{
		UnitTag:    storageAttachmentId0.UnitTag,
		StorageTag: storageAttachmentId0.StorageTag,
		Life:       life.Dying,
		Kind:       params.StorageKindUnknown, // unprovisioned
	}

	s.uniterClient.unit.storageWatcher.changes <- []string{"blob/0"}
	storageTag0Watcher.changes <- struct{}{}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	c.Assert(s.watcher.Snapshot().Storage, jc.DeepEquals, map[names.StorageTag]remotestate.StorageSnapshot{
		storageTag0: {
			Life: life.Dying,
		},
	})

	// Removing the storage attachment and then triggering the storage-
	// specific watcher should not cause an event to be emitted, but it
	// will cause that watcher to stop running. Triggering the top-level
	// storage watcher will remove it and update the snapshot.
	delete(s.uniterClient.storageAttachment, storageAttachmentId0)
	storageTag0Watcher.changes <- struct{}{}
	assertNoNotifyEvent(c, s.watcher.RemoteStateChanged(), "remote state change")
	c.Assert(storageTag0Watcher.Stopped(), jc.IsTrue)
	s.uniterClient.unit.storageWatcher.changes <- []string{"blob/0"}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().Storage, gc.HasLen, 0)
}

func (s *WatcherSuite) TestStorageChangedNotFoundInitially(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	// blob/0 is initially in state, but is removed between the
	// watcher signal and the uniter querying it. This should
	// not cause the watcher to raise an error.
	s.uniterClient.unit.storageWatcher.changes <- []string{"blob/0"}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().Storage, gc.HasLen, 0)
}

func (s *WatcherSuite) TestRelationsChanged(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	relationTag := names.NewRelationTag("mysql:peer")
	s.uniterClient.relations[relationTag] = &mockRelation{
		tag: relationTag, id: 123, life: life.Alive, suspended: false,
	}
	s.uniterClient.relationUnitsWatchers[relationTag] = newMockRelationUnitsWatcher()
	s.uniterClient.unit.relationsWatcher.changes <- []string{relationTag.Id()}
	s.uniterClient.relationAppWatchers[relationTag] = map[string]*mockNotifyWatcher{"mysql": newMockNotifyWatcher()}

	// There should not be any signal until the relation units watcher has
	// returned its initial event also.
	assertNoNotifyEvent(c, s.watcher.RemoteStateChanged(), "remote state change")
	s.uniterClient.relationUnitsWatchers[relationTag].changes <- watcher.RelationUnitsChange{
		Changed:    map[string]watcher.UnitSettings{"mysql/1": {1}, "mysql/2": {2}},
		AppChanged: map[string]int64{"mysql": 1},
	}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(
		s.watcher.Snapshot().Relations,
		jc.DeepEquals,
		map[int]remotestate.RelationSnapshot{
			123: {
				Life:               life.Alive,
				Suspended:          false,
				Members:            map[string]int64{"mysql/1": 1, "mysql/2": 2},
				ApplicationMembers: map[string]int64{"mysql": 1},
			},
		},
	)

	// If a relation is known, then updating it does not require any input
	// from the relation units watcher.
	s.uniterClient.relations[relationTag].life = life.Dying
	s.uniterClient.unit.relationsWatcher.changes <- []string{relationTag.Id()}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().Relations[123].Life, gc.Equals, life.Dying)

	// If a relation is not found, then it should be removed from the
	// snapshot and its relation units watcher stopped.
	delete(s.uniterClient.relations, relationTag)
	s.uniterClient.unit.relationsWatcher.changes <- []string{relationTag.Id()}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().Relations, gc.HasLen, 0)
	c.Assert(s.uniterClient.relationUnitsWatchers[relationTag].Stopped(), jc.IsTrue)
}

func (s *WatcherSuite) TestRelationsSuspended(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	relationTag := names.NewRelationTag("mysql:db wordpress:db")
	s.uniterClient.relations[relationTag] = &mockRelation{
		tag: relationTag, id: 123, life: life.Alive, suspended: false,
	}
	s.uniterClient.relationUnitsWatchers[relationTag] = newMockRelationUnitsWatcher()
	s.uniterClient.unit.relationsWatcher.changes <- []string{relationTag.Id()}
	assertNoNotifyEvent(c, s.watcher.RemoteStateChanged(), "remote state change")
	s.uniterClient.relationUnitsWatchers[relationTag].changes <- watcher.RelationUnitsChange{
		Changed:    map[string]watcher.UnitSettings{"mysql/1": {1}, "mysql/2": {2}},
		AppChanged: map[string]int64{"mysql": 1},
	}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	s.uniterClient.relations[relationTag].suspended = true
	s.uniterClient.unit.relationsWatcher.changes <- []string{relationTag.Id()}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().Relations[123].Suspended, jc.IsTrue)
	c.Assert(s.uniterClient.relationUnitsWatchers[relationTag].Stopped(), jc.IsTrue)
}

func (s *WatcherSuite) TestRelationUnitsChanged(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	relationTag := names.NewRelationTag("mysql:peer")
	s.uniterClient.relations[relationTag] = &mockRelation{
		tag: relationTag, id: 123, life: life.Alive,
	}
	s.uniterClient.relationUnitsWatchers[relationTag] = newMockRelationUnitsWatcher()

	s.uniterClient.unit.relationsWatcher.changes <- []string{relationTag.Id()}
	s.uniterClient.relationUnitsWatchers[relationTag].changes <- watcher.RelationUnitsChange{
		Changed:    map[string]watcher.UnitSettings{"mysql/1": {1}},
		AppChanged: map[string]int64{"mysql": 1},
	}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	s.uniterClient.relationUnitsWatchers[relationTag].changes <- watcher.RelationUnitsChange{
		Changed: map[string]watcher.UnitSettings{"mysql/1": {2}, "mysql/2": {1}},
	}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert( // Members is updated
		s.watcher.Snapshot().Relations[123].Members,
		jc.DeepEquals,
		map[string]int64{"mysql/1": 2, "mysql/2": 1},
	)
	c.Assert( // ApplicationMembers doesn't change
		s.watcher.Snapshot().Relations[123].ApplicationMembers,
		jc.DeepEquals,
		map[string]int64{"mysql": 1},
	)

	s.uniterClient.relationUnitsWatchers[relationTag].changes <- watcher.RelationUnitsChange{
		AppChanged: map[string]int64{"mysql": 2},
	}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert( // Members doesn't change
		s.watcher.Snapshot().Relations[123].Members,
		jc.DeepEquals,
		map[string]int64{"mysql/1": 2, "mysql/2": 1},
	)
	c.Assert( // But ApplicationMembers is updated
		s.watcher.Snapshot().Relations[123].ApplicationMembers,
		jc.DeepEquals,
		map[string]int64{"mysql": 2},
	)

	s.uniterClient.relationUnitsWatchers[relationTag].changes <- watcher.RelationUnitsChange{
		Departed: []string{"mysql/1", "mysql/42"},
	}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(
		s.watcher.Snapshot().Relations[123].Members,
		jc.DeepEquals,
		map[string]int64{"mysql/2": 1},
	)
}

func (s *WatcherSuite) TestRelationUnitsDontLeakReferences(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	relationTag := names.NewRelationTag("mysql:peer")
	s.uniterClient.relations[relationTag] = &mockRelation{
		tag: relationTag, id: 123, life: life.Alive,
	}
	s.uniterClient.relationUnitsWatchers[relationTag] = newMockRelationUnitsWatcher()

	s.uniterClient.unit.relationsWatcher.changes <- []string{relationTag.Id()}
	s.uniterClient.relationUnitsWatchers[relationTag].changes <- watcher.RelationUnitsChange{
		Changed: map[string]watcher.UnitSettings{"mysql/1": {1}},
	}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	snapshot := s.watcher.Snapshot()
	snapshot.Relations[123].Members["pwned"] = 2600
	c.Assert(
		s.watcher.Snapshot().Relations[123].Members,
		jc.DeepEquals,
		map[string]int64{"mysql/1": 1},
	)
}

func (s *WatcherSuite) TestUpdateStatusTicker(c *gc.C) {
	s.signalAll()
	initial := s.watcher.Snapshot()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	// Advance the clock past the trigger time.
	s.waitAlarmsStable(c)
	s.clock.Advance(5 * time.Minute)
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().UpdateStatusVersion, gc.Equals, initial.UpdateStatusVersion+1)

	// Advance again but not past the trigger time.
	s.waitAlarmsStable(c)
	s.clock.Advance(4 * time.Minute)
	assertNoNotifyEvent(c, s.watcher.RemoteStateChanged(), "unexpected remote state change")
	c.Assert(s.watcher.Snapshot().UpdateStatusVersion, gc.Equals, initial.UpdateStatusVersion+1)

	// And we hit the trigger time.
	s.clock.Advance(1 * time.Minute)
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().UpdateStatusVersion, gc.Equals, initial.UpdateStatusVersion+2)
}

func (s *WatcherSuite) TestUpdateStatusIntervalChanges(c *gc.C) {
	s.signalAll()
	initial := s.watcher.Snapshot()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	// Advance the clock past the trigger time.
	s.waitAlarmsStable(c)
	s.clock.Advance(5 * time.Minute)
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().UpdateStatusVersion, gc.Equals, initial.UpdateStatusVersion+1)

	// Change the update status interval to 10 seconds.
	s.uniterClient.updateStatusInterval = 10 * time.Second
	s.uniterClient.updateStatusIntervalWatcher.changes <- struct{}{}

	// Advance 10 seconds; the timer should be triggered.
	s.waitAlarmsStable(c)
	s.clock.Advance(10 * time.Second)
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().UpdateStatusVersion, gc.Equals, initial.UpdateStatusVersion+2)
}

// waitAlarmsStable is used to wait until the remote watcher's loop has
// stopped churning (at least for testing.ShortWait), so that we can
// then Advance the clock with some confidence that the SUT really is
// waiting for it. This seems likely to be more stable than waiting for
// a specific number of loop iterations; it's currently 9, but waiting
// for a specific number is very likely to start failing intermittently
// again, as in lp:1604955, if the SUT undergoes even subtle changes.
func (s *WatcherSuite) waitAlarmsStable(c *gc.C) {
	timeout := time.After(coretesting.LongWait)
	for i := 0; ; i++ {
		c.Logf("waiting for alarm %d", i)
		select {
		case <-s.clock.Alarms():
		case <-time.After(coretesting.ShortWait):
			return
		case <-timeout:
			c.Fatalf("never stopped setting alarms")
		}
	}
}

func (s *WatcherSuiteSidecar) TestWatcherConfig(c *gc.C) {
	_, err := remotestate.NewWatcher(remotestate.WatcherConfig{
		ModelType: model.IAAS,
		Sidecar:   true,
		Logger:    loggertesting.WrapCheckLog(c),
	})
	c.Assert(err, gc.ErrorMatches, `sidecar mode is only for "caas" model`)

	_, err = remotestate.NewWatcher(remotestate.WatcherConfig{
		ModelType: model.CAAS,
		Sidecar:   true,
		Logger:    loggertesting.WrapCheckLog(c),
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *WatcherSuite) TestWatcherConfigMissingLogger(c *gc.C) {
	_, err := remotestate.NewWatcher(remotestate.WatcherConfig{})
	c.Assert(err, gc.ErrorMatches, "nil Logger not valid")
}

func (s *WatcherSuiteSidecarCharmModVer) TestRemoteStateChanged(c *gc.C) {
	assertOneChange := func() {
		assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
		assertNoNotifyEvent(c, s.watcher.RemoteStateChanged(), "remote state change")
	}

	s.signalAll()
	assertOneChange()
	initial := s.watcher.Snapshot()

	s.uniterClient.unit.life = life.Dying
	s.uniterClient.unit.unitWatcher.changes <- struct{}{}
	assertOneChange()
	c.Assert(s.watcher.Snapshot().Life, gc.Equals, life.Dying)

	s.uniterClient.unit.resolved = params.ResolvedRetryHooks
	s.uniterClient.unit.unitWatcher.changes <- struct{}{}
	assertOneChange()
	c.Assert(s.watcher.Snapshot().ResolvedMode, gc.Equals, params.ResolvedRetryHooks)

	s.uniterClient.unit.addressesWatcher.changes <- []string{"addresseshash2"}
	assertOneChange()
	c.Assert(s.watcher.Snapshot().AddressesHash, gc.Equals, "addresseshash2")

	s.uniterClient.unit.storageWatcher.changes <- []string{}
	assertOneChange()

	s.uniterClient.unit.configSettingsWatcher.changes <- []string{"confighash2"}
	assertOneChange()
	c.Assert(s.watcher.Snapshot().ConfigHash, gc.Equals, "confighash2")

	rotateWatcher := remotestate.SecretRotateWatcher(s.watcher).(*mockSecretTriggerWatcher)
	secretURIs := []string{"secret:999e2mr0ui3e8a215n4g", "secret:9m4e2mr0ui3e8a215n4g", "secret:8b4e2mr1wi3e8a215n5h"}
	rotateWatcher.ch <- secretURIs
	assertOneChange()
	c.Assert(s.watcher.Snapshot().SecretRotations, jc.DeepEquals, secretURIs)

	expireWatcher := remotestate.SecretExpiryWatcherFunc(s.watcher).(*mockSecretTriggerWatcher)
	secretRevisions := []string{"secret:999e2mr0ui3e8a215n4g/666", "secret:9m4e2mr0ui3e8a215n4g/667", "secret:8b4e2mr1wi3e8a215n5h/668"}
	expireWatcher.ch <- secretRevisions
	assertOneChange()
	c.Assert(s.watcher.Snapshot().ExpiredSecretRevisions, jc.DeepEquals, secretRevisions)

	s.secretsClient.secretsWatcher.changes <- secretURIs
	assertOneChange()
	c.Assert(s.watcher.Snapshot().ConsumedSecretInfo, jc.DeepEquals, map[string]secrets.SecretRevisionInfo{
		"secret:9m4e2mr0ui3e8a215n4g": {
			LatestRevision: 666,
			Label:          "label-secret:9m4e2mr0ui3e8a215n4g",
		},
		"secret:8b4e2mr1wi3e8a215n5h": {
			LatestRevision: 667,
			Label:          "label-secret:8b4e2mr1wi3e8a215n5h",
		},
	})
	c.Assert(s.watcher.Snapshot().DeletedSecrets, jc.DeepEquals, []string{"secret:999e2mr0ui3e8a215n4g"})

	s.secretsClient.secretsRevisionsWatcher.changes <- []string{"secret:9m4e2mr0ui3e8a215n4g/666", "secret:9m4e2mr0ui3e8a215n4g/668", "secret:666e2mr0ui3e8a215n4g"}
	assertOneChange()
	c.Assert(s.watcher.Snapshot().ObsoleteSecretRevisions, jc.DeepEquals, map[string][]int{
		"secret:9m4e2mr0ui3e8a215n4g": {666, 668},
	})
	c.Assert(s.watcher.Snapshot().DeletedSecrets, jc.DeepEquals, []string{"secret:666e2mr0ui3e8a215n4g", "secret:999e2mr0ui3e8a215n4g"})

	s.uniterClient.unit.applicationConfigSettingsWatcher.changes <- []string{"trusthash2"}
	assertOneChange()
	c.Assert(s.watcher.Snapshot().TrustHash, gc.Equals, "trusthash2")

	s.uniterClient.unit.application.leaderSettingsWatcher.changes <- struct{}{}
	assertOneChange()
	c.Assert(s.watcher.Snapshot().LeaderSettingsVersion, gc.Equals, initial.LeaderSettingsVersion+1)

	s.uniterClient.unit.relationsWatcher.changes <- []string{}
	assertOneChange()

	if s.modelType == model.IAAS {
		s.uniterClient.unit.upgradeSeriesWatcher.changes <- struct{}{}
		assertOneChange()
		s.uniterClient.unit.instanceDataWatcher.changes <- struct{}{}
		assertOneChange()
	}
	s.uniterClient.unit.application.forceUpgrade = true
	s.applicationWatcher.changes <- struct{}{}
	assertOneChange()

	// EnforcedCharmModifiedVersion prevents the charm upgrading if it isn't the right version.
	snapshot := s.watcher.Snapshot()
	c.Assert(snapshot.CharmModifiedVersion, gc.Equals, 0)
	c.Assert(snapshot.CharmURL, gc.Equals, "")
	c.Assert(snapshot.ForceCharmUpgrade, gc.Equals, false)

	s.clock.Advance(5 * time.Minute)
	assertOneChange()
}

func (s *WatcherSuiteSidecarCharmModVer) TestSnapshot(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	snap := s.watcher.Snapshot()
	c.Assert(snap, jc.DeepEquals, remotestate.Snapshot{
		Life:                    s.uniterClient.unit.life,
		Relations:               map[int]remotestate.RelationSnapshot{},
		Storage:                 map[names.StorageTag]remotestate.StorageSnapshot{},
		ActionChanged:           map[string]int{},
		CharmModifiedVersion:    0,
		CharmURL:                "",
		ForceCharmUpgrade:       false,
		ResolvedMode:            s.uniterClient.unit.resolved,
		ConfigHash:              "confighash",
		TrustHash:               "trusthash",
		AddressesHash:           "addresseshash",
		LeaderSettingsVersion:   1,
		Leader:                  true,
		UpgradeMachineStatus:    model.UpgradeSeriesNotStarted,
		ConsumedSecretInfo:      map[string]secrets.SecretRevisionInfo{},
		ObsoleteSecretRevisions: map[string][]int{},
	})
}

func (s *WatcherSuite) TestWorkloadSignal(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	snap := s.watcher.Snapshot()
	c.Assert(snap.WorkloadEvents, gc.HasLen, 0)

	select {
	case s.workloadEventChannel <- "0":
	case <-time.After(testing.ShortWait):
		c.Fatalf("timed out waiting to signal workload event channel")
	}

	// Adding same event twice shouldn't re-add it.
	select {
	case s.workloadEventChannel <- "0":
	case <-time.After(testing.ShortWait):
		c.Fatalf("timed out waiting to signal workload event channel")
	}

	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	snap = s.watcher.Snapshot()
	c.Assert(snap.WorkloadEvents, gc.DeepEquals, []string{"0"})

	s.watcher.WorkloadEventCompleted("0")
	snap = s.watcher.Snapshot()
	c.Assert(snap.WorkloadEvents, gc.HasLen, 0)
}

func (s *WatcherSuite) TestInitialWorkloadEventIDs(c *gc.C) {
	config := remotestate.WatcherConfig{
		InitialWorkloadEventIDs: []string{"a", "b", "c"},
		Logger:                  loggertesting.WrapCheckLog(c),
	}
	w, err := remotestate.NewWatcher(config)
	c.Assert(err, jc.ErrorIsNil)

	snapshot := w.Snapshot()
	c.Assert(snapshot.WorkloadEvents, gc.DeepEquals, []string{"a", "b", "c"})
}

func (s *WatcherSuite) TestShutdown(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	snap := s.watcher.Snapshot()
	c.Assert(snap.Shutdown, jc.IsFalse)

	select {
	case s.shutdownChannel <- true:
	case <-time.After(testing.ShortWait):
		c.Fatalf("timed out waiting to signal workload event channel")
	}

	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	snap = s.watcher.Snapshot()
	c.Assert(snap.Shutdown, jc.IsTrue)
}

func (s *WatcherSuite) TestRotateSecretsSignal(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	snap := s.watcher.Snapshot()
	c.Assert(snap.SecretRotations, gc.HasLen, 0)

	rotateWatcher := remotestate.SecretRotateWatcher(s.watcher).(*mockSecretTriggerWatcher)

	select {
	case rotateWatcher.ch <- []string{"secret:9m4e2mr0ui3e8a215n4g"}:
	case <-time.After(testing.ShortWait):
		c.Fatalf("timed out waiting to signal rotate secret channel")
	}

	// Need to synchronize here in case the goroutine receiving from the
	// channel processes the first event but not the second (in which case the
	// assertion at the bottom of this test sometimes failed).
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	// Adding same event twice shouldn't re-add it.
	select {
	case rotateWatcher.ch <- []string{"secret:9m4e2mr0ui3e8a215n4g"}:
	case <-time.After(testing.ShortWait):
		c.Fatalf("timed out waiting to signal rotate secret channel")
	}

	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	snap = s.watcher.Snapshot()
	c.Assert(snap.SecretRotations, gc.DeepEquals, []string{"secret:9m4e2mr0ui3e8a215n4g"})

	s.watcher.RotateSecretCompleted("secret:9m4e2mr0ui3e8a215n4g")
	snap = s.watcher.Snapshot()
	c.Assert(snap.SecretRotations, gc.HasLen, 0)
}

func (s *WatcherSuite) TestExpireSecretRevisionsSignal(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	snap := s.watcher.Snapshot()
	c.Assert(snap.ExpiredSecretRevisions, gc.HasLen, 0)

	expireWatcher := remotestate.SecretExpiryWatcherFunc(s.watcher).(*mockSecretTriggerWatcher)
	select {
	case expireWatcher.ch <- []string{"secret:9m4e2mr0ui3e8a215n4g/666"}:
	case <-time.After(testing.ShortWait):
		c.Fatalf("timed out waiting to signal expire secret channel")
	}

	// Need to synchronize here in case the goroutine receiving from the
	// channel processes the first event but not the second (in which case the
	// assertion at the bottom of this test sometimes failed).
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	// Adding same event twice shouldn't re-add it.
	select {
	case expireWatcher.ch <- []string{"secret:9m4e2mr0ui3e8a215n4g/666"}:
	case <-time.After(testing.ShortWait):
		c.Fatalf("timed out waiting to signal expire secret channel")
	}

	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	snap = s.watcher.Snapshot()
	c.Assert(snap.ExpiredSecretRevisions, gc.DeepEquals, []string{"secret:9m4e2mr0ui3e8a215n4g/666"})

	s.watcher.ExpireRevisionCompleted("secret:9m4e2mr0ui3e8a215n4g/666")
	snap = s.watcher.Snapshot()
	c.Assert(snap.ExpiredSecretRevisions, gc.HasLen, 0)
}

func (s *WatcherSuite) TestDeleteSecretSignal(c *gc.C) {
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	snap := s.watcher.Snapshot()
	c.Assert(snap.ExpiredSecretRevisions, gc.HasLen, 0)

	secretWatcher := s.secretsClient.secretsWatcher
	select {
	case secretWatcher.changes <- []string{"secret:9m4e2mr0ui3e8a215n4g"}:
	case <-time.After(testing.ShortWait):
		c.Fatalf("timed out waiting to signal secret channel")
	}

	// Need to synchronize here in case the goroutine receiving from the
	// channel processes the first event but not the second (in which case the
	// assertion at the bottom of this test sometimes failed).
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	// Adding same event twice shouldn't re-add it.
	select {
	case secretWatcher.changes <- []string{"secret:9m4e2mr0ui3e8a215n4g"}:
	case <-time.After(testing.ShortWait):
		c.Fatalf("timed out waiting to signal secret channel")
	}

	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")

	snap = s.watcher.Snapshot()
	c.Assert(snap.DeletedSecrets, gc.DeepEquals, []string{"secret:9m4e2mr0ui3e8a215n4g"})

	s.watcher.RemoveSecretsCompleted([]string{"secret:9m4e2mr0ui3e8a215n4g"})
	snap = s.watcher.Snapshot()
	c.Assert(snap.DeletedSecrets, gc.HasLen, 0)
}

func (s *WatcherSuite) TestLeaderRunsSecretTriggerWatchers(c *gc.C) {
	s.leadership.claimTicket.result = false
	s.signalAll()
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().Leader, jc.IsFalse)

	s.leadership.leaderTicket.ch <- struct{}{}

	select {
	case unitName := <-s.rotateSecretWatcherEvent:
		c.Assert(unitName, gc.Equals, "mysql/0")
	case <-time.After(2000 * testing.LongWait):
		c.Fatalf("timed out waiting to signal rotate secret channel")
	}

	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().Leader, jc.IsTrue)

	rotateWatcher := remotestate.SecretRotateWatcher(s.watcher).(*mockSecretTriggerWatcher)
	rotateWatcher.ch <- []string{"secret:8b4e2mr1wi3e8a215n5h"}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().SecretRotations, jc.DeepEquals, []string{"secret:8b4e2mr1wi3e8a215n5h"})

	expiryWatcher := remotestate.SecretExpiryWatcherFunc(s.watcher).(*mockSecretTriggerWatcher)
	expiryWatcher.ch <- []string{"secret:8b4e2mr1wi3e8a215n5h/666"}
	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().ExpiredSecretRevisions, jc.DeepEquals, []string{"secret:8b4e2mr1wi3e8a215n5h/666"})

	// When not a leader anymore, stop the worker.
	s.leadership.minionTicket.ch <- struct{}{}
	select {
	case <-rotateWatcher.stopCh:
	case <-time.After(2000 * testing.LongWait):
		c.Fatalf("timed out waiting to signal stop worker channel")
	}

	// When not a leader anymore, clear any pending secrets to be rotated/expired.
	c.Assert(s.watcher.Snapshot().SecretRotations, gc.HasLen, 0)
	c.Assert(s.watcher.Snapshot().ExpiredSecretRevisions, gc.HasLen, 0)

	assertNotifyEvent(c, s.watcher.RemoteStateChanged(), "waiting for remote state change")
	c.Assert(s.watcher.Snapshot().Leader, jc.IsFalse)
}
