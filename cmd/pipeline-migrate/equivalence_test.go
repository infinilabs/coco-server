/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

/* P8 #114 acceptance tail — runtime equivalence of the one-shot migration.
 *
 * The two live pipelines pipeline-audit flags as needs-migration (fixtures
 * mirror the stored coco_pipelineconfigv2 rows) are replayed through the
 * real engine in BOTH shapes with identical samples: the legacy chain
 * (record processors at the batch root, how they run today) and the
 * migrated shape (the for_each wrap pipeline-migrate produces). The final
 * record state must be equal — that is the semantic-equivalence guarantee
 * #115-117's delete-and-flip rests on.
 *
 * The engine runs the real processors (no mocks beyond the dry-run write
 * capture); LLM steps fail identically in both shapes with "no language
 * model configured", which exercises the error-path equivalence too:
 * dry-run stops after the failing step's partial mutations, for_each's
 * on_failure=ignore skips the rest of the sub-chain — the same point in
 * the sequence. Volatile fields (timestamps) are normalized before
 * comparison. */

package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/cmd/internal/pipescan"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/modules/pipeline/dryrun"
)

// enrichDocumentsChain mirrors the stored enrich_documents pipeline
// (coco_pipelineconfigv2, fetched 2026-10-25) — 9 record processors at the
// batch root, the audit's first needs-migration hit.
var enrichDocumentsChain = []map[string]interface{}{
	{"file_type_detection": map[string]interface{}{}},
	{"file_metadata": map[string]interface{}{}},
	{"generate_document_cover": map[string]interface{}{}},
	{"document_text_attachment_extraction": map[string]interface{}{
		"chunk_size":          7000,
		"extract_attachments": true,
	}},
	{"face_extraction": map[string]interface{}{
		"pigo_facefinder_path": "./config/ai/facefinder",
	}},
	{"document_summarization": map[string]interface{}{
		"model_context_length":   128000,
		"ai_insights_max_length": 500,
	}},
	{"extract_tags": map[string]interface{}{
		"model_context_length": 128000,
	}},
	{"document_embedding": map[string]interface{}{}},
	{"extract_entities": map[string]interface{}{
		"model_context_length": 128000,
		"wiki_kb_id":           "db4c8173edbmiu6ilvh0",
	}},
}

// enrichAttachmentsChain mirrors the stored enrich_attachments pipeline —
// one attachment-namespace processor at the batch root.
var enrichAttachmentsChain = []map[string]interface{}{
	{"attachment_text_extraction": map[string]interface{}{}},
}

const documentSample = `{"id":"eq-probe-001","title":"Equivalence Probe","content":"The quick brown fox jumps over the lazy dog. Runtime equivalence sample.","lang":"en","type":"txt"}`

const attachmentSample = `{"id":"eq-att-001","filename":"probe.txt","content":"plain text attachment body"}`

func TestRuntimeEquivalence_EnrichDocuments(t *testing.T) {
	runEquivalence(t, "enrich_documents", enrichDocumentsChain,
		dryrun.Sample{Raw: []byte(documentSample), Codec: "coco_document"})
}

func TestRuntimeEquivalence_EnrichAttachments(t *testing.T) {
	runEquivalence(t, "enrich_attachments", enrichAttachmentsChain,
		dryrun.Sample{Raw: []byte(attachmentSample), Codec: "coco_attachment"})
}

func runEquivalence(t *testing.T, name string, chain []map[string]interface{}, sample dryrun.Sample) {
	t.Helper()

	// the fixture must stay the legacy shape the migration targets
	_, warns := pipescan.Validate(&pipeline.PipelineConfigV2{Name: name, Processors: chain})
	require.NotEmpty(t, warns, "fixture no longer trips the contract warning — update fixtures from the stored pipeline")

	migrated, err := migrateChain(cloneSteps(chain))
	require.NoError(t, err)

	// the migrated shape must be the fully compliant one the tool writes
	mErrs, mWarns := pipescan.Validate(&pipeline.PipelineConfigV2{Name: name, Processors: migrated})
	require.Empty(t, mErrs, "migrated shape must compile")
	require.Empty(t, mWarns, "migrated shape must carry no warnings")

	legacy, err := dryrun.Run(chain, []dryrun.Sample{sample}, dryrun.Options{})
	require.NoError(t, err)
	wrapped, err := dryrun.Run(migrated, []dryrun.Sample{sample}, dryrun.Options{})
	require.NoError(t, err)

	// both shapes must actually execute the fixture — a vacuous run-through
	// (empty replay) proves nothing. The legacy replay executes steps in
	// fixture order and stops after the first error, so assert an ordered
	// prefix (which step errors depends on the environment's LLM config).
	require.NotEmpty(t, legacy.Steps[0], "legacy shape replayed nothing")
	for i, st := range legacy.Steps[0] {
		require.Equal(t, stepName(chain[i]), st.Processor,
			"legacy replay must execute the fixture's steps in order")
	}
	require.NotEmpty(t, wrapped.Steps[0], "migrated shape replayed nothing")
	require.Equal(t, "ok", wrapped.Steps[0][len(wrapped.Steps[0])-1].Outcome,
		"for_each host must not fail on the adopted record: %v", wrapped.Steps[0])

	a := normalizeVolatile(finalState(legacy))
	b := normalizeVolatile(finalState(wrapped))
	if !assert.Equal(t, a, b, "final record state diverged between legacy and migrated shapes") {
		t.Logf("legacy:   %s", mustJSON(a))
		t.Logf("migrated: %s", mustJSON(b))
	}
}

// finalState is the last executed step's record snapshot of the first
// sample — the state the chain leaves the record in.
func finalState(rep *dryrun.Report) map[string]interface{} {
	steps := rep.Steps[0]
	last := steps[len(steps)-1]
	if last.After == nil {
		return nil
	}
	return last.After
}

// normalizeVolatile strips time-derived keys anywhere in the tree — both
// shapes run the same processors, but wall-clock reads land in fields like
// document.updated_at and would otherwise false-diff two back-to-back runs.
func normalizeVolatile(v interface{}) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(x))
		for k, val := range x {
			if volatileKey(k) {
				continue
			}
			out[k] = normalizeVolatile(val)
		}
		return out
	case map[string]string:
		out := make(map[string]interface{}, len(x))
		for k, val := range x {
			if volatileKey(k) {
				continue
			}
			out[k] = normalizeVolatile(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(x))
		for i, val := range x {
			out[i] = normalizeVolatile(val)
		}
		return out
	default:
		return v
	}
}

func volatileKey(k string) bool {
	lk := strings.ToLower(k)
	for _, suffix := range []string{"timestamp", "_at", "duration", "duration_ms", "_time", "time_ms"} {
		if strings.HasSuffix(lk, suffix) {
			return true
		}
	}
	return false
}

func mustJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "<marshal error>"
	}
	return string(b)
}
