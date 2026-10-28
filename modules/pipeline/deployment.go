/* Copyright © INFINI LTD.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package pipeline

// Fleet pipeline deployments (接入中心 · 管道部署): CRUD for the desired
// state (core.PipelineDeployment), a 15s reconciler that converges targets
// by delivering the pipeline's config file through the framework configs
// channel, and a status projection the studio renders (desired vs
// acknowledged per instance).
//
// Delivery primitive: a configsserver.ManagedConfig row per (instance,
// config file). Instances pull it on their periodic /configs/_sync and the
// framework client writes the file into the local config dir, where
// fsnotify hot-reloads pipelines — so a converged instance is one whose
// last sync is newer than the managed row's update stamp.

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"gopkg.in/yaml.v3"

	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/model"
	"infini.sh/framework/core/orm"
	fpipeline "infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	configsserver "infini.sh/framework/modules/configs/server"

	"infini.sh/coco/core"
)

const (
	deploymentConfigPrefix  = "pipeline-deploy-"
	reconcileInterval       = 15 * time.Second
	deploymentInstanceLimit = 500
)

type DeploymentAPIHandler struct {
	api.Handler
}

func deploymentPermission(action string) security.PermissionKey {
	return security.GetOrInitPermission("generic", "pipeline", action)
}

func managedConfigName(deploymentID string) string {
	return deploymentConfigPrefix + deploymentID + ".yml"
}

func init() {
	h := DeploymentAPIHandler{}
	read := deploymentPermission(security.Read)
	create := deploymentPermission(security.Create)
	update := deploymentPermission(security.Update)
	remove := deploymentPermission(security.Delete)

	api.HandleUIMethod(api.GET, "/pipeline/deployments", h.listDeployments, api.RequireLogin(), api.RequirePermission(read))
	api.HandleUIMethod(api.POST, "/pipeline/deployments", h.createDeployment, api.RequireLogin(), api.RequirePermission(create))
	api.HandleUIMethod(api.GET, "/pipeline/deployments/:id/status", h.deploymentStatus, api.RequireLogin(), api.RequirePermission(read))
	api.HandleUIMethod(api.PUT, "/pipeline/deployments/:id", h.updateDeployment, api.RequireLogin(), api.RequirePermission(update))
	api.HandleUIMethod(api.DELETE, "/pipeline/deployments/:id", h.deleteDeployment, api.RequireLogin(), api.RequirePermission(remove))

	global.RegisterFuncAfterSetup(func() {
		global.RegisterBackgroundCallback(&global.BackgroundTask{
			Tag:      "pipeline deployment reconciler",
			Interval: reconcileInterval,
			Func:     reconcileDeployments,
		})
	})
}

// ─── CRUD ───────────────────────────────────────────────────────────────────

type deploymentBody struct {
	Name       string                    `json:"name"`
	PipelineID string                    `json:"pipeline_id"`
	Enabled    bool                      `json:"enabled"`
	Placement  core.DeploymentPlacement  `json:"placement"`
}

func (h *DeploymentAPIHandler) listDeployments(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	ctx := orm.NewContextWithParent(req.Context()).DirectAccess()
	orm.WithModel(ctx, &core.PipelineDeployment{})
	q := orm.NewQuery().Size(deploymentInstanceLimit)
	res, err := orm.SearchV2(ctx, q)
	if err != nil || res == nil {
		h.WriteError(w, "deployment query failed", http.StatusInternalServerError)
		return
	}
	items, total, _ := elastic.DecodeHits[core.PipelineDeployment](res)
	if items == nil {
		items = []core.PipelineDeployment{}
	}
	h.WriteJSON(w, util.MapStr{"total": total, "items": items}, http.StatusOK)
}

func (h *DeploymentAPIHandler) createDeployment(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	body := deploymentBody{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.Error400(w, err.Error())
		return
	}
	if strings.TrimSpace(body.Name) == "" || body.PipelineID == "" {
		h.Error400(w, "name and pipeline_id are required")
		return
	}
	if body.Placement.Strategy == "" {
		body.Placement.Strategy = "all"
	}

	dep := core.PipelineDeployment{
		Name:       strings.TrimSpace(body.Name),
		PipelineID: body.PipelineID,
		Enabled:    body.Enabled,
		Version:    1,
		Placement:  body.Placement,
	}
	ctx := orm.NewContextWithParent(req.Context()).DirectAccess()
	ctx.Refresh = orm.WaitForRefresh
	if err := orm.Create(ctx, &dep); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteCreatedOKJSON(w, dep.GetID())
}

func (h *DeploymentAPIHandler) updateDeployment(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	body := deploymentBody{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.Error400(w, err.Error())
		return
	}

	ctx := orm.NewContextWithParent(req.Context()).DirectAccess()
	orm.WithModel(ctx, &core.PipelineDeployment{})
	dep := core.PipelineDeployment{}
	dep.ID = id
	exists, err := orm.GetV2(ctx, &dep)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	// version bump only when the effective desired state changed, so an
	// unchanged save does not churn every target into PENDING
	effectiveChanged := dep.Name != body.Name ||
		dep.PipelineID != body.PipelineID ||
		dep.Enabled != body.Enabled ||
		!placementEquals(dep.Placement, body.Placement)

	dep.Name = body.Name
	dep.PipelineID = body.PipelineID
	dep.Enabled = body.Enabled
	dep.Placement = body.Placement
	if effectiveChanged {
		dep.Version++
	}

	if err := orm.Update(ctx, &dep); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteOKJSON(w, dep)
}

func placementEquals(a, b core.DeploymentPlacement) bool {
	if a.Strategy != b.Strategy || !sameStringList(a.Instances, b.Instances) {
		return false
	}
	if len(a.Selector) != len(b.Selector) {
		return false
	}
	for k, v := range a.Selector {
		if bv, ok := b.Selector[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

func (h *DeploymentAPIHandler) deleteDeployment(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	ctx := orm.NewContextWithParent(req.Context()).DirectAccess()
	orm.WithModel(ctx, &core.PipelineDeployment{})
	dep := core.PipelineDeployment{}
	dep.ID = id
	exists, err := orm.GetV2(ctx, &dep)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}
	if err := orm.Delete(ctx, &dep); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// recall pushed configs — the next reconciler pass won't re-push since
	// the deployment is gone
	recallDeploymentConfig(id)
	h.WriteOKJSON(w, util.MapStr{"_id": id, "result": "deleted"})
}

// recallDeploymentConfig removes every managed config row pushed for a
// deployment (any instance) so instances drop the file on their next sync.
func recallDeploymentConfig(deploymentID string) {
	ctx := orm.NewContext().DirectAccess()
	orm.WithModel(ctx, &configsserver.ManagedConfig{})
	name := managedConfigName(deploymentID)
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(deploymentInstanceLimit).Filter(orm.TermQuery("name", name)))
	if err != nil || res == nil {
		return
	}
	rows, _, _ := elastic.DecodeHits[configsserver.ManagedConfig](res)
	for i := range rows {
		if err := orm.Delete(ctx, &rows[i]); err != nil {
			log.Warnf("deployment: failed to recall managed config for instance %s: %v", rows[i].InstanceID, err)
		}
	}
}

// ─── reconciler ─────────────────────────────────────────────────────────────

func reconcileDeployments() {
	ctx := orm.NewContext().DirectAccess()
	orm.WithModel(ctx, &core.PipelineDeployment{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(deploymentInstanceLimit))
	if err != nil || res == nil {
		log.Warnf("deployment: reconciler query failed: %v", err)
		return
	}
	deployments, _, _ := elastic.DecodeHits[core.PipelineDeployment](res)
	if len(deployments) == 0 {
		return
	}

	instances := loadManagedInstances()
	for i := range deployments {
		dep := &deployments[i]
		if !dep.Enabled {
			recallDeploymentConfig(dep.ID)
			continue
		}
		content, rerr := renderDeploymentConfig(dep.PipelineID)
		if rerr != nil {
			log.Warnf("deployment [%s]: render pipeline config failed: %v", dep.Name, rerr)
			continue
		}
		for _, instID := range resolveDeploymentTargets(dep, instances) {
			if err := upsertDeploymentConfig(instID, dep, content); err != nil {
				log.Warnf("deployment [%s]: push to instance %s failed: %v", dep.Name, instID, err)
			}
		}
	}
}

func loadManagedInstances() []model.Instance {
	ctx := orm.NewContext().DirectAccess()
	orm.WithModel(ctx, &model.Instance{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(deploymentInstanceLimit))
	if err != nil || res == nil {
		return nil
	}
	instances, _, _ := elastic.DecodeHits[model.Instance](res)
	return instances
}

// resolveDeploymentTargets computes the concrete instance list: group
// selectors re-evaluate membership every tick (new group members
// auto-join), explicit instance lists pass through, and singleton trims to
// the first target.
func resolveDeploymentTargets(dep *core.PipelineDeployment, instances []model.Instance) []string {
	var cands []string
	selector := dep.Placement.Selector
	if len(selector) > 0 {
		groups := map[string]struct{}{}
		if g, ok := selector["group"]; ok {
			for _, part := range strings.Split(g, ",") {
				if part = strings.TrimSpace(part); part != "" {
					groups[part] = struct{}{}
				}
			}
		}
		labels := map[string]string{}
		for k, v := range selector {
			if k != "group" {
				labels[k] = v
			}
		}
		for _, inst := range instances {
			if inst.Status == "pending" {
				continue // unadmitted instances receive no configs
			}
			if len(groups) > 0 && !groupsIntersect(inst.Groups, groups) {
				continue
			}
			match := true
			for k, v := range labels {
				if inst.Labels[k] != v {
					match = false
					break
				}
			}
			if match {
				cands = append(cands, inst.ID)
			}
		}
	} else {
		cands = append(cands, dep.Placement.Instances...)
	}
	if dep.Placement.Strategy == "singleton" && len(cands) > 1 {
		cands = cands[:1]
	}
	return cands
}

func sameStringList(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := map[string]int{}
	for _, v := range a {
		counts[v]++
	}
	for _, v := range b {
		counts[v]--
		if counts[v] < 0 {
			return false
		}
	}
	return true
}

func groupsIntersect(values []string, set map[string]struct{}) bool {
	for _, v := range values {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}

func keysOf(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

// renderDeploymentConfig renders the referenced pipeline as an instance
// config file: a top-level `pipeline:` list the framework merges from its
// config dir and hot-reloads via fsnotify.
func renderDeploymentConfig(pipelineID string) (string, error) {
	ctx := orm.NewContext().DirectAccess()
	orm.WithModel(ctx, &fpipeline.PipelineConfigV2{})
	pc := fpipeline.PipelineConfigV2{}
	pc.ID = pipelineID
	exists, err := orm.GetV2(ctx, &pc)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("pipeline %s not found", pipelineID)
	}
	if len(pc.Processors) == 0 {
		return "", fmt.Errorf("pipeline %s has no processors", pipelineID)
	}

	entry := map[string]interface{}{
		"name":         pc.Name,
		"enabled":      true,
		"auto_start":   true,
		"keep_running": true,
		"processor":    pc.Processors,
	}
	file := struct {
		Pipeline []map[string]interface{} `yaml:"pipeline"`
	}{Pipeline: []map[string]interface{}{entry}}
	b, err := yaml.Marshal(file)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func upsertDeploymentConfig(instanceID string, dep *core.PipelineDeployment, content string) error {
	ctx := orm.NewContext().DirectAccess()
	orm.WithModel(ctx, &configsserver.ManagedConfig{})
	name := managedConfigName(dep.ID)

	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("instance_id", instanceID)).
		Filter(orm.TermQuery("name", name)))
	if err != nil {
		return err
	}
	rows, _, _ := elastic.DecodeHits[configsserver.ManagedConfig](res)
	if len(rows) > 0 {
		existing := &rows[0]
		if existing.Version == dep.Version && existing.Content == content {
			return nil // already converged
		}
		existing.Content = content
		existing.Version = dep.Version
		return orm.Update(ctx, existing)
	}
	mc := configsserver.ManagedConfig{
		InstanceID: instanceID,
		Name:       name,
		Content:    content,
		Version:    dep.Version,
	}
	return orm.Create(ctx, &mc)
}

// ─── status projection ──────────────────────────────────────────────────────

func (h *DeploymentAPIHandler) deploymentStatus(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	ctx := orm.NewContextWithParent(req.Context()).DirectAccess()
	orm.WithModel(ctx, &core.PipelineDeployment{})
	dep := core.PipelineDeployment{}
	dep.ID = id
	exists, err := orm.GetV2(ctx, &dep)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	instances := loadManagedInstances()
	byID := map[string]model.Instance{}
	for _, inst := range instances {
		byID[inst.ID] = inst
	}
	targets := resolveDeploymentTargets(&dep, instances)

	reports := make([]util.MapStr, 0, len(targets))
	ready := 0
	for _, tid := range targets {
		state, version, stale, reason := deploymentInstanceState(&dep, tid, byID[tid])
		if state == "running" {
			ready++
		}
		lastSeen := ""
		if inst, ok := byID[tid]; ok {
			if ms := lastSyncMillis(inst); ms > 0 {
				lastSeen = time.UnixMilli(ms).Format(time.RFC3339)
			}
		}
		reports = append(reports, util.MapStr{
			"instance_id": tid,
			"state":       state,
			"version":     version,
			"last_seen":   lastSeen,
			"stale":       stale,
			"reason":      reason,
		})
	}

	h.WriteJSON(w, util.MapStr{
		"deployment_id": dep.ID,
		"version":       dep.Version,
		"instances":     reports,
		"desired_count": len(targets),
		"ready_count":   ready,
	}, http.StatusOK)
}

// deploymentInstanceState derives the per-target state from the delivery
// record: no managed row (yet) = pending; row present but the instance
// hasn't synced since the row's update = pending; synced past the stamp =
// running. Offline instances are flagged stale. A disabled deployment
// reports stopped.
func deploymentInstanceState(dep *core.PipelineDeployment, instanceID string, inst model.Instance) (state string, version int64, stale bool, reason string) {
	if inst.ID == "" {
		return "pending", 0, true, "instance unknown"
	}
	if !instanceOnline(inst) {
		stale = true
	}
	if !dep.Enabled {
		return "stopped", 0, stale, "deployment disabled"
	}

	mc, ok := loadDeploymentConfig(instanceID, dep.ID)
	if !ok {
		return "pending", 0, stale, "waiting for reconciler push"
	}
	if mc.Version < dep.Version {
		return "pending", mc.Version, stale, "config update not yet pushed"
	}
	syncedAt := lastSyncMillis(inst)
	if syncedAt == 0 {
		return "pending", mc.Version, stale, "instance never synced"
	}
	if mc.Updated != nil && syncedAt < mc.Updated.UnixMilli() {
		return "pending", mc.Version, stale, "waiting for instance config sync"
	}
	return "running", mc.Version, stale, ""
}

func loadDeploymentConfig(instanceID, deploymentID string) (configsserver.ManagedConfig, bool) {
	ctx := orm.NewContext().DirectAccess()
	orm.WithModel(ctx, &configsserver.ManagedConfig{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("instance_id", instanceID)).
		Filter(orm.TermQuery("name", managedConfigName(deploymentID))))
	if err != nil || res == nil {
		return configsserver.ManagedConfig{}, false
	}
	rows, _, _ := elastic.DecodeHits[configsserver.ManagedConfig](res)
	if len(rows) == 0 {
		return configsserver.ManagedConfig{}, false
	}
	return rows[0], true
}

func lastSyncMillis(inst model.Instance) int64 {
	if v, ok := inst.Labels[configsserver.LabelLastSyncAt]; ok {
		if ms, err := strconv.ParseInt(v, 10, 64); err == nil {
			return ms
		}
	}
	return 0
}

func instanceOnline(inst model.Instance) bool {
	ms := lastSyncMillis(inst)
	return ms > 0 && time.Since(time.UnixMilli(ms)) < 90*time.Second
}
