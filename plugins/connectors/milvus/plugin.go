package milvus

import (
	"fmt"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/plugins/connectors"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
)

const ConnectorName = "milvus"

// Plugin implements the pipeline.Processor interface for the Milvus connector.
type Plugin struct {
	cmn.ConnectorProcessorBase
}

func init() {
	pipeline.RegisterProcessorPlugin(ConnectorName, New)
}

// New creates a new instance of the Milvus connector plugin.
func New(c *config.Config) (pipeline.Processor, error) {
	runner := &Plugin{}
	runner.Init(c, runner)
	return runner, nil
}

// Name returns the name of the connector.
func (p *Plugin) Name() string {
	return ConnectorName
}

// Fetch implements the main pipeline execution method for the Milvus connector.
func (p *Plugin) Fetch(ctx *pipeline.Context, connector *core.Connector, datasource *core.DataSource) error {
	if err := connectors.CheckContextDone(ctx); err != nil {
		_ = log.Warnf("[%s] context cancelled before scan for datasource [%s]: %v", ConnectorName, datasource.Name, err)
		return fmt.Errorf("context cancelled: %w", err)
	}

	// Parse and validate configuration
	cfg := &Config{}
	err := connectors.ParseConnectorConfigure(connector, datasource, cfg)
	if err != nil {
		_ = log.Errorf("[%s] failed to parse configuration for datasource [%s]: %v", ConnectorName, datasource.Name, err)
		return err
	}

	if err := cfg.Validate(); err != nil {
		_ = log.Errorf("[%s] invalid configuration for datasource [%s]: %v", ConnectorName, datasource.Name, err)
		return err
	}

	log.Infof("[%s] starting Milvus connector for datasource [%s]: %s", ConnectorName, datasource.Name, cfg.String())

	// Initialize cursor state manager for incremental sync
	stateManager := &cmn.CursorStateManager{
		ConnectorID:  connector.ID,
		DatasourceID: datasource.ID,
		Serializer:   cmn.NewCursorSerializer(cfg.Incremental.GetPropertyType()),
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
		_ = log.Errorf("[%s] scan failed for datasource [%s]: %v", ConnectorName, datasource.Name, err)
		return err
	}

	log.Infof("[%s] Milvus connector completed successfully for datasource [%s]", ConnectorName, datasource.Name)
	return nil
}
