/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package process_documents

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
	"infini.sh/framework/core/util"
)

func TestApplyProcessingOutcomeSuccess(t *testing.T) {
	doc := core.Document{Status: core.DocumentStatusIndexing, ErrorMessage: "old failure"}
	applyProcessingOutcome(&doc, true, "enrich_documents", 1234, "")

	assert.Equal(t, core.DocumentStatusCompleted, doc.Status)
	assert.True(t, doc.Processed)
	assert.Empty(t, doc.ErrorMessage)

	runs, ok := doc.Metadata["pipeline_runs"].([]interface{})
	require.True(t, ok)
	require.Len(t, runs, 1)
	rec := runs[0].(util.MapStr)
	assert.Equal(t, "enrich_documents", rec["pipeline"])
	assert.Equal(t, int64(1234), rec["took_ms"])
	assert.Equal(t, true, rec["success"])
}

func TestApplyProcessingOutcomeFailure(t *testing.T) {
	doc := core.Document{Status: core.DocumentStatusIndexing}
	long := make([]byte, 900)
	for i := range long {
		long[i] = 'x'
	}
	applyProcessingOutcome(&doc, false, "enrich_documents", 50, string(long))

	assert.Equal(t, core.DocumentStatusFailed, doc.Status)
	assert.False(t, doc.Processed)
	assert.Len(t, doc.ErrorMessage, 500, "error message must be truncated")
	rec := doc.Metadata["pipeline_runs"].([]interface{})[0].(util.MapStr)
	assert.Equal(t, false, rec["success"])
}

func TestPassthroughOutcomePreservesFailure(t *testing.T) {
	// a failed re-run whose datasource lost its pipeline keeps the honest
	// failed state — passthrough must not whitewash it
	doc := core.Document{Status: core.DocumentStatusFailed, ErrorMessage: "boom"}
	applyPassthroughOutcome(&doc, "no-pipeline-configured")
	assert.Equal(t, core.DocumentStatusFailed, doc.Status)
	assert.Equal(t, "boom", doc.ErrorMessage)
	assert.False(t, doc.Processed)

	fresh := core.Document{Status: core.DocumentStatusIndexing}
	applyPassthroughOutcome(&fresh, "no-datasource")
	assert.Equal(t, core.DocumentStatusCompleted, fresh.Status)
	rec := fresh.Metadata["pipeline_runs"].([]interface{})[0].(util.MapStr)
	assert.Equal(t, "no-datasource", rec["passthrough"])
}

func TestPipelineRunsCappedNewestFirst(t *testing.T) {
	doc := core.Document{}
	for i := 0; i < 8; i++ {
		applyProcessingOutcome(&doc, true, "p", int64(i), "")
	}
	runs := doc.Metadata["pipeline_runs"].([]interface{})
	require.Len(t, runs, pipelineRunsCap)
	first := runs[0].(util.MapStr)
	assert.Equal(t, int64(7), first["took_ms"], "newest run must be first")
}

func TestIngestDedupGateEnvSwitch(t *testing.T) {
	assert.True(t, ingestDedupGateEnabled(), "default must be on")
	for _, off := range []string{"off", "false", "0", "OFF"} {
		t.Setenv("INGEST_DEDUP_GATE", off)
		assert.False(t, ingestDedupGateEnabled(), "%s must disable the gate", off)
	}
	t.Setenv("INGEST_DEDUP_GATE", "on")
	assert.True(t, ingestDedupGateEnabled())
}
