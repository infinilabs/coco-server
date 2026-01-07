/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package dispatcher

import (
	"fmt"

	"infini.sh/coco/core"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/security"
)

func (processor *Dispatcher) syncDatasource(c *core.DataSource) error {

	//check datasource and connector's config
	//create pipeline based sub tasks

	//[Pipeline] [Connector] [Datasource] is Running

	ctx := orm.NewContext()
	ctx.DirectReadAccess()

	ctx.Set(orm.ReadPermissionCheckingScope, security.PermissionScopePlatform)

	connector := core.Connector{}
	connector.ID = c.Connector.ConnectorID
	exists, err := orm.GetV2(ctx, &connector)

	if !exists {
		return errors.Errorf("connector %s not found", connector.ID)
	}
	if err != nil {
		panic(errors.Errorf("invalid %s connector:%v", connector.ID, err))
	}

	if !connector.Processor.Enabled {
		return errors.Errorf("connector %s not enable pipeline", connector.ID)
	}

	if connector.Processor.Name == "" {
		return errors.Errorf("connector %s not have a valid processor name", connector.ID)
	}

	pipelineCfg := pipeline.PipelineConfigV2{}
	pipelineCfg.Name = fmt.Sprintf("dynamic-datasource-task-%v", c.ID)
	pipelineCfg.Singleton = true
	pipelineCfg.Processors = []map[string]interface{}{}

	connectorProcessor := map[string]interface{}{}
	connectorProcessor[connector.Processor.Name] = c.Connector.Config

	pipelineCfg.Processors = append(pipelineCfg.Processors, connectorProcessor)
	pipelineCfg.Transient = true
	pipelineCfg.AutoStart = true
	pipelineCfg.MaxRunningInMs = int64(1000 * processor.config.MaxRunningTimeoutInSeconds)

	if len(pipelineCfg.Processors) > 0 {
		ctx := pipeline.AcquireContext(pipelineCfg)
		ctx.Set(core.PipelineContextConnector, &connector)
		ctx.Set(core.PipelineContextDatasource, c)
		if processor.config.PipelinesInSync {
			return pipeline.RunPipelineSync(pipelineCfg, ctx)
		} else {
			return pipeline.RunPipelineAsync(pipelineCfg, ctx)
		}
	} else {
		return errors.Errorf("invalid pipeline config for datasource: %v,%v, processor not found", c.ID, c.Name)
	}

	return nil
}
