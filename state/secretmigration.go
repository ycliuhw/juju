// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"strconv"

	"github.com/juju/collections/set"
	"github.com/juju/errors"
	"github.com/juju/mgo/v3/bson"
	"github.com/juju/mgo/v3/txn"
	"github.com/kr/pretty"

	"github.com/juju/juju/mongo"
)

// SecretMigrationTaskStatus represents the possible end states for a secret migration task.
// TODO: rename ActionStatus to TaskStatus and move it to core package the reuse it here?
type SecretMigrationTaskStatus string

const (

	// SecretMigrationTaskPending is the default status when an SecretMigrationTask is first queued.
	SecretMigrationTaskPending SecretMigrationTaskStatus = "pending"

	// SecretMigrationTaskRunning indicates that the SecretMigrationTask is currently running.
	SecretMigrationTaskRunning SecretMigrationTaskStatus = "running"

	// SecretMigrationTaskFailed signifies that the SecretMigrationTask did not complete successfully.
	SecretMigrationTaskFailed SecretMigrationTaskStatus = "failed"

	// SecretMigrationTaskCompleted indicates that the action ran to completion as intended.
	SecretMigrationTaskCompleted SecretMigrationTaskStatus = "completed"
)

// secretMigrationTaskDoc tracks a secret migration task.
type secretMigrationTaskDoc struct {
	DocID string `bson:"_id"`

	OwnerTag string `bson:"owner-tag"`

	// Status is the status of the secret migration task.
	// It can be "pending", "running", "completed", "failed".
	// This document will be removed by the secretmigrate worker on the controller once the owner
	// unit migrated all its secrets to the new secret backend and marks the status to "completed".
	Status string `bson:"status"`

	// TODO: Schedule next run if it was failed!!!!
	// ShouldRetry bool ?? then the worker changes the status to "pending" and the next run will be scheduled????
}

// TODO: we should handle properly the case when the model's secret backend changed very quickly!!!
// We don't want to create a huge mount of new tasks to make all owner units too busy!!!

// ScheduleSecretMigrationTasks creates a secret migration task for each secret owner when the model's secret backend changed.
func (m *Model) ScheduleSecretMigrationTasks() error {
	logger.Criticalf("ScheduleSecretMigrationTasks called for model %q", m.UUID())
	secretMetadataCol, closer := m.st.db().GetCollection(secretMetadataC)
	defer closer()

	var ownerTags []string
	err := secretMetadataCol.Find(nil).Distinct("owner-tag", &ownerTags)
	if err != nil {
		return errors.Trace(err)
	}
	ownerTagsSet := set.NewStrings(ownerTags...)
	logger.Criticalf("ScheduleSecretMigrationTasks secretMetadataC found %s", pretty.Sprint(ownerTags))

	secretMigrationTasksCol, closer := m.st.db().GetCollection(secretMigrationTasksC)
	defer closer()

	var alreadyScheduled []string
	err = secretMigrationTasksCol.Find(nil).Distinct("owner-tag", &alreadyScheduled)
	if err != nil {
		return errors.Trace(err)
	}
	logger.Criticalf("ScheduleSecretMigrationTasks secretMigrationTasksC found %s", pretty.Sprint(alreadyScheduled))
	for _, tag := range alreadyScheduled {
		ownerTagsSet.Remove(tag)
	}
	// TODO: ownerTagsSet.Remove(secrets does not have any inactive backend revisions)

	buildTxn := func(attempt int) ([]txn.Op, error) {
		var ops []txn.Op
		for _, ownerTag := range ownerTagsSet.Values() {
			id, err := sequenceWithMin(m.st, "secretMigrationTask", 1)
			if err != nil {
				return nil, errors.Trace(err)
			}
			taskId := strconv.Itoa(id)
			ops = append(ops, txn.Op{
				C:  secretMigrationTasksC,
				Id: taskId,
				Insert: &secretMigrationTaskDoc{
					DocID:    m.st.docID(taskId),
					OwnerTag: ownerTag,
					Status:   string(SecretMigrationTaskPending),
				},
			})
		}
		logger.Criticalf("ScheduleSecretMigrationTasks secretMigrationTasksC ops %s", pretty.Sprint(ops))
		return ops, nil
	}
	err = m.st.db().Run(buildTxn)
	return errors.Trace(err)
}

func (m *Model) cancelNonRunningSecretMigrationTasks(col mongo.Collection) error {
	_, err := col.Writeable().RemoveAll(bson.M{"status": bson.M{"$ne": SecretMigrationTaskRunning}})
	return errors.Trace(err)
}

// StartSecretMigrationTask marks a secret migration task as running.
// This is called by the owner unit when it starts migrating its secrets to the new secret backend.
func (m *Model) StartSecretMigrationTask(id string) error {
	logger.Criticalf("StartSecretMigrationTask => %q", id)
	ops := []txn.Op{{
		C:  secretMigrationTasksC,
		Id: m.st.docID(id),
		Assert: bson.M{"status": bson.M{"$in": []string{
			string(SecretMigrationTaskPending),
			string(SecretMigrationTaskFailed), // TODO: should we allow to retry failed tasks?
		}}},
		Update: bson.M{"$set": bson.M{"status": SecretMigrationTaskRunning}},
	}}
	return m.st.db().RunTransaction(ops)
}

// FailSecretMigrationTask marks a secret migration task as failed.
// This is called by the owner units when the task was failed.
func (m *Model) FailSecretMigrationTask(id string) error {
	logger.Criticalf("FailSecretMigrationTask => %q", id)
	ops := []txn.Op{{
		C:      secretMigrationTasksC,
		Id:     m.st.docID(id),
		Assert: bson.M{"status": SecretMigrationTaskRunning},
		Update: bson.M{"$set": bson.M{"status": SecretMigrationTaskFailed}},
	}}
	return m.st.db().RunTransaction(ops)
}

// CompleteSecretMigrationTask marks a secret migration task as completed.
// This is called by the owner units when they migrated all their secrets to the new secret backend.
func (m *Model) CompleteSecretMigrationTask(id string) error {
	logger.Criticalf("CompleteSecretMigrationTask => %q", id)
	ops := []txn.Op{{
		C:      secretMigrationTasksC,
		Id:     m.st.docID(id),
		Assert: bson.M{"status": SecretMigrationTaskRunning},
		Update: bson.M{"$set": bson.M{"status": SecretMigrationTaskCompleted}},
	}}
	return m.st.db().RunTransaction(ops)
}

// RemoveSecretMigrationTasks removes completed secret migration tasks.
// This is called by the secretmigrate worker on the controller to clean up completed tasks.
func (m *Model) RemoveSecretMigrationTasks() error {
	logger.Criticalf("RemoveSecretMigrationTasks called for model %q", m.UUID())
	col, closer := m.st.db().GetCollection(secretMigrationTasksC)
	defer closer()

	_, err := col.Writeable().RemoveAll(bson.M{"status": SecretMigrationTaskCompleted})
	return errors.Trace(err)
}
