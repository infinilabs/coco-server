/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

// Package documentview is the typed view bridge between the record model
// and coco's domain object: Fields["document"] is the single authority,
// Materialize decodes it into *core.Document for processors that need
// typed access, Project writes mutations back. One conversion each way
// per invocation — generic processors never pay it.
package documentview

import (
	"fmt"

	"infini.sh/coco/core"
	"infini.sh/coco/core/codecs"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/util"
)

// Materialize decodes the record's document subtree into a typed
// *core.Document. Returns an error when the record carries no document
// subtree (callers in a document-namespace chain treat that as a skip).
func Materialize(rec *event.Event) (*core.Document, error) {
	v, err := rec.Fields.GetValue(codecs.FieldDocument)
	if err != nil || v == nil {
		return nil, fmt.Errorf("record carries no %q subtree", codecs.FieldDocument)
	}
	if m, ok := v.(util.MapStr); ok && len(m) == 0 {
		return nil, fmt.Errorf("record carries an empty %q subtree", codecs.FieldDocument)
	}
	b, err := util.ToJSONBytes(v)
	if err != nil {
		return nil, err
	}
	doc := &core.Document{}
	if err := util.FromJSONBytes(b, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// Project writes the typed document back into the record's document
// subtree, replacing it wholesale (Fields stays the authority).
func Project(doc *core.Document, rec *event.Event) error {
	if doc == nil {
		return nil
	}
	b, err := util.ToJSONBytes(doc)
	if err != nil {
		return err
	}
	m := util.MapStr{}
	if err := util.FromJSONBytes(b, &m); err != nil {
		return err
	}
	rec.Fields[codecs.FieldDocument] = m
	return nil
}
