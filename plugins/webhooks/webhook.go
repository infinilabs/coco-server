/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package webhooks

import (
	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"net/http"
)

func WebhookHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	datasourceID := ps.MustGetParameter("id")
	ctx := orm.NewContext()
	ctx.DirectReadAccess()
	datasource, err := common.GetDatasourceConfig(ctx, datasourceID)
	if err != nil {
		panic(err)
	}

	if datasource.WebhookConfig.Enabled {
		//append enrichment pipeline process
		if datasource.EnrichmentPipeline != nil {

			doc := common.Document{}
			doc.Metadata = map[string]interface{}{}
			doc.Payload = map[string]interface{}{}
			doc.Source.ID = datasource.ID
			doc.SetOwnerID(datasource.GetOwnerID())

			ctx := pipeline.AcquireContext(*datasource.EnrichmentPipeline)
			log.Error("running enrichment pipeline")

			b, err := util.ReadBody(req)

			ctx.Set("body_bytes", b)
			ctx.Set("document", doc)

			err = pipeline.RunPipelineSync(*datasource.EnrichmentPipeline, ctx)
			if err != nil {
				panic(err)
			}

			docV := ctx.Get("document")
			document, ok := docV.(common.Document)
			if ok {
				queueCfg := &queue.QueueConfig{Name: "indexing_documents"}
				data := util.MustToJSONBytes(document)
				err := queue.Push(queue.SmartGetOrInitConfig(queueCfg), data)
				if err != nil {
					panic(err)
				}
			}

		}
	} else {
		panic(errors.Errorf("invalid webhook config: %v", datasource.Name))
	}
}

func init() {
	api.HandleUIMethod(api.POST, "/webhooks/:id", WebhookHandler)
}
