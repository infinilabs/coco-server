/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package gateway

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/model"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	configs_server "infini.sh/framework/modules/configs/server"
)

// instanceOfflineAfter: online = a /configs/_sync heartbeat within 3× the
// framework client's default 30s sync interval.
const instanceOfflineAfter = 90 * time.Second

// proxyReadExact lists instance-local read-only endpoints the detail panel
// may fetch through the reverse channel (exact path match). Same endpoint
// set Logpilot's ingestion center proxies — the framework's standard
// observability surface every managed instance serves.
var proxyReadExact = map[string]string{
	"/_info":               "version",  // version, build, host, cpu, memory summary
	"/_whoami":             "whoami",
	"/health":              "health",
	"/stats":               "stats",
	"/queue/stats":         "queues", // queue overview: depth/size/consumers
	"/tasks/":              "tasks",  // scheduled tasks
	"/pipeline/tasks/":     "pipelines", // running pipeline tasks
	"/pipeline/processors": "processors",
	"/pipeline/codecs":     "codecs",
	"/pipeline/lanes":      "lanes", // lane depth/drops/spills observability
	"/debug/goroutines":    "routines",
	"/config/":             "configs", // local config files (name/version/size/managed)
	"/config/runtime":      "runtime_config",
}

// proxyReadPrefixes are read-only GET prefixes (paths carry dynamic ids).
// Queue message browsing (/queue/<id>/_scroll) is the instance-side
// debug-consumer API: it never touches real consumer offsets. Logging is
// /logging/files + /logging/tail (tailable .log/.json only, instance-side
// path-guarded). /pipeline/task/<id>/stages|flight and /pipelines/<id>/versions
// give per-task drill-down and version history.
var proxyReadPrefixes = []string{
	"/queue/",
	"/logging/",
	"/pipeline/task/",
	"/pipelines/",
}

// matchProxyRead resolves a proxied GET path against the whitelist.
func matchProxyRead(path string) (string, bool) {
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	if label, ok := proxyReadExact[path]; ok {
		return label, true
	}
	for _, p := range proxyReadPrefixes {
		if strings.HasPrefix(path, p) {
			switch p {
			case "/logging/":
				return "logs", true
			case "/queue/":
				return "queues", true
			case "/pipeline/task/":
				return "pipelines", true
			case "/pipelines/":
				return "pipeline_config", true
			}
		}
	}
	return "", false
}

// listInstances — GET /gateway/instances?type=gateway|agent
// Registered instances with admission status and heartbeat-derived online
// state (managed_last_sync_at label).
func (h *APIHandler) listInstances(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	appType := strings.TrimSpace(req.URL.Query().Get("type"))

	ctx := orm.NewContextWithParent(req.Context()).DirectAccess()
	orm.WithModel(ctx, &model.Instance{})
	qb := orm.NewQuery().Size(500)
	if appType != "" {
		qb = qb.Filter(orm.TermQuery("application.name", appType))
	}
	res, err := orm.SearchV2(ctx, qb)
	if err != nil || res == nil {
		h.WriteError(w, "instance query failed", http.StatusInternalServerError)
		return
	}
	instances, _, _ := elastic.DecodeHits[model.Instance](res)

	now := time.Now()
	out := make([]util.MapStr, 0, len(instances))
	for _, inst := range instances {
		lastSync := int64(0)
		if v, ok := inst.Labels[configs_server.LabelLastSyncAt]; ok {
			if ms, err := strconv.ParseInt(v, 10, 64); err == nil {
				lastSync = ms
			}
		}
		online := lastSync > 0 && now.Sub(time.UnixMilli(lastSync)) < instanceOfflineAfter
		out = append(out, util.MapStr{
			"id":       inst.ID,
			"name":     inst.Name,
			"type":     inst.Application.Name,
			"status":   inst.Status, // pending | approved
			"online":   online,
			"version":  inst.Application.Version.VersionNumber,
			"endpoint": inst.Endpoint,
			"ip":       inst.Network.MajorIP,
			"hostname": hostNameOf(&inst),
			"os":       hostOSOf(&inst),
			"labels":   inst.Labels,
			"tags":     inst.Tags,
			"groups":   inst.Groups,
			"alias":    inst.Alias,
			"note":     inst.Description,
			"last_sync": lastSync,
			"reverse_connected": configs_server.ReverseIsConnected(inst.ID),
			"created":           inst.Created,
		})
	}
	h.WriteJSON(w, util.MapStr{"total": len(out), "items": out}, http.StatusOK)
}

// Host is a pointer and a bare-bones sync client may omit it entirely —
// guard every read or a minimal heartbeat record panics the list.
func hostNameOf(inst *model.Instance) string {
	if inst.Host == nil {
		return ""
	}
	return inst.Host.Name
}

func hostOSOf(inst *model.Instance) string {
	if inst.Host == nil {
		return ""
	}
	return inst.Host.OS.Name
}

// instanceDetail — GET /gateway/instances/:id/detail
// Registration info + assigned managed configs (per-instance, "*" catch-all
// and group-targeted rows) + reverse-channel state.
func (h *APIHandler) instanceDetail(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	ctx := orm.NewContextWithParent(req.Context()).DirectAccess()
	orm.WithModel(ctx, &model.Instance{})

	inst := model.Instance{}
	inst.ID = id
	exists, err := orm.GetV2(ctx, &inst)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	lastSync := int64(0)
	if v, ok := inst.Labels[configs_server.LabelLastSyncAt]; ok {
		if ms, err := strconv.ParseInt(v, 10, 64); err == nil {
			lastSync = ms
		}
	}

	out := util.MapStr{
		"id":                inst.ID,
		"name":              inst.Name,
		"alias":             inst.Alias,
		"type":              inst.Application.Name,
		"version":           inst.Application.Version.VersionNumber,
		"endpoint":          inst.Endpoint,
		"ip":                inst.Network.MajorIP,
		"hostname":          hostNameOf(&inst),
		"os":                hostOSOf(&inst),
		"labels":            inst.Labels,
		"tags":              inst.Tags,
		"groups":            inst.Groups,
		"note":              inst.Description,
		"status":            inst.Status,
		"services":          inst.Services,
		"last_sync":         lastSync,
		"created":           inst.Created,
		"reverse_connected": configs_server.ReverseIsConnected(id),
	}

	// assigned dynamic configs: rows for this instance, the "*" catch-all,
	// and group-targeted rows intersecting this instance's groups
	assigned := []util.MapStr{}
	orm.WithModel(ctx, &configs_server.ManagedConfig{})
	qb := orm.NewQuery().
		Filter(orm.ShouldQuery(
			orm.TermQuery("instance_id", id),
			orm.TermQuery("instance_id", configs_server.AllInstancesID),
		)).
		Size(200)
	if res, err := orm.SearchV2(ctx, qb); err == nil && res != nil {
		if configs, _, _ := elastic.DecodeHits[configs_server.ManagedConfig](res); configs != nil {
			for _, c := range configs {
				assigned = append(assigned, util.MapStr{
					"name":    c.Name,
					"version": c.Version,
					"size":    len(c.Content),
					"target":  c.InstanceID,
				})
			}
		}
	}
	if len(inst.Groups) > 0 {
		groupSet := map[string]struct{}{}
		for _, g := range inst.Groups {
			groupSet[strings.TrimSpace(g)] = struct{}{}
		}
		if res, err := orm.SearchV2(ctx, orm.NewQuery().
			Filter(orm.ExistsQuery("groups")).Size(200)); err == nil && res != nil {
			if configs, _, _ := elastic.DecodeHits[configs_server.ManagedConfig](res); configs != nil {
				for _, c := range configs {
					matched := false
					for _, g := range c.Groups {
						if _, ok := groupSet[strings.TrimSpace(g)]; ok {
							matched = true
							break
						}
					}
					if !matched {
						continue
					}
					assigned = append(assigned, util.MapStr{
						"name":    c.Name,
						"version": c.Version,
						"size":    len(c.Content),
						"target":  "groups: " + strings.Join(c.Groups, ","),
					})
				}
			}
		}
	}
	out["assigned_configs"] = assigned

	h.WriteJSON(w, out, http.StatusOK)
}

// approveInstance — POST /gateway/instances/:id/_approve
// Flip a pending instance to approved and mint its manager token; the
// instance picks the credential up on its next register/sync. Mirrors the
// framework's token-gated protocol handler, but on Coco's session+permission
// surface so the admin UI can call it with its own login.
func (h *APIHandler) approveInstance(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	ctx := orm.NewContextWithParent(req.Context()).DirectAccess()
	orm.WithModel(ctx, &model.Instance{})

	inst := model.Instance{}
	inst.ID = id
	exists, err := orm.GetV2(ctx, &inst)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	if inst.Status != configs_server.StatusApproved {
		inst.Status = configs_server.StatusApproved
		if err := orm.Save(ctx, &inst); err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	token, err := configs_server.MintPublicManagerToken(id, inst.Name)
	if err != nil {
		h.WriteError(w, "mint manager token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteJSON(w, util.MapStr{"id": id, "status": configs_server.StatusApproved, "manager_token": token}, http.StatusOK)
}

// updateInstanceMeta — PUT /gateway/instances/:id/_meta
// body: {"alias": "...", "note": "...", "groups": ["prod", "edge"]}
// Server-owned fields: heartbeat upserts preserve them (unlike self-reported
// registration fields). Missing = keep; empty string = clear (groups
// replace wholesale).
func (h *APIHandler) updateInstanceMeta(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	var body struct {
		Alias  *string  `json:"alias"`
		Note   *string  `json:"note"`
		Groups []string `json:"groups"`
	}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := orm.NewContextWithParent(req.Context()).DirectAccess()
	orm.WithModel(ctx, &model.Instance{})
	inst := model.Instance{}
	inst.ID = id
	exists, err := orm.GetV2(ctx, &inst)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	if body.Alias != nil {
		inst.Alias = strings.TrimSpace(*body.Alias)
	}
	if body.Note != nil {
		inst.Description = strings.TrimSpace(*body.Note)
	}
	if body.Groups != nil {
		seen := map[string]struct{}{}
		groups := make([]string, 0, len(body.Groups))
		for _, g := range body.Groups {
			g = strings.TrimSpace(g)
			if g == "" {
				continue
			}
			if _, dup := seen[g]; dup {
				continue
			}
			seen[g] = struct{}{}
			groups = append(groups, g)
		}
		inst.Groups = groups
	}

	if err := orm.Save(ctx, &inst); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteJSON(w, util.MapStr{
		"id":     id,
		"alias":  inst.Alias,
		"note":   inst.Description,
		"groups": inst.Groups,
	}, http.StatusOK)
}

// forgetInstance — DELETE /gateway/instances/:id
// Remove the registration record + per-instance credentials + assigned
// per-instance configs. A live instance's next sync gets 401 and it
// re-registers as a brand-new pending instance.
func (h *APIHandler) forgetInstance(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	ctx := orm.NewContextWithParent(req.Context()).DirectAccess()
	orm.WithModel(ctx, &model.Instance{})

	inst := model.Instance{}
	inst.ID = id
	exists, err := orm.GetV2(ctx, &inst)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err != nil || !exists {
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	if err := orm.Delete(ctx, &inst); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// per-instance token records
	orm.WithModel(ctx, &configs_server.InstanceToken{})
	if res, err := orm.SearchV2(ctx, orm.NewQuery().
		Filter(orm.TermQuery("instance_id", id)).Size(100)); err == nil && res != nil {
		tokens, _, _ := elastic.DecodeHits[configs_server.InstanceToken](res)
		for _, t := range tokens {
			_ = orm.Delete(ctx, &t)
		}
	}
	// assigned per-instance dynamic configs
	orm.WithModel(ctx, &configs_server.ManagedConfig{})
	if res, err := orm.SearchV2(ctx, orm.NewQuery().
		Filter(orm.TermQuery("instance_id", id)).Size(500)); err == nil && res != nil {
		configs, _, _ := elastic.DecodeHits[configs_server.ManagedConfig](res)
		for _, c := range configs {
			_ = orm.Delete(ctx, &c)
		}
	}

	h.WriteJSON(w, util.MapStr{"forgotten": true, "id": id, "name": inst.Name}, http.StatusOK)
}

// instanceProxy — GET /gateway/instances/:id/proxy?path=/_info
// Whitelisted read-only fetch through the instance's reverse channel (the
// one-way model: instances dial out, the manager calls back over that
// websocket). GET only.
func (h *APIHandler) instanceProxy(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	path := req.URL.Query().Get("path")

	label, allowed := matchProxyRead(path)
	if !allowed {
		allowedList := make([]string, 0, len(proxyReadExact)+len(proxyReadPrefixes))
		for k := range proxyReadExact {
			allowedList = append(allowedList, k)
		}
		allowedList = append(allowedList, proxyReadPrefixes...)
		h.WriteError(w, "path not allowed; GET: "+strings.Join(allowedList, " "), http.StatusForbidden)
		return
	}
	if !configs_server.ReverseIsConnected(id) {
		h.WriteError(w, "reverse channel not connected", http.StatusServiceUnavailable)
		return
	}

	targetPath := path
	q := req.URL.Query()
	q.Del("path")
	if extra := q.Encode(); extra != "" {
		targetPath += "?" + extra
	}

	res, err := configs_server.ReverseProxyRequest(id, &util.Request{Method: http.MethodGet, Path: targetPath})
	if err != nil {
		h.WriteError(w, "reverse call failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	h.WriteJSON(w, util.MapStr{
		"instance_id": id,
		"label":       label,
		"path":        path,
		"status":      res.StatusCode,
		"data":        json.RawMessage(res.Body),
	}, http.StatusOK)
}
