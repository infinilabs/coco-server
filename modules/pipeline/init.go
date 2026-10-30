/* Copyright © INFINI LTD.
 * Web: https://infini.ltd */

// Package pipeline registers coco's chain generator on the framework's
// unified pipeline endpoints: POST /pipeline/ai-generate (LLM chain
// generation, see ai_generate.go) and POST /pipeline/dry-run (replay).
// Pipeline CRUD itself stays on the framework's generic /pipelines/
// routes; the former studio endpoints are gone by design.
package pipeline

import (
	_ "infini.sh/coco/core/codecs" // register coco_document/coco_attachment record codecs
)
