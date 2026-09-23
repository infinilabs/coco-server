/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package skill

import (
	"errors"

	"infini.sh/coco/core"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/api/crud"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

const Category = "coco"
const Resource = "skill"

var (
	errTitleRequired        = errors.New("title is required")
	errInstructionsRequired = errors.New("instructions is required")
	errBuiltinNotDeletable  = errors.New("builtin skills cannot be deleted, disable them instead")
)

var (
	readPermission   = security.GetSimplePermission(Category, Resource, string(security.Read))
	createPermission = security.GetSimplePermission(Category, Resource, string(security.Create))
	updatePermission = security.GetSimplePermission(Category, Resource, string(security.Update))
	deletePermission = security.GetSimplePermission(Category, Resource, string(security.Delete))
	searchPermission = security.GetSimplePermission(Category, Resource, string(security.Search))
)

type APIHandler struct {
	api.Handler
}

func init() {
	// skills shape the system prompt of every assistant — administration
	// only, not part of the widget role
	security.GetOrInitPermissionKeys(readPermission, createPermission, updatePermission, deletePermission, searchPermission)
	security.RegisterPermissionsToRole(security.RoleAdmin, readPermission, createPermission, updatePermission, deletePermission, searchPermission)

	crud.RegisterCRUD[core.Skill](skillConfig())

	handler := APIHandler{}

	// quick-config export for external agents: drop-in client config and a
	// SKILLS.md generated from the user's real skill records
	api.HandleUIMethod(api.GET, "/skill/_export/mcp_json", handler.exportMCPJSON,
		api.RequireLogin(), api.RequirePermission(searchPermission))
	api.HandleUIMethod(api.GET, "/skill/_export/skills_md", handler.exportSkillsMD,
		api.RequireLogin(), api.RequirePermission(searchPermission))

	// seed builtin skills once per process; user edits survive reboots
	global.RegisterFuncAfterSetup(func() {
		seedBuiltinSkills()
	})
}

func skillConfig() crud.Config[core.Skill] {
	return crud.Config[core.Skill]{
		Prefix:             "/skill",
		Resource:           Resource,
		Permission:         func(action string) api.PermissionKey { return security.GetSimplePermission(Category, Resource, action) },
		DefaultQueryFields: []string{"name", "title", "description", "combined_fulltext"},
		MCP:                true,
		MCPDescs: map[string]string{
			crud.ActionSearch: "Search skills (prompt overlays that shape how coco assistants behave)",
			crud.ActionRead:   "Get a skill by id, including its full instruction markdown",
		},
		PrepareCreate: func(obj *core.Skill) error {
			if obj.Title == "" {
				return errTitleRequired
			}
			if obj.Instructions == "" {
				return errInstructionsRequired
			}
			if obj.Name == "" {
				obj.Name = "skill-" + util.GetUUID()[:8]
			}
			// user-created skills are never builtin
			obj.Builtin = false
			return nil
		},
		// builtin is preserved on updates: seeds can be edited and disabled,
		// but a patch cannot promote a user skill to builtin (or demote one)
		ProtectedFields: []string{"builtin"},
		GuardDelete: func(obj *core.Skill) error {
			if obj.Builtin {
				return errBuiltinNotDeletable
			}
			return nil
		},
	}
}
