// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"fmt"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/mgo/v3"
	"github.com/juju/mgo/v3/bson"
	"github.com/juju/mgo/v3/txn"
	"github.com/juju/names/v4"
)

// annotatorDoc represents the internal state of annotations for an Entity in
// MongoDB. Note that the annotations map is not maintained in local storage
// due to the fact that it is not accessed directly, but through
// Annotations/Annotation below.
// Note also the correspondence with AnnotationInfo in apiserver/params.
type annotatorDoc struct {
	ModelUUID   string            `bson:"model-uuid"`
	GlobalKey   string            `bson:"globalkey"`
	Tag         string            `bson:"tag"`
	Annotations map[string]string `bson:"annotations"`
}

// SetAnnotations adds key/value pairs to annotations in MongoDB.
func (m *Model) SetAnnotations(entity GlobalEntity, annotations map[string]string) (err error) {
	defer errors.DeferredAnnotatef(&err, "cannot update annotations on %s", entity.Tag())
	if len(annotations) == 0 {
		return nil
	}
	// Collect in separate maps pairs to be inserted/updated or removed.
	toRemove := make(bson.M)
	toInsert := make(map[string]string)
	toUpdate := make(bson.M)
	for key, value := range annotations {
		if strings.Contains(key, ".") {
			return fmt.Errorf("invalid key %q", key)
		}
		if value == "" {
			toRemove[key] = true
		} else {
			toInsert[key] = value
			toUpdate[key] = value
		}
	}
	// Set up and call the necessary transactions - if the document does not
	// already exist, one of the clients will create it and the others will
	// fail, then all the rest of the clients should succeed on their second
	// attempt. If the referred-to entity has disappeared, and removed its
	// annotations in the meantime, we consider that worthy of an error
	// (will be fixed when new entities can never share names with old ones).
	buildTxn := func(attempt int) ([]txn.Op, error) {
		annotations, closer := m.st.db().GetCollection(annotationsC)
		defer closer()
		if count, err := annotations.FindId(entity.globalKey()).Count(); err != nil {
			return nil, err
		} else if count == 0 {
			// Check that the annotator entity was not previously destroyed.
			if attempt != 0 {
				return nil, fmt.Errorf("%s no longer exists", entity.Tag())
			}
			return insertAnnotationsOps(m.st, entity, toInsert)
		}
		return updateAnnotations(m.st, entity, toUpdate, toRemove), nil
	}
	return m.st.db().Run(buildTxn)
}

// Annotations returns all the annotations corresponding to an entity.
func (m *Model) Annotations(entity GlobalEntity) (map[string]string, error) {
	doc := new(annotatorDoc)
	annotations, closer := m.st.db().GetCollection(annotationsC)
	defer closer()
	err := annotations.FindId(entity.globalKey()).One(doc)
	if err == mgo.ErrNotFound {
		// Returning an empty map if there are no annotations.
		return make(map[string]string), nil
	}
	if err != nil {
		return nil, errors.Trace(err)
	}
	return doc.Annotations, nil
}

// Annotation returns the annotation value corresponding to the given key.
// If the requested annotation is not found, an empty string is returned.
func (m *Model) Annotation(entity GlobalEntity, key string) (string, error) {
	ann, err := m.Annotations(entity)
	if err != nil {
		return "", errors.Trace(err)
	}
	return ann[key], nil
}

// insertAnnotationsOps returns the operations required to insert annotations in MongoDB.
func insertAnnotationsOps(st *State, entity GlobalEntity, toInsert map[string]string) ([]txn.Op, error) {
	tag := entity.Tag()
	ops := []txn.Op{{
		C:      annotationsC,
		Id:     st.docID(entity.globalKey()),
		Assert: txn.DocMissing,
		Insert: &annotatorDoc{
			GlobalKey:   entity.globalKey(),
			Tag:         tag.String(),
			Annotations: toInsert,
		},
	}}

	switch tag := tag.(type) {
	case names.ModelTag:
		if tag.Id() == st.ControllerModelUUID() {
			// This is the controller model, and cannot be removed.
			// Ergo, we can skip the existence check below.
			return ops, nil
		}
	}

	// If the entity is not the controller model, add a DocExists check on the
	// entity document, in order to avoid possible races between entity
	// removal and annotation creation.
	coll, id, err := st.tagToCollectionAndId(tag)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return append(ops, txn.Op{
		C:      coll,
		Id:     id,
		Assert: txn.DocExists,
	}), nil
}

// updateAnnotations returns the operations required to update or remove annotations in MongoDB.
func updateAnnotations(mb modelBackend, entity GlobalEntity, toUpdate, toRemove bson.M) []txn.Op {
	return []txn.Op{{
		C:      annotationsC,
		Id:     mb.docID(entity.globalKey()),
		Assert: txn.DocExists,
		Update: setUnsetUpdateAnnotations(toUpdate, toRemove),
	}}
}

// annotationRemoveOp returns an operation to remove a given annotation
// document from MongoDB.
func annotationRemoveOp(mb modelBackend, id string) txn.Op {
	return txn.Op{
		C:      annotationsC,
		Id:     mb.docID(id),
		Remove: true,
	}
}

// setUnsetUpdateAnnotations returns a bson.D for use
// in an annotationsC txn.Op's Update field, containing $set and
// $unset operators if the corresponding operands
// are non-empty.
func setUnsetUpdateAnnotations(set, unset bson.M) bson.D {
	var update bson.D
	if len(set) > 0 {
		set = bson.M(subDocKeys(map[string]interface{}(set), "annotations"))
		update = append(update, bson.DocElem{Name: "$set", Value: set})
	}
	if len(unset) > 0 {
		unset = bson.M(subDocKeys(map[string]interface{}(unset), "annotations"))
		update = append(update, bson.DocElem{Name: "$unset", Value: unset})
	}
	return update
}
