package neo4j

import (
	"fmt"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
)

const ConnectorNeo4j = "neo4j"

func init() {
	pipeline.RegisterProcessorPlugin(ConnectorNeo4j, New)
}

type Plugin struct {
	cmn.ConnectorProcessorBase
}

func New(c *config.Config) (pipeline.Processor, error) {
	runner := Plugin{}
	runner.Init(c, &runner)
	return &runner, nil
}

func (p *Plugin) Name() string {
	return ConnectorNeo4j
}

func (p *Plugin) Fetch(ctx *pipeline.Context, connector *common.Connector, datasource *common.DataSource) error {
	worker := &scanner{
		name:       ConnectorNeo4j,
		connector:  connector,
		datasource: datasource,
		stateStore: connectors.NewSyncStateStore(),
		// Use Collect pattern instead of direct queue.Push
		collectFunc: func(doc common.Document) error {
			p.Collect(ctx, connector, datasource, doc)
			return nil
		},
	}

	if err := worker.Scan(ctx); err != nil {
		return fmt.Errorf("failed to scan datasource: %w", err)
	}

	log.Infof("[%s connector] finished fetching datasource [%s]", ConnectorNeo4j, datasource.Name)
	return nil
}
