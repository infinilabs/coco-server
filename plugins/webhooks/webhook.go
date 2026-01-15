/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package webhooks

import (
	"context"
	"net/http"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
)

func WebhookHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	datasourceID := ps.MustGetParameter("id")
	ctx := orm.NewContext()
	ctx.DirectReadAccess()

	ctx.PermissionScope(security.PermissionScopePlatform)

	datasource, err := common.GetDatasourceConfig(ctx, datasourceID)
	if err != nil {
		panic(err)
	}

	if datasource.WebhookConfig.Enabled {
		//append enrichment pipeline process
		if datasource.EnrichmentPipeline != nil {
			b, err := util.ReadBody(req)
			doc := core.Document{}
			doc.Metadata = map[string]interface{}{}
			doc.Payload = map[string]interface{}{}
			doc.Source.ID = datasource.ID
			doc.Source.Name = datasource.Name
			doc.Source.Type = "webhook"
			doc.SetOwnerID(datasource.GetOwnerID())

			ctx := pipeline.AcquireContext(*datasource.EnrichmentPipeline)
			log.Trace("running enrichment pipeline")

			ctx.Set("body_bytes", b)
			ctx.Set("document", doc)

			task.RunWithContext("refresh_cluster_health", func(ctx context.Context) error {
				pipeCtx := ctx.Value("ctx")
				ctx1, ok := pipeCtx.(*pipeline.Context)
				if ok {
					err = pipeline.RunPipelineSync(*datasource.EnrichmentPipeline, ctx1)
					if err != nil {
						panic(err)
					}

					docV := ctx1.Get("document")
					document, ok := docV.(core.Document)
					if ok {
						if document.ID != "" {
							queueCfg := &queue.QueueConfig{Name: "indexing_documents"}
							data := util.MustToJSONBytes(document)
							err := queue.Push(queue.SmartGetOrInitConfig(queueCfg), data)
							if err != nil {
								panic(err)
							}
						}
					}
				}
				return nil
			}, context.WithValue(req.Context(), "ctx", ctx))
		}
	} else {
		panic(errors.Errorf("invalid webhook config: %v", datasource.Name))
	}
	api.WriteAckOKJSON(w)
}

func init() {
	api.HandleUIMethod(api.POST, "/webhooks/:id", WebhookHandler)
}
