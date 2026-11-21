/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"net/http"
	"sort"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common/engineguard"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// Engine security surface (S3, dry-run v1): probe + tier projection.
// The projection writes the tier role/user into the engine but switches
// NO read path — enforcement (security_run_as on searches) is a separate,
// deliberate flip once drift observation is green (DESIGN §S3.8 step 3).

// engineSecurityStatusHandler reports the probe verdict and the currently
// projected coco_tier_* roles (drift surface).
func (h *APIHandler) engineSecurityStatusHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	caps := engineguard.Probe()
	tiers := listProjectedTiers(req)
	h.WriteOKJSON(w, util.MapStr{
		"capabilities": caps,
		"tiers":        tiers,
		"enforcement":  "app-layer (dry-run: projection only, no run-as)",
	})
}

func listProjectedTiers(req *http.Request) []util.MapStr {
	// v1: derive the tenant's distinct facets from the sharing model's
	// datasource set — one tier per distinct visible-set signature, admin
	// gets the full-set tier
	out := []util.MapStr{}
	seen := map[string]bool{}
	for _, f := range tenantTierFacets(req) {
		sig := engineguard.TierSignature(f)
		if seen[sig] {
			continue
		}
		seen[sig] = true
		out = append(out, util.MapStr{
			"signature":   sig,
			"user":        engineguard.TierUserName(sig),
			"datasources": len(f.DatasourceIDs),
			"excludes":    len(f.FieldExcludes),
			"masks":       len(f.MaskRules),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["signature"].(string) < out[j]["signature"].(string) })
	return out
}

// tenantTierFacets enumerates the coarse tier shapes for status display:
// admin (everything) and one tier per distinct shared-datasource set
// found in the share rules. The per-user exact resolution happens at
// enforcement time; the status surface only needs the shape census.
func tenantTierFacets(req *http.Request) []engineguard.TierFacets {
	facets := []engineguard.TierFacets{{
		DatasourceIDs: []string{"*"},
		FieldExcludes: documentSourceExcludesForGuard(),
	}}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.DataSource{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(200).Include("id"))
	if err != nil {
		return facets
	}
	sources, _, err := elastic.DecodeHits[core.DataSource](res)
	if err != nil {
		return facets
	}
	ids := make([]string, 0, len(sources))
	for _, s := range sources {
		ids = append(ids, s.ID)
	}
	if len(ids) > 0 {
		facets = append(facets, engineguard.TierFacets{
			DatasourceIDs: ids,
			FieldExcludes: documentSourceExcludesForGuard(),
		})
	}
	return facets
}

func documentSourceExcludesForGuard() []string {
	return []string{"payload", "document_chunk", "ai_insights.embedding"}
}

// engineSecuritySyncHandler applies the tier roles (and their users) for
// the facets the status surface enumerates. Dry-run safe: projection
// only, searches keep the app-layer filters (double-layer, stricter wins).
func (h *APIHandler) engineSecuritySyncHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	caps := engineguard.Probe()
	if !caps.SecurityAPI {
		h.WriteError(w, "engine security API unavailable: "+caps.Reason, http.StatusServiceUnavailable)
		return
	}

	indices := []string{orm.GetIndexName(&core.Document{})}
	applied, failed := 0, 0
	seen := map[string]bool{}
	var errs []string
	for _, f := range tenantTierFacets(req) {
		sig := engineguard.TierSignature(f)
		if seen[sig] {
			continue
		}
		seen[sig] = true
		name := engineguard.TierUserName(sig)
		role := engineguard.CompileRoleBody(f, indices)
		if err := engineguard.ApplyTierRole(name, role); err != nil {
			failed++
			errs = append(errs, name+": "+err.Error())
			continue
		}
		applied++
	}
	body := util.MapStr{"applied": applied, "failed": failed}
	if len(errs) > 0 {
		body["errors"] = errs
	}
	h.WriteOKJSON(w, body)
}
