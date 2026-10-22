/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package utils

import (
	"context"
	"fmt"
	"time"

	log "github.com/cihub/seelog"
	"golang.org/x/text/language"
	"infini.sh/coco/core/codecs"
	"infini.sh/coco/core"
	"infini.sh/coco/core/documentview"
	"infini.sh/coco/modules/datasource"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
)

// GetDatasource returns the datasource for the given document.
func GetDatasource(doc *core.Document) (*core.DataSource, error) {
	if doc.Source.ID == "" {
		return nil, fmt.Errorf("document has no datasource ID")
	}

	datasources, err := datasource.GetDatasourceByID([]string{doc.Source.ID})
	if err != nil {
		return nil, fmt.Errorf("failed to get datasource: %w", err)
	}
	if len(datasources) != 1 {
		return nil, fmt.Errorf("expected exactly 1 datasource, got %d", len(datasources))
	}

	return &datasources[0], nil
}

// GetConnectorID returns the connector ID for the given document.
func GetConnectorID(doc *core.Document) (string, error) {
	ds, err := GetDatasource(doc)
	if err != nil {
		return "", err
	}

	if ds.Connector.ConnectorID == "" {
		return "", fmt.Errorf("datasource has no connector ID")
	}

	return ds.Connector.ConnectorID, nil
}

// ValidateAndNormalizeLLMLang validates and normalizes a language tag for LLM generation.
// Defaults to "en-US" if empty or invalid.
// Returns the normalized BCP 47 language tag (e.g., "en-US", "zh-CN").
func ValidateAndNormalizeLLMLang(processorName, lang string) string {
	// Default to English if not set
	if lang == "" {
		return "en-US"
	}

	// Validate and normalize language tag
	tag, err := language.Parse(lang)
	if err != nil {
		log.Warnf("Processor [%s]: invalid llm_generation_lang %q, falling back to en-US: %v", processorName, lang, err)
		return "en-US"
	}

	// Normalize to BCP 47 format (e.g., "en_US" -> "en-US")
	return tag.String()
}

// Dual-mode record adaptation (P3): the shared bridge between the legacy
// messages convention and the record convention. Document processors keep
// ONE per-document function; Process/ProcessBatch routes to it.

// DocumentFn is a per-document processing step. Return handled=false to
// leave the record untouched (the skip semantics of the legacy `continue`).
type DocumentFn func(ctx context.Context, doc *core.Document) (handled bool, err error)

// EachDocumentRecord adapts DocumentFn to the record convention: materialize
// → fn → project. Records without a document subtree are skipped (they are
// not this processor's family). Always projects handled documents back —
// Fields stays the single authority.
func EachDocumentRecord(ctx context.Context, records []*event.Event, fn DocumentFn) error {
	for _, rec := range records {
		if rec == nil {
			continue
		}
		doc, err := documentview.Materialize(rec)
		if err != nil {
			// not a document-family record — skip, matching the legacy
			// type-assertion warn-and-continue behavior
			continue
		}
		handled, err := fn(ctx, doc)
		if err != nil {
			return err
		}
		if handled {
			if err := documentview.Project(doc, rec); err != nil {
				return err
			}
		}
	}
	return nil
}

// DocumentRecordContract is the standard record contract for Coco document
// processors (D2): record scope, document namespace. granularity varies
// per processor and is set by the caller.
func DocumentRecordContract(granularity pipeline.Granularity) pipeline.RecordContract {
	return pipeline.RecordContract{
		Scope:       pipeline.ScopeRecord,
		Namespace:   "document",
		Granularity: granularity,
	}
}

// AttachmentFn is a per-attachment processing step (handled=false skips).
type AttachmentFn func(ctx context.Context, att *core.Attachment) (handled bool, err error)

// EachAttachmentRecord adapts AttachmentFn to the record convention
// (attachment namespace subtree "attachment").
func EachAttachmentRecord(ctx context.Context, records []*event.Event, fn AttachmentFn) error {
	for _, rec := range records {
		if rec == nil {
			continue
		}
		v, err := rec.Fields.GetValue(codecs.FieldAttachment)
		if err != nil || v == nil {
			continue // not an attachment-family record
		}
		if m, ok := v.(util.MapStr); ok && len(m) == 0 {
			continue
		}
		b, err := util.ToJSONBytes(v)
		if err != nil {
			return err
		}
		att := &core.Attachment{}
		if err := util.FromJSONBytes(b, att); err != nil {
			continue
		}
		handled, err := fn(ctx, att)
		if err != nil {
			return err
		}
		if handled {
			nb, err := util.ToJSONBytes(att)
			if err != nil {
				return err
			}
			nm := util.MapStr{}
			if err := util.FromJSONBytes(nb, &nm); err != nil {
				return err
			}
			rec.Fields[codecs.FieldAttachment] = nm
		}
	}
	return nil
}

// AttachmentRecordContract is the standard contract for attachment-family
// processors (record scope, attachment namespace).
func AttachmentRecordContract(granularity pipeline.Granularity) pipeline.RecordContract {
	return pipeline.RecordContract{
		Scope:       pipeline.ScopeRecord,
		Namespace:   "attachment",
		Granularity: granularity,
	}
}

// PipelineLLMContext bounds an LLM call made inside a document-processing
// pipeline. langchaingo's default HTTP client has no overall timeout and the
// pipeline context carries no deadline, so one stalled provider connection
// hangs the (serial) queue consumer forever — every datasource's sync run
// then reads as stale ("疑似僵死") and ingestion stops silently. Three
// minutes is far above the observed latency of every pipeline LLM call
// (max ~40s summarization) while capping the damage of a dead connection.
func PipelineLLMContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 3*time.Minute)
}
