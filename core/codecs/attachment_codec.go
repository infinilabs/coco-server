/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package codecs

import (
	"fmt"

	"infini.sh/framework/core/event"
	pipeline "infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
)

const (
	// CodecNameAttachment / NamespaceAttachment identify the attachment codec.
	CodecNameAttachment = "coco_attachment"
	NamespaceAttachment = "attachment"

	// FieldAttachment is the event Fields key carrying the projected
	// attachment tree.
	FieldAttachment = NamespaceAttachment
)

// AttachmentCodec projects a core.Attachment JSON payload onto
// *event.Event, mirroring DocumentCodec's conventions.
type AttachmentCodec struct{}

func (AttachmentCodec) Name() string      { return CodecNameAttachment }
func (AttachmentCodec) Namespace() string { return NamespaceAttachment }

func (AttachmentCodec) Decode(data []byte) (*event.Event, error) {
	m := util.MapStr{}
	if err := util.FromJSONBytes(data, &m); err != nil {
		return nil, fmt.Errorf("%s: decode payload: %w", CodecNameAttachment, err)
	}
	rec := &event.Event{Fields: util.MapStr{FieldAttachment: m}}
	rec.SetID(asString(m["id"]))
	labels := util.MapStr{codecLabel: CodecNameAttachment}
	putLabel(labels, "mime_type", m["mime_type"])
	rec.Metadata.Labels = labels
	return rec, nil
}

func (AttachmentCodec) Encode(rec *event.Event) ([]byte, error) {
	v, err := rec.Fields.GetValue(FieldAttachment)
	if err != nil || v == nil {
		return nil, fmt.Errorf("%s: missing %q subtree in record fields", CodecNameAttachment, FieldAttachment)
	}
	return hotJSON.Marshal(v)
}

func init() {
	pipeline.MustRegisterRecordCodec(AttachmentCodec{})
}
