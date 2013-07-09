// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"labix.org/v2/mgo"
	"launchpad.net/loggo"
	"launchpad.net/tomb"

	"launchpad.net/juju-core/environs/config"
	"launchpad.net/juju-core/errors"
	"launchpad.net/juju-core/instance"
	"launchpad.net/juju-core/state/watcher"
	"launchpad.net/juju-core/utils/set"
)

var watchLogger = loggo.GetLogger("juju.state.watch")

// NotifyWatcher generates signals when something changes, but it does not
// return any content for those changes
type NotifyWatcher interface {
	Stop() error
	Err() error
	Changes() <-chan struct{}
}

// commonWatcher is part of all client watchers.
type commonWatcher struct {
	st   *State
	tomb tomb.Tomb
}

// Stop stops the watcher, and returns any error encountered while running
// or shutting down.
func (w *commonWatcher) Stop() error {
	w.tomb.Kill(nil)
	return w.tomb.Wait()
}

// Err returns any error encountered while running or shutting down, or
// tomb.ErrStillAlive if the watcher is still running.
func (w *commonWatcher) Err() error {
	return w.tomb.Err()
}

// collect combines the effects of the one change, and any further changes read
// from more in the next 10ms. The result map describes the existence, or not,
// of every id observed to have changed. If a value is read from the supplied
// stop chan, collect returns false immediately.
func collect(one watcher.Change, more <-chan watcher.Change, stop <-chan struct{}) (map[string]bool, bool) {
	var count int
	result := map[string]bool{}
	handle := func(ch watcher.Change) {
		count++
		result[ch.Id.(string)] = ch.Revno != -1
	}
	handle(one)
	timeout := time.After(10 * time.Millisecond)
	for done := false; !done; {
		select {
		case <-stop:
			return nil, false
		case another := <-more:
			handle(another)
		case <-timeout:
			done = true
		}
	}
	watchLogger.Tracef("read %d events for %d documents", count, len(result))
	return result, true
}

func hasString(changes []string, name string) bool {
	for _, v := range changes {
		if v == name {
			return true
		}
	}
	return false
}

// LifecycleWatcher notifies about lifecycle changes for a set of entities of
// the same kind. The first event emitted will contain the ids of all non-Dead
// entities; subsequent events are emitted whenever one or more entities are
// added, or change their lifecycle state. After an entity is found to be
// Dead, no further event will include it.
type LifecycleWatcher struct {
	commonWatcher
	out chan []string
	// coll is the collection holding all interesting entities.
	coll *mgo.Collection
	// members is used to select the initial set of interesting entities.
	members D
	// filter is used to exclude events not affecting interesting entities.
	filter func(interface{}) bool
	// life holds the most recent known life states of interesting entities.
	life map[string]Life
}

// WatchServices returns a LifecycleWatcher that notifies of changes to
// the lifecycles of the services in the environment.
func (st *State) WatchServices() *LifecycleWatcher {
	return newLifecycleWatcher(st, st.services, nil, nil)
}

// WatchUnits returns a LifecycleWatcher that notifies of changes to the
// lifecycles of units of s.
func (s *Service) WatchUnits() *LifecycleWatcher {
	members := D{{"service", s.doc.Name}}
	prefix := s.doc.Name + "/"
	filter := func(id interface{}) bool {
		return strings.HasPrefix(id.(string), prefix)
	}
	return newLifecycleWatcher(s.st, s.st.units, members, filter)
}

// WatchRelations returns a LifecycleWatcher that notifies of changes to the
// lifecycles of relations involving s.
func (s *Service) WatchRelations() *LifecycleWatcher {
	members := D{{"endpoints.servicename", s.doc.Name}}
	prefix := s.doc.Name + ":"
	infix := " " + prefix
	filter := func(key interface{}) bool {
		k := key.(string)
		return strings.HasPrefix(k, prefix) || strings.Contains(k, infix)
	}
	return newLifecycleWatcher(s.st, s.st.relations, members, filter)
}

// WatchEnvironMachines returns a LifecycleWatcher that notifies of changes to
// the lifecycles of the machines (but not containers) in the environment.
func (st *State) WatchEnvironMachines() *LifecycleWatcher {
	members := D{{"containertype", ""}}
	filter := func(id interface{}) bool {
		return !strings.Contains(id.(string), "/")
	}
	return newLifecycleWatcher(st, st.machines, members, filter)
}

// WatchContainers returns a LifecycleWatcher that notifies of changes to the
// lifecycles of containers on a machine.
func (m *Machine) WatchContainers(ctype instance.ContainerType) *LifecycleWatcher {
	members := D{{"parent", m.doc.Id}}
	match := fmt.Sprintf("^%s/%s/%s$", m.doc.Id, ctype, numberSnippet)
	child := regexp.MustCompile(match)
	filter := func(key interface{}) bool {
		return child.MatchString(key.(string))
	}
	return newLifecycleWatcher(m.st, m.st.machines, members, filter)
}

func newLifecycleWatcher(st *State, coll *mgo.Collection, members D, filter func(key interface{}) bool) *LifecycleWatcher {
	w := &LifecycleWatcher{
		commonWatcher: commonWatcher{st: st},
		coll:          coll,
		members:       members,
		filter:        filter,
		life:          make(map[string]Life),
		out:           make(chan []string),
	}
	go func() {
		defer w.tomb.Done()
		defer close(w.out)
		w.tomb.Kill(w.loop())
	}()
	return w
}

type lifeDoc struct {
	Id   string `bson:"_id"`
	Life Life
}

var lifeFields = D{{"_id", 1}, {"life", 1}}

// Changes returns the event channel for the LifecycleWatcher.
func (w *LifecycleWatcher) Changes() <-chan []string {
	return w.out
}

func (w *LifecycleWatcher) initial() (ids *set.Strings, err error) {
	ids = &set.Strings{}
	var doc lifeDoc
	iter := w.coll.Find(w.members).Select(lifeFields).Iter()
	for iter.Next(&doc) {
		ids.Add(doc.Id)
		if doc.Life != Dead {
			w.life[doc.Id] = doc.Life
		}
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func (w *LifecycleWatcher) merge(ids *set.Strings, updates map[string]bool) error {
	// Separate ids into those thought to exist and those known to be removed.
	changed := []string{}
	latest := map[string]Life{}
	for id, exists := range updates {
		if exists {
			changed = append(changed, id)
		} else {
			latest[id] = Dead
		}
	}

	// Collect life states from ids thought to exist. Any that don't actually
	// exist are ignored (we'll hear about them in the next set of updates --
	// all that's actually happened in that situation is that the watcher
	// events have lagged a little behind reality).
	iter := w.coll.Find(D{{"_id", D{{"$in", changed}}}}).Select(lifeFields).Iter()
	var doc lifeDoc
	for iter.Next(&doc) {
		latest[doc.Id] = doc.Life
	}
	if err := iter.Err(); err != nil {
		return err
	}

	// Add to ids any whose life state is known to have changed.
	for id, newLife := range latest {
		gone := newLife == Dead
		oldLife, known := w.life[id]
		switch {
		case known && gone:
			delete(w.life, id)
		case !known && !gone:
			w.life[id] = newLife
		case known && newLife != oldLife:
			w.life[id] = newLife
		default:
			continue
		}
		ids.Add(id)
	}
	return nil
}

func (w *LifecycleWatcher) loop() (err error) {
	in := make(chan watcher.Change)
	w.st.watcher.WatchCollectionWithFilter(w.coll.Name, in, w.filter)
	defer w.st.watcher.UnwatchCollection(w.coll.Name, in)
	ids, err := w.initial()
	if err != nil {
		return err
	}
	out := w.out
	for {
		select {
		case <-w.tomb.Dying():
			return tomb.ErrDying
		case <-w.st.watcher.Dead():
			return watcher.MustErr(w.st.watcher)
		case ch := <-in:
			updates, ok := collect(ch, in, w.tomb.Dying())
			if !ok {
				return tomb.ErrDying
			}
			if err := w.merge(ids, updates); err != nil {
				return err
			}
			if !ids.IsEmpty() {
				out = w.out
			}
		case out <- ids.Values():
			ids = &set.Strings{}
			out = nil
		}
	}
	return nil
}

// MinUnitsWatcher notifies about MinUnits changes of the services requiring
// a minimum number of units to be alive. The first event returned by the
// watcher is the set of service names requiring a minimum number of units.
// Subsequent events are generated when a service increases MinUnits, or when
// one or more units belonging to a service are destroyed.
type MinUnitsWatcher struct {
	commonWatcher
	known map[string]int
	out   chan []string
}

func newMinUnitsWatcher(st *State) *MinUnitsWatcher {
	w := &MinUnitsWatcher{
		commonWatcher: commonWatcher{st: st},
		known:         make(map[string]int),
		out:           make(chan []string),
	}
	go func() {
		defer w.tomb.Done()
		defer close(w.out)
		w.tomb.Kill(w.loop())
	}()
	return w
}

func (st *State) WatchMinUnits() *MinUnitsWatcher {
	return newMinUnitsWatcher(st)
}

func (w *MinUnitsWatcher) initial() (*set.Strings, error) {
	serviceNames := new(set.Strings)
	doc := &minUnitsDoc{}
	iter := w.st.minUnits.Find(nil).Iter()
	for iter.Next(doc) {
		w.known[doc.ServiceName] = doc.Revno
		serviceNames.Add(doc.ServiceName)
	}
	return serviceNames, iter.Err()
}

func (w *MinUnitsWatcher) merge(serviceNames *set.Strings, change watcher.Change) error {
	serviceName := change.Id.(string)
	if change.Revno == -1 {
		delete(w.known, serviceName)
		serviceNames.Remove(serviceName)
		return nil
	}
	doc := minUnitsDoc{}
	if err := w.st.minUnits.FindId(serviceName).One(&doc); err != nil {
		return err
	}
	revno, known := w.known[serviceName]
	w.known[serviceName] = doc.Revno
	if !known || doc.Revno > revno {
		serviceNames.Add(serviceName)
	}
	return nil
}

func (w *MinUnitsWatcher) loop() (err error) {
	ch := make(chan watcher.Change)
	w.st.watcher.WatchCollection(w.st.minUnits.Name, ch)
	defer w.st.watcher.UnwatchCollection(w.st.minUnits.Name, ch)
	serviceNames, err := w.initial()
	if err != nil {
		return err
	}
	out := w.out
	for {
		select {
		case <-w.tomb.Dying():
			return tomb.ErrDying
		case change, ok := <-ch:
			if !ok {
				return watcher.MustErr(w.st.watcher)
			}
			if err = w.merge(serviceNames, change); err != nil {
				return err
			}
			if !serviceNames.IsEmpty() {
				out = w.out
			}
		case out <- serviceNames.Values():
			out = nil
			serviceNames = new(set.Strings)
		}
	}
	return nil
}

func (w *MinUnitsWatcher) Changes() <-chan []string {
	return w.out
}

// RelationScopeWatcher observes changes to the set of units
// in a particular relation scope.
type RelationScopeWatcher struct {
	commonWatcher
	prefix     string
	ignore     string
	knownUnits set.Strings
	out        chan *RelationScopeChange
}

// RelationScopeChange contains information about units that have
// entered or left a particular scope.
type RelationScopeChange struct {
	Entered []string
	Left    []string
}

func newRelationScopeWatcher(st *State, scope, ignore string) *RelationScopeWatcher {
	w := &RelationScopeWatcher{
		commonWatcher: commonWatcher{st: st},
		prefix:        scope + "#",
		ignore:        ignore,
		out:           make(chan *RelationScopeChange),
	}
	go func() {
		defer w.tomb.Done()
		defer close(w.out)
		w.tomb.Kill(w.loop())
	}()
	return w
}

// Changes returns a channel that will receive changes when units enter and
// leave a relation scope. The Entered field in the first event on the channel
// holds the initial state.
func (w *RelationScopeWatcher) Changes() <-chan *RelationScopeChange {
	return w.out
}

func (changes *RelationScopeChange) isEmpty() bool {
	return len(changes.Entered)+len(changes.Left) == 0
}

func (w *RelationScopeWatcher) mergeChange(changes *RelationScopeChange, ch watcher.Change) (err error) {
	doc := &relationScopeDoc{ch.Id.(string)}
	if !strings.HasPrefix(doc.Key, w.prefix) {
		return nil
	}
	name := doc.unitName()
	if name == w.ignore {
		return nil
	}
	if ch.Revno == -1 {
		if w.knownUnits.Contains(name) {
			changes.Left = append(changes.Left, name)
			w.knownUnits.Remove(name)
		}
		return nil
	}
	if !w.knownUnits.Contains(name) {
		changes.Entered = append(changes.Entered, name)
		w.knownUnits.Add(name)
	}
	return nil
}

func (w *RelationScopeWatcher) getInitialEvent() (initial *RelationScopeChange, err error) {
	changes := &RelationScopeChange{}
	docs := []relationScopeDoc{}
	sel := D{{"_id", D{{"$regex", "^" + w.prefix}}}}
	err = w.st.relationScopes.Find(sel).All(&docs)
	if err != nil {
		return nil, err
	}
	for _, doc := range docs {
		if name := doc.unitName(); name != w.ignore {
			changes.Entered = append(changes.Entered, name)
			w.knownUnits.Add(name)
		}
	}
	return changes, nil
}

func (w *RelationScopeWatcher) loop() error {
	ch := make(chan watcher.Change)
	w.st.watcher.WatchCollection(w.st.relationScopes.Name, ch)
	defer w.st.watcher.UnwatchCollection(w.st.relationScopes.Name, ch)
	changes, err := w.getInitialEvent()
	if err != nil {
		return err
	}
	out := w.out
	for {
		select {
		case <-w.st.watcher.Dead():
			return watcher.MustErr(w.st.watcher)
		case <-w.tomb.Dying():
			return tomb.ErrDying
		case c := <-ch:
			if err := w.mergeChange(changes, c); err != nil {
				return err
			}
			if !changes.isEmpty() {
				out = w.out
			}
		case out <- changes:
			changes = &RelationScopeChange{}
			out = nil
		}
	}
	return nil
}

// RelationUnitsWatcher sends notifications of units entering and leaving the
// scope of a RelationUnit, and changes to the settings of those units known
// to have entered.
type RelationUnitsWatcher struct {
	commonWatcher
	sw       *RelationScopeWatcher
	watching set.Strings
	updates  chan watcher.Change
	out      chan RelationUnitsChange
}

// RelationUnitsChange holds notifications of units entering and leaving the
// scope of a RelationUnit, and changes to the settings of those units known
// to have entered.
//
// When a counterpart first enters scope, it is/ noted in the Joined field,
// and its settings are noted in the Changed field. Subsequently, settings
// changes will be noted in the Changed field alone, until the couterpart
// leaves the scope; at that point, it will be noted in the Departed field,
// and no further events will be sent for that counterpart unit.
type RelationUnitsChange struct {
	Joined   []string
	Changed  map[string]UnitSettings
	Departed []string
}

// Watch returns a watcher that notifies of changes to conterpart units in
// the relation.
func (ru *RelationUnit) Watch() *RelationUnitsWatcher {
	return newRelationUnitsWatcher(ru)
}

func newRelationUnitsWatcher(ru *RelationUnit) *RelationUnitsWatcher {
	w := &RelationUnitsWatcher{
		commonWatcher: commonWatcher{st: ru.st},
		sw:            ru.WatchScope(),
		updates:       make(chan watcher.Change),
		out:           make(chan RelationUnitsChange),
	}
	go func() {
		defer w.finish()
		w.tomb.Kill(w.loop())
	}()
	return w
}

// Changes returns a channel that will receive the changes to
// counterpart units in a relation. The first event on the
// channel holds the initial state of the relation in its
// Joined and Changed fields.
func (w *RelationUnitsWatcher) Changes() <-chan RelationUnitsChange {
	return w.out
}

func (changes *RelationUnitsChange) empty() bool {
	return len(changes.Joined)+len(changes.Changed)+len(changes.Departed) == 0
}

// mergeSettings reads the relation settings node for the unit with the
// supplied id, and sets a value in the Changed field keyed on the unit's
// name. It returns the mgo/txn revision number of the settings node.
func (w *RelationUnitsWatcher) mergeSettings(changes *RelationUnitsChange, key string) (int64, error) {
	node, err := readSettings(w.st, key)
	if err != nil {
		return -1, err
	}
	name := (&relationScopeDoc{key}).unitName()
	settings := UnitSettings{node.txnRevno, node.Map()}
	if changes.Changed == nil {
		changes.Changed = map[string]UnitSettings{name: settings}
	} else {
		changes.Changed[name] = settings
	}
	return node.txnRevno, nil
}

// mergeScope starts and stops settings watches on the units entering and
// leaving the scope in the supplied RelationScopeChange event, and applies
// the expressed changes to the supplied RelationUnitsChange event.
func (w *RelationUnitsWatcher) mergeScope(changes *RelationUnitsChange, c *RelationScopeChange) error {
	for _, name := range c.Entered {
		key := w.sw.prefix + name
		revno, err := w.mergeSettings(changes, key)
		if err != nil {
			return err
		}
		changes.Joined = append(changes.Joined, name)
		changes.Departed = remove(changes.Departed, name)
		w.st.watcher.Watch(w.st.settings.Name, key, revno, w.updates)
		w.watching.Add(key)
	}
	for _, name := range c.Left {
		key := w.sw.prefix + name
		changes.Departed = append(changes.Departed, name)
		if changes.Changed != nil {
			delete(changes.Changed, name)
		}
		changes.Joined = remove(changes.Joined, name)
		w.st.watcher.Unwatch(w.st.settings.Name, key, w.updates)
		w.watching.Remove(key)
	}
	return nil
}

// remove removes s from strs and returns the modified slice.
func remove(strs []string, s string) []string {
	for i, v := range strs {
		if s == v {
			strs[i] = strs[len(strs)-1]
			return strs[:len(strs)-1]
		}
	}
	return strs
}

func (w *RelationUnitsWatcher) finish() {
	watcher.Stop(w.sw, &w.tomb)
	for _, watchedValue := range w.watching.Values() {
		w.st.watcher.Unwatch(w.st.settings.Name, watchedValue, w.updates)
	}
	close(w.updates)
	close(w.out)
	w.tomb.Done()
}

func (w *RelationUnitsWatcher) loop() (err error) {
	sentInitial := false
	changes := RelationUnitsChange{}
	out := w.out
	out = nil
	for {
		select {
		case <-w.st.watcher.Dead():
			return watcher.MustErr(w.st.watcher)
		case <-w.tomb.Dying():
			return tomb.ErrDying
		case c, ok := <-w.sw.Changes():
			if !ok {
				return watcher.MustErr(w.sw)
			}
			if err = w.mergeScope(&changes, c); err != nil {
				return err
			}
			if !sentInitial || !changes.empty() {
				out = w.out
			} else {
				out = nil
			}
		case c := <-w.updates:
			if _, err = w.mergeSettings(&changes, c.Id.(string)); err != nil {
				return err
			}
			out = w.out
		case out <- changes:
			sentInitial = true
			changes = RelationUnitsChange{}
			out = nil
		}
	}
	panic("unreachable")
}

// UnitsWatcher notifies of changes to a set of units. Notifications will be
// sent when units enter or leave the set, and when units in the set change
// their lifecycle status. The initial event contains all units in the set,
// regardless of lifecycle status; once a unit observed to be Dead or removed
// has been reported, it will not be reported again.
type UnitsWatcher struct {
	commonWatcher
	tag      string
	getUnits func() ([]string, error)
	life     map[string]Life
	in       chan watcher.Change
	out      chan []string
}

// WatchSubordinateUnits returns a UnitsWatcher tracking the unit's subordinate units.
func (u *Unit) WatchSubordinateUnits() *UnitsWatcher {
	u = &Unit{st: u.st, doc: u.doc}
	coll := u.st.units.Name
	getUnits := func() ([]string, error) {
		if err := u.Refresh(); err != nil {
			return nil, err
		}
		return u.doc.Subordinates, nil
	}
	return newUnitsWatcher(u.st, u.Tag(), getUnits, coll, u.doc.Name, u.doc.TxnRevno)
}

// WatchPrincipalUnits returns a UnitsWatcher tracking the machine's principal
// units.
func (m *Machine) WatchPrincipalUnits() *UnitsWatcher {
	m = &Machine{st: m.st, doc: m.doc}
	coll := m.st.machines.Name
	getUnits := func() ([]string, error) {
		if err := m.Refresh(); err != nil {
			return nil, err
		}
		return m.doc.Principals, nil
	}
	return newUnitsWatcher(m.st, m.Tag(), getUnits, coll, m.doc.Id, m.doc.TxnRevno)
}

func newUnitsWatcher(st *State, tag string, getUnits func() ([]string, error), coll, id string, revno int64) *UnitsWatcher {
	w := &UnitsWatcher{
		commonWatcher: commonWatcher{st: st},
		tag:           tag,
		getUnits:      getUnits,
		life:          map[string]Life{},
		in:            make(chan watcher.Change),
		out:           make(chan []string),
	}
	go func() {
		defer w.tomb.Done()
		defer close(w.out)
		w.tomb.Kill(w.loop(coll, id, revno))
	}()
	return w
}

// Tag returns the tag of the entity whose units are being watched.
func (w *UnitsWatcher) Tag() string {
	return w.tag
}

// Changes returns the UnitsWatcher's output channel.
func (w *UnitsWatcher) Changes() <-chan []string {
	return w.out
}

// lifeWatchDoc holds the fields used in starting and maintaining a watch
// on a entity's lifecycle.
type lifeWatchDoc struct {
	Id       string `bson:"_id"`
	Life     Life
	TxnRevno int64 `bson:"txn-revno"`
}

// lifeWatchFields specifies the fields of a lifeWatchDoc.
var lifeWatchFields = D{{"_id", 1}, {"life", 1}, {"txn-revno", 1}}

// initial returns every member of the tracked set.
func (w *UnitsWatcher) initial() ([]string, error) {
	initial, err := w.getUnits()
	if err != nil {
		return nil, err
	}
	docs := []lifeWatchDoc{}
	query := D{{"_id", D{{"$in", initial}}}}
	if err := w.st.units.Find(query).Select(lifeWatchFields).All(&docs); err != nil {
		return nil, err
	}
	changes := []string{}
	for _, doc := range docs {
		changes = append(changes, doc.Id)
		if doc.Life != Dead {
			w.life[doc.Id] = doc.Life
			w.st.watcher.Watch(w.st.units.Name, doc.Id, doc.TxnRevno, w.in)
		}
	}
	return changes, nil
}

// update adds to and returns changes, such that it contains the names of any
// non-Dead units to have entered or left the tracked set.
func (w *UnitsWatcher) update(changes []string) ([]string, error) {
	latest, err := w.getUnits()
	if err != nil {
		return nil, err
	}
	for _, name := range latest {
		if _, known := w.life[name]; !known {
			changes, err = w.merge(changes, name)
			if err != nil {
				return nil, err
			}
		}
	}
	for name := range w.life {
		if hasString(latest, name) {
			continue
		}
		if !hasString(changes, name) {
			changes = append(changes, name)
		}
		delete(w.life, name)
		w.st.watcher.Unwatch(w.st.units.Name, name, w.in)
	}
	return changes, nil
}

// merge adds to and returns changes, such that it contains the supplied unit
// name if that unit is unknown and non-Dead, or has changed lifecycle status.
func (w *UnitsWatcher) merge(changes []string, name string) ([]string, error) {
	doc := lifeWatchDoc{}
	err := w.st.units.FindId(name).Select(lifeWatchFields).One(&doc)
	gone := false
	if err == mgo.ErrNotFound {
		gone = true
	} else if err != nil {
		return nil, err
	} else if doc.Life == Dead {
		gone = true
	}
	life, known := w.life[name]
	switch {
	case known && gone:
		delete(w.life, name)
		w.st.watcher.Unwatch(w.st.units.Name, name, w.in)
	case !known && !gone:
		w.st.watcher.Watch(w.st.units.Name, name, doc.TxnRevno, w.in)
		w.life[name] = doc.Life
	case known && life != doc.Life:
		w.life[name] = doc.Life
	default:
		return changes, nil
	}
	if !hasString(changes, name) {
		changes = append(changes, name)
	}
	return changes, nil
}

func (w *UnitsWatcher) loop(coll, id string, revno int64) error {
	w.st.watcher.Watch(coll, id, revno, w.in)
	defer func() {
		w.st.watcher.Unwatch(coll, id, w.in)
		for name := range w.life {
			w.st.watcher.Unwatch(w.st.units.Name, name, w.in)
		}
	}()
	changes, err := w.initial()
	if err != nil {
		return err
	}
	out := w.out
	for {
		select {
		case <-w.st.watcher.Dead():
			return watcher.MustErr(w.st.watcher)
		case <-w.tomb.Dying():
			return tomb.ErrDying
		case c := <-w.in:
			name := c.Id.(string)
			if name == id {
				changes, err = w.update(changes)
			} else {
				changes, err = w.merge(changes, name)
			}
			if err != nil {
				return err
			}
			if len(changes) > 0 {
				out = w.out
			}
		case out <- changes:
			out = nil
			changes = nil
		}
	}
	return nil
}

// EnvironConfigWatcher observes changes to the
// environment configuration.
type EnvironConfigWatcher struct {
	commonWatcher
	out chan *config.Config
}

// WatchEnvironConfig returns a watcher for observing changes
// to the environment configuration.
func (s *State) WatchEnvironConfig() *EnvironConfigWatcher {
	return newEnvironConfigWatcher(s)
}

func newEnvironConfigWatcher(s *State) *EnvironConfigWatcher {
	w := &EnvironConfigWatcher{
		commonWatcher: commonWatcher{st: s},
		out:           make(chan *config.Config),
	}
	go func() {
		defer w.tomb.Done()
		defer close(w.out)
		w.tomb.Kill(w.loop())
	}()
	return w
}

// Changes returns a channel that will receive the new environment
// configuration when a change is detected. Note that multiple changes may
// be observed as a single event in the channel.
func (w *EnvironConfigWatcher) Changes() <-chan *config.Config {
	return w.out
}

func (w *EnvironConfigWatcher) loop() (err error) {
	sw := w.st.watchSettings(environGlobalKey)
	defer sw.Stop()
	out := w.out
	out = nil
	cfg := &config.Config{}
	for {
		select {
		case <-w.st.watcher.Dead():
			return watcher.MustErr(w.st.watcher)
		case <-w.tomb.Dying():
			return tomb.ErrDying
		case settings, ok := <-sw.Changes():
			if !ok {
				return watcher.MustErr(sw)
			}
			cfg, err = config.New(settings.Map())
			if err == nil {
				out = w.out
			} else {
				out = nil
			}
		case out <- cfg:
			out = nil
		}
	}
	return nil
}

type settingsWatcher struct {
	commonWatcher
	out chan *Settings
}

// watchSettings creates a watcher for observing changes to settings.
func (s *State) watchSettings(key string) *settingsWatcher {
	return newSettingsWatcher(s, key)
}

func newSettingsWatcher(s *State, key string) *settingsWatcher {
	w := &settingsWatcher{
		commonWatcher: commonWatcher{st: s},
		out:           make(chan *Settings),
	}
	go func() {
		defer w.tomb.Done()
		defer close(w.out)
		w.tomb.Kill(w.loop(key))
	}()
	return w
}

// Changes returns a channel that will receive the new settings.
// Multiple changes may be observed as a single event in the channel.
func (w *settingsWatcher) Changes() <-chan *Settings {
	return w.out
}

func (w *settingsWatcher) loop(key string) (err error) {
	ch := make(chan watcher.Change)
	revno := int64(-1)
	settings, err := readSettings(w.st, key)
	if err == nil {
		revno = settings.txnRevno
	} else if !errors.IsNotFoundError(err) {
		return err
	}
	w.st.watcher.Watch(w.st.settings.Name, key, revno, ch)
	defer w.st.watcher.Unwatch(w.st.settings.Name, key, ch)
	out := w.out
	if revno == -1 {
		out = nil
	}
	for {
		select {
		case <-w.st.watcher.Dead():
			return watcher.MustErr(w.st.watcher)
		case <-w.tomb.Dying():
			return tomb.ErrDying
		case <-ch:
			settings, err = readSettings(w.st, key)
			if err != nil {
				return err
			}
			out = w.out
		case out <- settings:
			out = nil
		}
	}
	return nil
}

// entityWatcher generates an event when a document in the db changes
type entityWatcher struct {
	commonWatcher
	out chan struct{}
}

// WatchHardwareCharacteristics returns a watcher for observing changes to a machine's hardware characteristics.
func (m *Machine) WatchHardwareCharacteristics() NotifyWatcher {
	return newEntityWatcher(m.st, m.st.instanceData, m.doc.Id)
}

// Watch returns a watcher for observing changes to a machine.
func (m *Machine) Watch() NotifyWatcher {
	return newEntityWatcher(m.st, m.st.machines, m.doc.Id)
}

// Watch returns a watcher for observing changes to a service.
func (s *Service) Watch() NotifyWatcher {
	return newEntityWatcher(s.st, s.st.services, s.doc.Name)
}

// Watch returns a watcher for observing changes to a unit.
func (u *Unit) Watch() NotifyWatcher {
	return newEntityWatcher(u.st, u.st.units, u.doc.Name)
}

// WatchForEnvironConfigChanges return a NotifyWatcher waiting for the Environ
// Config to change. This differs from WatchEnvironConfig in that the watcher
// is a NotifyWatcher that does not give content during Changes()
func (st *State) WatchForEnvironConfigChanges() NotifyWatcher {
	return newEntityWatcher(st, st.settings, environGlobalKey)
}

// WatchConfigSettings returns a watcher for observing changes to the
// unit's service configuration settings. The unit must have a charm URL
// set before this method is called, and the returned watcher will be
// valid only while the unit's charm URL is not changed.
// TODO(fwereade): this could be much smarter; if it were, uniter.Filter
// could be somewhat simpler.
func (u *Unit) WatchConfigSettings() (NotifyWatcher, error) {
	if u.doc.CharmURL == nil {
		return nil, fmt.Errorf("unit charm not set")
	}
	settingsKey := serviceSettingsKey(u.doc.Service, u.doc.CharmURL)
	return newEntityWatcher(u.st, u.st.settings, settingsKey), nil
}

func newEntityWatcher(st *State, coll *mgo.Collection, key string) NotifyWatcher {
	w := &entityWatcher{
		commonWatcher: commonWatcher{st: st},
		out:           make(chan struct{}),
	}
	go func() {
		defer w.tomb.Done()
		defer close(w.out)
		w.tomb.Kill(w.loop(coll, key))
	}()
	return w
}

// Changes returns the event channel for the entityWatcher.
func (w *entityWatcher) Changes() <-chan struct{} {
	return w.out
}

func (w *entityWatcher) loop(coll *mgo.Collection, key string) (err error) {
	doc := &struct {
		TxnRevno int64 `bson:"txn-revno"`
	}{}
	fields := D{{"txn-revno", 1}}
	if err := coll.FindId(key).Select(fields).One(doc); err == mgo.ErrNotFound {
		doc.TxnRevno = -1
	} else if err != nil {
		return err
	}
	in := make(chan watcher.Change)
	w.st.watcher.Watch(coll.Name, key, doc.TxnRevno, in)
	defer w.st.watcher.Unwatch(coll.Name, key, in)
	out := w.out
	for {
		select {
		case <-w.tomb.Dying():
			return tomb.ErrDying
		case <-w.st.watcher.Dead():
			return watcher.MustErr(w.st.watcher)
		case ch := <-in:
			if _, ok := collect(ch, in, w.tomb.Dying()); !ok {
				return tomb.ErrDying
			}
			out = w.out
		case out <- struct{}{}:
			out = nil
		}
	}
	return nil
}

// MachineUnitsWatcher notifies about assignments and lifecycle changes
// for all units of a machine.
//
// The first event emitted contains the unit names of all units currently
// assigned to the machine, irrespective of their life state. From then on,
// a new event is emitted whenever a unit is assigned to or unassigned from
// the machine, or the lifecycle of a unit that is currently assigned to
// the machine changes.
//
// After a unit is found to be Dead, no further event will include it.
type MachineUnitsWatcher struct {
	commonWatcher
	machine *Machine
	out     chan []string
	in      chan watcher.Change
	known   map[string]Life
}

// WatchUnits returns a new MachineUnitsWatcher for m.
func (m *Machine) WatchUnits() *MachineUnitsWatcher {
	return newMachineUnitsWatcher(m)
}

func newMachineUnitsWatcher(m *Machine) *MachineUnitsWatcher {
	w := &MachineUnitsWatcher{
		commonWatcher: commonWatcher{st: m.st},
		out:           make(chan []string),
		in:            make(chan watcher.Change),
		known:         make(map[string]Life),
		machine:       &Machine{st: m.st, doc: m.doc}, // Copy so it may be freely refreshed
	}
	go func() {
		defer w.tomb.Done()
		defer close(w.out)
		w.tomb.Kill(w.loop())
	}()
	return w
}

// Changes returns the event channel for w.
func (w *MachineUnitsWatcher) Changes() <-chan []string {
	return w.out
}

func (w *MachineUnitsWatcher) updateMachine(pending []string) (new []string, err error) {
	err = w.machine.Refresh()
	if err != nil {
		return nil, err
	}
	for _, unit := range w.machine.doc.Principals {
		if _, ok := w.known[unit]; !ok {
			pending, err = w.merge(pending, unit)
			if err != nil {
				return nil, err
			}
		}
	}
	return pending, nil
}

func (w *MachineUnitsWatcher) merge(pending []string, unit string) (new []string, err error) {
	doc := unitDoc{}
	err = w.st.units.FindId(unit).One(&doc)
	if err != nil && err != mgo.ErrNotFound {
		return nil, err
	}
	life, known := w.known[unit]
	if err == mgo.ErrNotFound || doc.Principal == "" && (doc.MachineId == "" || doc.MachineId != w.machine.doc.Id) {
		// Unit was removed or unassigned from w.machine.
		if known {
			delete(w.known, unit)
			w.st.watcher.Unwatch(w.st.units.Name, unit, w.in)
			if life != Dead && !hasString(pending, unit) {
				pending = append(pending, unit)
			}
			for _, subunit := range doc.Subordinates {
				if sublife, subknown := w.known[subunit]; subknown {
					delete(w.known, subunit)
					w.st.watcher.Unwatch(w.st.units.Name, subunit, w.in)
					if sublife != Dead && !hasString(pending, subunit) {
						pending = append(pending, subunit)
					}
				}
			}
		}
		return pending, nil
	}
	if !known {
		w.st.watcher.Watch(w.st.units.Name, unit, doc.TxnRevno, w.in)
		pending = append(pending, unit)
	} else if life != doc.Life && !hasString(pending, unit) {
		pending = append(pending, unit)
	}
	w.known[unit] = doc.Life
	for _, subunit := range doc.Subordinates {
		if _, ok := w.known[subunit]; !ok {
			pending, err = w.merge(pending, subunit)
			if err != nil {
				return nil, err
			}
		}
	}
	return pending, nil
}

func (w *MachineUnitsWatcher) loop() (err error) {
	defer func() {
		for unit := range w.known {
			w.st.watcher.Unwatch(w.st.units.Name, unit, w.in)
		}
	}()
	machineCh := make(chan watcher.Change)
	w.st.watcher.Watch(w.st.machines.Name, w.machine.doc.Id, w.machine.doc.TxnRevno, machineCh)
	defer w.st.watcher.Unwatch(w.st.machines.Name, w.machine.doc.Id, machineCh)
	changes, err := w.updateMachine([]string(nil))
	if err != nil {
		return err
	}
	out := w.out
	for {
		select {
		case <-w.st.watcher.Dead():
			return watcher.MustErr(w.st.watcher)
		case <-w.tomb.Dying():
			return tomb.ErrDying
		case <-machineCh:
			changes, err = w.updateMachine(changes)
			if err != nil {
				return err
			}
			if len(changes) > 0 {
				out = w.out
			}
		case c := <-w.in:
			changes, err = w.merge(changes, c.Id.(string))
			if err != nil {
				return err
			}
			if len(changes) > 0 {
				out = w.out
			}
		case out <- changes:
			out = nil
			changes = nil
		}
	}
	panic("unreachable")
}

// CleanupWatcher notifies of changes in the cleanups collection.
type CleanupWatcher struct {
	commonWatcher
	out chan struct{}
}

// WatchCleanups starts and returns a CleanupWatcher.
func (st *State) WatchCleanups() *CleanupWatcher {
	return newCleanupWatcher(st)
}

func newCleanupWatcher(st *State) *CleanupWatcher {
	w := &CleanupWatcher{
		commonWatcher: commonWatcher{st: st},
		out:           make(chan struct{}),
	}
	go func() {
		defer w.tomb.Done()
		defer close(w.out)
		w.tomb.Kill(w.loop())
	}()
	return w
}

// Changes returns the event channel for w.
func (w *CleanupWatcher) Changes() <-chan struct{} {
	return w.out
}

func (w *CleanupWatcher) loop() (err error) {
	in := make(chan watcher.Change)

	w.st.watcher.WatchCollection(w.st.cleanups.Name, in)
	defer w.st.watcher.UnwatchCollection(w.st.cleanups.Name, in)

	out := w.out
	for {
		select {
		case <-w.tomb.Dying():
			return tomb.ErrDying
		case <-w.st.watcher.Dead():
			return watcher.MustErr(w.st.watcher)
		case <-in:
			// Simply emit event for each change.
			out = w.out
		case out <- struct{}{}:
			out = nil
		}
	}
	panic("unreachable")
}
