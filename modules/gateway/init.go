/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

// Package gateway is Coco's 接入中心 (instance management center): the
// management surface over gateway/agent instances that register themselves
// through the framework's managed-config protocol.
//
// Two surfaces, two auth models — deliberately separate:
//
//	protocol  /instance/_register, /configs/_sync, /instance/:id/_exchange_token
//	          (framework configs server; static bearer tokens / enrollment
//	          tickets — what the gateway instances present)
//	management  /gateway/instances...
//	          (this module; Coco login session + permission keys — what the
//	          admin UI calls)
//
// The protocol side is enabled by configs_server.Setup() below (opt-out via
// configs.server.enabled: false); this module adds the UI-facing queries and
// actions on top of the same stored records, so the two never share routes.
package gateway

import (
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/security"
	configs_server "infini.sh/framework/modules/configs/server"
)

const Category = "coco"

const instanceResource = "gateway_instance"

var (
	readPermission   = security.GetSimplePermission(Category, instanceResource, string(security.Read))
	searchPermission = security.GetSimplePermission(Category, instanceResource, string(security.Search))
	updatePermission = security.GetSimplePermission(Category, instanceResource, string(security.Update))
	deletePermission = security.GetSimplePermission(Category, instanceResource, string(security.Delete))
)

type APIHandler struct {
	api.Handler
}

func init() {
	security.GetOrInitPermissionKeys(
		readPermission,
		searchPermission,
		updatePermission,
		deletePermission,
	)

	global.RegisterFuncBeforeSetup(func() {
		// configs_server.Setup reads configs.server.* from the loaded
		// config — it must run after config init, before module start.
		// It registers the protocol routes, the ORM schemas
		// (instance / managed-configs / instance-tokens / enrollment
		// tokens) and the reverse-channel websocket commands.
		configs_server.Setup()

		handler := APIHandler{}
		api.HandleUIMethod(api.GET, "/gateway/instances", handler.listInstances,
			api.RequireLogin(), api.RequirePermission(searchPermission),
			api.MCPTool("gateway_list_instances", "List registered gateway/agent instances with status (pending/approved), online state (heartbeat-derived) and reverse-channel connectivity"))
		api.HandleUIMethod(api.GET, "/gateway/instances/:id/detail", handler.instanceDetail,
			api.RequireLogin(), api.RequirePermission(readPermission),
			api.MCPTool("gateway_instance_detail", "Get one instance's full registration info, assigned managed configs and reverse-channel state"))
		api.HandleUIMethod(api.POST, "/gateway/instances/:id/_approve", handler.approveInstance,
			api.RequireLogin(), api.RequirePermission(updatePermission),
			api.MCPTool("gateway_approve_instance", "Approve a pending instance admission: flips status to approved and mints its manager token (delivered to the instance on its next register/sync)"))
		api.HandleUIMethod(api.PUT, "/gateway/instances/:id/_meta", handler.updateInstanceMeta,
			api.RequireLogin(), api.RequirePermission(updatePermission),
			api.MCPTool("gateway_update_instance_meta", "Update server-owned instance metadata: alias, note, groups (config delivery targeting); heartbeat upserts preserve these"))
		api.HandleUIMethod(api.DELETE, "/gateway/instances/:id", handler.forgetInstance,
			api.RequireLogin(), api.RequirePermission(deletePermission),
			api.MCPTool("gateway_forget_instance", "Remove an instance: deletes its registration record, credentials and assigned per-instance configs; a live instance re-registers as a brand-new pending one"))
		api.HandleUIMethod(api.GET, "/gateway/instances/:id/proxy", handler.instanceProxy,
			api.RequireLogin(), api.RequirePermission(readPermission),
			api.MCPTool("gateway_instance_proxy", "Fetch a whitelisted read-only endpoint from a connected instance through its reverse channel: /_info /_whoami /health /stats /queue/stats /queue/<id>/_scroll /tasks/ /pipeline/tasks/ /pipeline/processors /pipeline/codecs /pipeline/lanes /pipeline/task/<id>/stages /pipelines/<id>/versions /debug/goroutines /config/ /config/runtime /logging/files /logging/tail"))
	})
}
