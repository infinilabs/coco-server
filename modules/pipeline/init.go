/* Copyright © INFINI LTD.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

// Package pipeline adds the pipeline studio APIs: a step-by-step dry-run of
// a processor chain over sample documents (the same execution contract as
// process_documents) and an AI generate/refine endpoint that turns sample
// documents plus a natural-language requirement into a validated chain.
// Pipeline CRUD itself stays on the framework's generic /pipelines/ routes.
package pipeline

import (
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/security"
)

type APIHandler struct {
	api.Handler
}

func init() {
	// Reuse the framework's generic pipeline permission keys so studio
	// access is granted exactly with pipeline CRUD access.
	searchPermission := security.GetOrInitPermission("generic", "pipeline", string(security.Search))
	updatePermission := security.GetOrInitPermission("generic", "pipeline", string(security.Update))
	security.GetOrInitPermissionKeys(searchPermission, updatePermission)

	handler := APIHandler{}

	api.HandleUIMethod(api.POST, "/pipeline/studio/test", handler.testPipelineChain,
		api.RequireLogin(), api.RequirePermission(searchPermission))
	api.HandleUIMethod(api.POST, "/pipeline/studio/ai-generate", handler.aiGeneratePipelineChain,
		api.RequireLogin(), api.RequirePermission(updatePermission))
}
