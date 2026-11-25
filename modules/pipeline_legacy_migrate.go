/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package modules

import (
	"context"
	"encoding/json"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	elastic1 "infini.sh/framework/modules/elastic/common"
	"time"
)

// Before the explicit schema registration (MustRegisterSchemaWithIndexName)
// the ORM fell back to the Go type name for PipelineConfigV2, so any
// environment that created pipelines under an old build persisted them in
// `<prefix>pipelineconfigv2`. Environments that only ever used the two
// built-in pipelines (enrich_documents / enrich_attachments) are unaffected:
// setup templates wrote those into `<prefix>pipeline-configs` all along.
//
// After upgrading, the ORM reads `<prefix>pipeline-configs` only, so
// user-created pipelines would vanish from the UI even though the data is
// still in the legacy index. This one-time migration copies them over.
//
// It is idempotent and safe to run on every boot:
//   - no legacy index (the common fresh-install case) → no-op
//   - op_type=create + conflicts=proceed → documents already present in the
//     target (e.g. the two built-in configs) are never overwritten
//   - the legacy index is kept as a backup and simply ignored afterwards

const legacyPipelineIndexSuffix = "pipelineconfigv2"
const pipelineConfigIndexSuffix = "pipeline-configs"

// pipelineRawClient is the subset of the concrete ES client needed here;
// asserted off the API interface like engineRawClient in common.
type pipelineRawClient interface {
	Request(ctx context.Context, method, url string, body []byte) (*util.Result, error)
	GetEndpoint() string
}

// ormIndexPrefix returns the configured ES index prefix, defaulting to
// "coco_" (mirrors setupIndexPrefix in modules/system).
func ormIndexPrefix() string {
	cfg := elastic1.ORMConfig{}
	exist, err := env.ParseConfig("elastic.orm", &cfg)
	if exist && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}
	if cfg.IndexPrefix == "" {
		return "coco_"
	}
	return cfg.IndexPrefix
}

func migrateLegacyPipelineConfigs() {
	prefix := ormIndexPrefix()
	legacy := prefix + legacyPipelineIndexSuffix
	target := prefix + pipelineConfigIndexSuffix

	ctx := context.Background()

	// wait briefly for the cluster: a missing client or 5xx here usually
	// means ES is still initializing (coco starts right after ES in dev setups)
	var raw pipelineRawClient
	var status int
	for attempt := 0; attempt < 6; attempt++ {
		client := elastic.GetClientNoPanic(global.MustLookupString(elastic.GlobalSystemElasticsearchID))
		if client == nil {
			log.Warnf("[pipeline-migrate] elasticsearch client unavailable (attempt %d), retrying", attempt+1)
			time.Sleep(10 * time.Second)
			continue
		}
		r, ok := client.(pipelineRawClient)
		if !ok {
			log.Warnf("[pipeline-migrate] elasticsearch client does not support raw requests, skipping legacy pipeline config migration")
			return
		}
		res, err := r.Request(ctx, util.Verb_HEAD, r.GetEndpoint()+"/"+legacy, nil)
		if err != nil {
			log.Warnf("[pipeline-migrate] probing %v failed (attempt %d): %v, retrying", legacy, attempt+1, err)
			time.Sleep(10 * time.Second)
			continue
		}
		raw = r
		status = resStatus(res)
		break
	}
	if raw == nil {
		log.Warnf("[pipeline-migrate] elasticsearch unreachable after retries, skipping legacy pipeline config migration this boot")
		return
	}

	// 404 → no legacy data, nothing to migrate (fresh installs land here)
	if status == 404 {
		log.Debugf("[pipeline-migrate] no legacy index %v, nothing to migrate", legacy)
		return
	}
	if status != 200 {
		log.Warnf("[pipeline-migrate] probe of %v returned status %d, skipping migration this boot", legacy, status)
		return
	}

	res, err := raw.Request(ctx, util.Verb_GET, raw.GetEndpoint()+"/"+legacy+"/_count", nil)
	if err != nil || res == nil || res.StatusCode != 200 {
		log.Warnf("[pipeline-migrate] counting %v failed: %v, skipping migration this boot", legacy, err)
		return
	}
	var countRes struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(res.Body, &countRes); err != nil {
		log.Warnf("[pipeline-migrate] parsing count of %v failed: %v", legacy, err)
		return
	}
	if countRes.Count <= 0 {
		log.Debugf("[pipeline-migrate] legacy index %v is empty, nothing to migrate", legacy)
		return
	}

	// target must exist (setup templates create it); never create it here —
	// if it is missing the setup phase has not run yet, retry on next boot
	res, err = raw.Request(ctx, util.Verb_HEAD, raw.GetEndpoint()+"/"+target, nil)
	if err != nil || res == nil || res.StatusCode != 200 {
		log.Warnf("[pipeline-migrate] target index %v missing (status %d, err %v), skipping migration this boot", target, resStatus(res), err)
		return
	}

	// op_type=create + conflicts=proceed: built-in configs (enrich_documents /
	// enrich_attachments) already in the target stay untouched, only docs new
	// to the target are copied
	body, _ := json.Marshal(map[string]interface{}{
		"source":    map[string]interface{}{"index": legacy, "size": 500},
		"dest":      map[string]interface{}{"index": target, "op_type": "create"},
		"conflicts": "proceed",
	})
	res, err = raw.Request(ctx, util.Verb_POST, raw.GetEndpoint()+"/_reindex?refresh=true", body)
	if err != nil || res == nil || res.StatusCode != 200 {
		log.Errorf("[pipeline-migrate] reindex %v -> %v failed: status %d, err %v, body %s", legacy, target, resStatus(res), err, resBody(res))
		return
	}
	var reindexRes struct {
		Created          int `json:"created"`
		Updated          int `json:"updated"`
		VersionConflicts int `json:"version_conflicts"`
	}
	if err := json.Unmarshal(res.Body, &reindexRes); err != nil {
		log.Warnf("[pipeline-migrate] parsing reindex result failed: %v, body %s", err, resBody(res))
		return
	}
	if reindexRes.Created > 0 {
		log.Infof("[pipeline-migrate] migrated %d user pipeline configs from legacy index %v to %v (%d already present, kept as-is)",
			reindexRes.Created, legacy, target, reindexRes.VersionConflicts)
	} else {
		log.Infof("[pipeline-migrate] legacy index %v holds %d pipeline configs but all already exist in %v, nothing copied", legacy, countRes.Count, target)
	}
}

func resStatus(res *util.Result) int {
	if res == nil {
		return 0
	}
	return res.StatusCode
}

func resBody(res *util.Result) string {
	if res == nil {
		return ""
	}
	return string(res.Body)
}
