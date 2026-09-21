/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/service"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/api/crud"
	"infini.sh/framework/core/util"
)

// registerAssistantCRUD replaces the hand-written create/update/delete/
// search endpoints with the framework generator. The get endpoint stays
// hand-written (service.GetAssistant is cache-first, uses DirectAccess
// and expands the datasource wildcard), so ActionRead is skipped and
// registered separately in init.go.
func registerAssistantCRUD() {
	crud.RegisterCRUD[core.Assistant](crud.Config[core.Assistant]{
		Prefix:      "/assistant",
		Resource:    "assistant",
		SkipActions: []string{crud.ActionRead},

		Permission: func(action string) api.PermissionKey {
			switch action {
			case crud.ActionCreate:
				return createAssistantPermission
			case crud.ActionUpdate:
				return updateAssistantPermission
			case crud.ActionDelete:
				return deleteAssistantPermission
			case crud.ActionSearch:
				return searchAssistantPermission
			}
			return ""
		},

		ExtraOptions: func(action string) []api.Option {
			if action == crud.ActionSearch {
				return []api.Option{api.Feature(core.FeatureCORS), api.AllowOPTIONSS()}
			}
			return nil
		},

		DefaultQueryFields: []string{"name", "name.pinyin", "combined_fulltext"},
		SharingResource:    "assistant",
		MCP:                true,
		MCPDescs: map[string]string{
			crud.ActionCreate: "Create a new assistant",
			crud.ActionUpdate: "Update an assistant",
			crud.ActionDelete: "Delete an assistant",
			crud.ActionSearch: "Search assistants",
		},

		PrepareCreate: func(obj *core.Assistant) error {
			obj.Builtin = false
			return nil
		},

		UpdateMode:      crud.UpdateModeFull,
		ProtectedFields: []string{"created", "builtin"},
		PrepareUpdate: func(obj *core.Assistant, delta util.MapStr) error {
			if obj.Builtin {
				if _, renaming := delta["name"]; renaming {
					return errStr("name of a built-in assistant cannot be changed")
				}
			}
			return nil
		},
		PostUpdate: func(obj *core.Assistant) error {
			common.GeneralObjectCache.Delete(core.AssistantCachePrimary, obj.ID)
			service.ClearAssistantsCache()
			return nil
		},
		PostDelete: func(obj *core.Assistant) error {
			common.GeneralObjectCache.Delete(core.AssistantCachePrimary, obj.ID)
			service.ClearAssistantsCache()
			return nil
		},
	})
}

type errStr string

func (e errStr) Error() string { return string(e) }
