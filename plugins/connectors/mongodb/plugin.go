package mongodb

import (
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/plugins/connectors"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
)

const ConnectorName = "mongodb"

type Plugin struct {
	cmn.ConnectorProcessorBase
}

func init() {
	pipeline.RegisterProcessorPlugin(ConnectorName, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	runner := &Plugin{}
	runner.Init(c, runner)
	return runner, nil
}

func (p *Plugin) Name() string {
	return ConnectorName
}

// Fetch implements the main pipeline execution method for MongoDB connector
func (p *Plugin) Fetch(ctx *pipeline.Context, connector *core.Connector, datasource *core.DataSource) error {
	// Parse and validate configuration
	cfg := &Config{}
	err := connectors.ParseConnectorConfigure(connector, datasource, cfg)
	if err != nil {
		_ = log.Errorf("[%s] failed to parse configuration: %v", ConnectorName, err)
		return err
	}

	if err := cfg.Validate(); err != nil {
		_ = log.Errorf("[%s] invalid configuration: %v", ConnectorName, err)
		return err
	}

	log.Infof("[%s] starting MongoDB connector: %s", ConnectorName, cfg.String())

	stateManager := &cmn.CursorStateManager{
		ConnectorID:  connector.ID,
		DatasourceID: datasource.ID,
		Serializer:   cmn.NewCursorSerializer(cfg.GetPropertyType()),
		StateStore:   connectors.NewSyncStateStore(),
	}

	// Create and run scanner
	scanner := &scanner{
		config:     cfg,
		connector:  connector,
		datasource: datasource,
		collectFunc: func(doc core.Document) error {
			p.Collect(ctx, connector, datasource, doc)
			return nil
		},
		cursorStateManager: stateManager,
	}

	if err := scanner.Scan(ctx); err != nil {
		_ = log.Errorf("[%s] scan failed: %v", ConnectorName, err)
		return err
	}

	log.Infof("[%s] MongoDB connector completed successfully", ConnectorName)
	return nil
}
