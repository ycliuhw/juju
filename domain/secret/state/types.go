// Copyright 2024 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"fmt"
	"time"

	"github.com/juju/errors"

	coresecrets "github.com/juju/juju/core/secrets"
	domainsecret "github.com/juju/juju/domain/secret"
)

// These structs represent the persistent secretMetadata entity schema in the database.

type secretID struct {
	ID string `db:"id"`
}

type secretBackendID struct {
	ID string `db:"id"`
}

type revisionUUID struct {
	UUID string `db:"uuid"`
}

type entityRef struct {
	UUID string `db:"uuid"`
	ID   string `db:"id"`
}

type unit struct {
	UUID     string `db:"uuid"`
	UnitName string `db:"unit_id"`
}

type secretRef struct {
	ID         string `db:"secret_id"`
	SourceUUID string `db:"source_uuid"`
	Revision   int    `db:"revision"`
}

type secretMetadata struct {
	ID             string    `db:"secret_id"`
	Version        int       `db:"version"`
	Description    string    `db:"description"`
	AutoPrune      bool      `db:"auto_prune"`
	RotatePolicyID int       `db:"rotate_policy_id"`
	CreateTime     time.Time `db:"create_time"`
	UpdateTime     time.Time `db:"update_time"`
}

// secretInfo is used because sqlair doesn't seem to like struct embedding.
type secretInfo struct {
	ID           string    `db:"secret_id"`
	Version      int       `db:"version"`
	Description  string    `db:"description"`
	RotatePolicy string    `db:"policy"`
	AutoPrune    bool      `db:"auto_prune"`
	CreateTime   time.Time `db:"create_time"`
	UpdateTime   time.Time `db:"update_time"`

	NextRotateTime   time.Time `db:"next_rotation_time"`
	LatestExpireTime time.Time `db:"latest_expire_time"`
	LatestRevision   int       `db:"latest_revision"`
}

type secretModelOwner struct {
	SecretID string `db:"secret_id"`
	Label    string `db:"label"`
}

type secretApplicationOwner struct {
	SecretID        string `db:"secret_id"`
	ApplicationUUID string `db:"application_uuid"`
	Label           string `db:"label"`
}

type secretUnitOwner struct {
	SecretID string `db:"secret_id"`
	UnitUUID string `db:"unit_uuid"`
	Label    string `db:"label"`
}

type secretOwner struct {
	SecretID  string `db:"secret_id"`
	OwnerID   string `db:"owner_id"`
	OwnerKind string `db:"owner_kind"`
	Label     string `db:"label"`
}

type secretRotate struct {
	SecretID       string    `db:"secret_id"`
	NextRotateTime time.Time `db:"next_rotation_time"`
}

type secretRevision struct {
	ID            string    `db:"uuid"`
	SecretID      string    `db:"secret_id"`
	Revision      int       `db:"revision"`
	Obsolete      bool      `db:"obsolete"`
	PendingDelete bool      `db:"pending_delete"`
	CreateTime    time.Time `db:"create_time"`
	UpdateTime    time.Time `db:"update_time"`
}

type secretRevisionExpire struct {
	RevisionUUID string    `db:"revision_uuid"`
	ExpireTime   time.Time `db:"expire_time"`
}

type secretContent struct {
	RevisionUUID string `db:"revision_uuid"`
	Name         string `db:"name"`
	Content      string `db:"content"`
}

type secretValueRef struct {
	RevisionUUID string `db:"revision_uuid"`
	BackendUUID  string `db:"backend_uuid"`
	RevisionID   string `db:"revision_id"`
}

type secretExternalRevision struct {
	Revision    int    `db:"revision"`
	BackendUUID string `db:"backend_uuid"`
	RevisionID  string `db:"revision_id"`
}

type secretUnitConsumer struct {
	UnitUUID        string `db:"unit_uuid"`
	SecretID        string `db:"secret_id"`
	SourceModelUUID string `db:"source_model_uuid"`
	Label           string `db:"label"`
	CurrentRevision int    `db:"current_revision"`
}

type secretRemoteUnitConsumer struct {
	UnitID          string `db:"unit_id"`
	SecretID        string `db:"secret_id"`
	CurrentRevision int    `db:"current_revision"`
}

type remoteSecret struct {
	SecretID       string `db:"secret_id"`
	LatestRevision int    `db:"latest_revision"`
}

type secretPermission struct {
	SecretID      string                        `db:"secret_id"`
	RoleID        domainsecret.Role             `db:"role_id"`
	SubjectUUID   string                        `db:"subject_uuid"`
	SubjectTypeID domainsecret.GrantSubjectType `db:"subject_type_id"`
	ScopeUUID     string                        `db:"scope_uuid"`
	ScopeTypeID   domainsecret.GrantScopeType   `db:"scope_type_id"`
}

type secretAccessor struct {
	SecretID      string                        `db:"secret_id"`
	SubjectID     string                        `db:"subject_id"`
	SubjectTypeID domainsecret.GrantSubjectType `db:"subject_type_id"`
	RoleID        domainsecret.Role             `db:"role_id"`
}

type secretAccessorType struct {
	AppSubjectTypeID   domainsecret.GrantSubjectType `db:"app_type_id"`
	UnitSubjectTypeID  domainsecret.GrantSubjectType `db:"unit_type_id"`
	ModelSubjectTypeID domainsecret.GrantSubjectType `db:"model_type_id"`
}

var secretAccessorTypeParam = secretAccessorType{
	AppSubjectTypeID:   domainsecret.SubjectApplication,
	UnitSubjectTypeID:  domainsecret.SubjectUnit,
	ModelSubjectTypeID: domainsecret.SubjectModel,
}

type secretAccessScope struct {
	ScopeID     string                      `db:"scope_id"`
	ScopeTypeID domainsecret.GrantScopeType `db:"scope_type_id"`
}

type ownerKind struct {
	Model       string `db:"model_owner_kind"`
	Unit        string `db:"unit_owner_kind"`
	Application string `db:"application_owner_kind"`
}

var ownerKindParam = ownerKind{
	Model:       string(coresecrets.ModelOwner),
	Unit:        string(coresecrets.UnitOwner),
	Application: string(coresecrets.ApplicationOwner),
}

type secrets []secretInfo

func (rows secrets) toSecretMetadata(secretOwners []secretOwner) ([]*coresecrets.SecretMetadata, error) {
	if len(rows) != len(secretOwners) {
		// Should never happen.
		return nil, errors.New("row length mismatch composing secret results")
	}

	result := make([]*coresecrets.SecretMetadata, len(rows))
	for i, row := range rows {
		uri, err := coresecrets.ParseURI(row.ID)
		if err != nil {
			return nil, errors.NotValidf("secret URI %q", row.ID)
		}
		result[i] = &coresecrets.SecretMetadata{
			URI:         uri,
			Version:     row.Version,
			Description: row.Description,
			Label:       secretOwners[i].Label,
			Owner: coresecrets.Owner{
				Kind: coresecrets.OwnerKind(secretOwners[i].OwnerKind),
				ID:   secretOwners[i].OwnerID,
			},
			CreateTime:     row.CreateTime,
			UpdateTime:     row.UpdateTime,
			LatestRevision: row.LatestRevision,
			AutoPrune:      row.AutoPrune,
			RotatePolicy:   coresecrets.RotatePolicy(row.RotatePolicy),
		}
		if tm := row.NextRotateTime; !tm.IsZero() {
			result[i].NextRotateTime = &tm
		}
		if tm := row.LatestExpireTime; !tm.IsZero() {
			result[i].LatestExpireTime = &tm
		}
	}
	return result, nil
}

func (rows secrets) toSecretRevisionRef(refs secretValueRefs) ([]*coresecrets.SecretRevisionRef, error) {
	if len(rows) != len(refs) {
		// Should never happen.
		return nil, errors.New("row length mismatch composing secret results")
	}

	result := make([]*coresecrets.SecretRevisionRef, len(rows))
	for i, row := range rows {
		uri, err := coresecrets.ParseURI(row.ID)
		if err != nil {
			return nil, errors.NotValidf("secret URI %q", row.ID)
		}
		result[i] = &coresecrets.SecretRevisionRef{
			URI:        uri,
			RevisionID: refs[i].RevisionID,
		}
	}
	return result, nil
}

type (
	secretIDs               []secretID
	secretExternalRevisions []secretExternalRevision
)

func (rows secretIDs) toSecretMetadataForDrain(revRows secretExternalRevisions) ([]*coresecrets.SecretMetadataForDrain, error) {
	if len(rows) != len(revRows) {
		// Should never happen.
		return nil, errors.New("row length mismatch composing secret results")
	}

	var (
		result  []*coresecrets.SecretMetadataForDrain
		current *coresecrets.SecretMetadataForDrain
	)
	for i, row := range rows {
		if current == nil || current.URI.ID != row.ID {
			// Encountered a new record.
			uri, err := coresecrets.ParseURI(row.ID)
			if err != nil {
				return nil, errors.NotValidf("secret URI %q", row.ID)
			}
			md := coresecrets.SecretMetadataForDrain{
				URI: uri,
			}
			current = &md
			result = append(result, current)
		}
		rev := coresecrets.SecretExternalRevision{
			Revision: revRows[i].Revision,
		}
		if revRows[i].BackendUUID != "" {
			rev.ValueRef = &coresecrets.ValueRef{
				BackendID:  revRows[i].BackendUUID,
				RevisionID: revRows[i].RevisionID,
			}
		}
		current.Revisions = append(current.Revisions, rev)
	}
	return result, nil
}

type secretRevisions []secretRevision
type secretRevisionsExpire []secretRevisionExpire

func (rows secretRevisions) toSecretRevisions(valueRefs secretValueRefs, revExpire secretRevisionsExpire) ([]*coresecrets.SecretRevisionMetadata, error) {
	if n := len(rows); n != len(revExpire) || n != len(valueRefs) {
		// Should never happen.
		return nil, errors.New("row length mismatch composing secret revision results")
	}

	result := make([]*coresecrets.SecretRevisionMetadata, len(rows))
	for i, row := range rows {
		result[i] = &coresecrets.SecretRevisionMetadata{
			Revision:   row.Revision,
			CreateTime: row.CreateTime,
			UpdateTime: row.UpdateTime,
		}
		if tm := revExpire[i].ExpireTime; !tm.IsZero() {
			result[i].ExpireTime = &tm
		}
		if v := valueRefs[i]; v.BackendUUID != "" {
			result[i].ValueRef = &coresecrets.ValueRef{
				BackendID:  v.BackendUUID,
				RevisionID: v.RevisionID,
			}
		}
	}
	return result, nil
}

type secretValues []secretContent

func (rows secretValues) toSecretData() coresecrets.SecretData {
	result := make(coresecrets.SecretData)
	for _, row := range rows {
		result[row.Name] = row.Content
	}
	return result
}

type secretValueRefs []secretValueRef

func (rows secretValueRefs) toValueRefs() []coresecrets.ValueRef {
	result := make([]coresecrets.ValueRef, len(rows))
	for i, row := range rows {
		result[i] = coresecrets.ValueRef{
			BackendID:  row.BackendUUID,
			RevisionID: row.RevisionID,
		}
	}
	return result
}

type secretRemoteUnitConsumers []secretRemoteUnitConsumer

func (rows secretRemoteUnitConsumers) toSecretConsumers() []*coresecrets.SecretConsumerMetadata {
	result := make([]*coresecrets.SecretConsumerMetadata, len(rows))
	for i, row := range rows {
		result[i] = &coresecrets.SecretConsumerMetadata{
			CurrentRevision: row.CurrentRevision,
		}
	}
	return result
}

type secretUnitConsumers []secretUnitConsumer

func (rows secretUnitConsumers) toSecretConsumers() []*coresecrets.SecretConsumerMetadata {
	result := make([]*coresecrets.SecretConsumerMetadata, len(rows))
	for i, row := range rows {
		result[i] = &coresecrets.SecretConsumerMetadata{
			Label:           row.Label,
			CurrentRevision: row.CurrentRevision,
		}
	}
	return result
}

type secretAccessors []secretAccessor

type secretAccessScopes []secretAccessScope

func (rows secretAccessors) toSecretGrants(scopes secretAccessScopes) ([]domainsecret.GrantParams, error) {
	if len(rows) != len(scopes) {
		// Should never happen.
		return nil, errors.New("row length mismatch composing grant results")
	}
	result := make([]domainsecret.GrantParams, len(rows))
	for i, row := range rows {
		result[i] = domainsecret.GrantParams{
			SubjectTypeID: row.SubjectTypeID,
			SubjectID:     row.SubjectID,
			RoleID:        row.RoleID,
			ScopeTypeID:   scopes[i].ScopeTypeID,
			ScopeID:       scopes[i].ScopeID,
		}
	}
	return result, nil
}

type obsoleteRevisionRow struct {
	SecretID string `db:"secret_id"`
	Revision string `db:"revision"`
}

type obsoleteRevisionRows []obsoleteRevisionRow

func (rows obsoleteRevisionRows) toRevIDs() []string {
	result := make([]string, len(rows))
	for i, row := range rows {
		result[i] = fmt.Sprintf("%s/%s", row.SecretID, row.Revision)
	}
	return result
}
