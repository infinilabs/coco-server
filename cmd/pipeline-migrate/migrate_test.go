/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infinilabs.com */

package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/cmd/internal/pipescan"
	"infini.sh/framework/core/pipeline"
)

func TestMain(m *testing.M) {
	pipescan.RegisterCLIEnvironment()
	m.Run()
}

func step(name string) map[string]interface{} {
	return map[string]interface{}{name: map[string]interface{}{}}
}

func forEachCodec(m map[string]interface{}) string {
	fe, ok := m["for_each"].(map[string]interface{})
	if !ok {
		return ""
	}
	c, _ := fe["codec"].(string)
	return c
}

func forEachSteps(m map[string]interface{}) []map[string]interface{} {
	fe, _ := m["for_each"].(map[string]interface{})
	procs, _ := fe["processor"].([]map[string]interface{})
	return procs
}

// document-namespace chain wraps whole into one for_each{coco_document}
// and the migrated shape validates fully clean through the real gate
func TestMigrateChain_DocumentRun(t *testing.T) {
	procs := []map[string]interface{}{
		step("file_type_detection"),
		step("file_metadata"),
		step("extract_tags"),
	}
	out, err := migrateChain(procs)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "coco_document", forEachCodec(out[0]))
	assert.Len(t, forEachSteps(out[0]), 3)

	errs, warns := pipescan.Validate(&pipeline.PipelineConfigV2{Name: "probe", Processors: out})
	assert.Empty(t, errs, "migrated shape must compile clean")
	assert.Empty(t, warns, "migrated shape must carry no contract warnings")
}

// attachment-namespace steps get their own codec — the single-codec wrap
// the first draft produced fails validation here, this must not
func TestMigrateChain_AttachmentRun(t *testing.T) {
	procs := []map[string]interface{}{step("attachment_text_extraction")}
	out, err := migrateChain(procs)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "coco_attachment", forEachCodec(out[0]))

	errs, warns := pipescan.Validate(&pipeline.PipelineConfigV2{Name: "probe", Processors: out})
	assert.Empty(t, errs)
	assert.Empty(t, warns)
}

// namespace alternation segments into consecutive hosts, order preserved
func TestMigrateChain_NamespaceSegmentation(t *testing.T) {
	procs := []map[string]interface{}{
		step("file_type_detection"),        // document
		step("attachment_text_extraction"), // attachment
		step("extract_tags"),               // document
	}
	out, err := migrateChain(procs)
	require.NoError(t, err)
	require.Len(t, out, 3, "three maximal runs → three hosts")
	assert.Equal(t, "coco_document", forEachCodec(out[0]))
	assert.Equal(t, "coco_attachment", forEachCodec(out[1]))
	assert.Equal(t, "coco_document", forEachCodec(out[2]))
}

// batch-scoped steps refuse: segmentation there changes semantics
func TestMigrateChain_BatchScopedRefused(t *testing.T) {
	procs := []map[string]interface{}{
		step("file_type_detection"),
		step("connector_dispatcher"),
	}
	_, err := migrateChain(procs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manual segmentation")
}

// the source chain itself must be the legacy shape these tests assume
func TestLegacyShapeWarns(t *testing.T) {
	procs := []map[string]interface{}{step("file_type_detection"), step("file_metadata")}
	errs, warns := pipescan.Validate(&pipeline.PipelineConfigV2{Name: "probe", Processors: procs})
	require.Empty(t, errs)
	require.NotEmpty(t, warns, "record processors at the batch root must warn")
	assert.True(t, strings.Contains(warns[0].Message, "record scope"))
}
