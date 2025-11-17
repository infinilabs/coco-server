package neo4j

import (
	"fmt"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/plugins/connectors"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
)

const ConnectorNeo4j = "neo4j"

type Config struct {
	ConnectionURI string                 `config:"connection_uri"`
	Username      string                 `config:"username"`
	Password      string                 `config:"password"`
	AuthToken     string                 `config:"auth_token"`
	Database      string                 `config:"database"`
	Cypher        string                 `config:"cypher"`
	Parameters    map[string]interface{} `config:"parameters"`
	Pagination    bool                   `config:"pagination"`
	PageSize      uint                   `config:"page_size"`
	Incremental   cmn.IncrementalConfig  `config:"incremental"`
	FieldMapping  cmn.FieldMapping       `config:"field_mapping"`
}

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

func (p *Plugin) Fetch(ctx *pipeline.Context, connector *core.Connector, datasource *core.DataSource) error {
	if err := connectors.CheckContextDone(ctx); err != nil {
		_ = log.Warnf("[%s connector] context cancelled before scan for datasource [%s]: %v", ConnectorNeo4j, datasource.Name, err)
		return fmt.Errorf("context cancelled: %w", err)
	}

	cfg := Config{}
	if err := connectors.ParseConnectorConfigure(connector, datasource, &cfg); err != nil {
		_ = log.Errorf("[%s connector] parsing connector configuration failed for datasource [%s]: %v", ConnectorNeo4j, datasource.Name, err)
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	if err := cfg.validate(); err != nil {
		_ = log.Errorf("[%s connector] invalid configuration for datasource [%s]: %v", ConnectorNeo4j, datasource.Name, err)
		return fmt.Errorf("invalid configuration: %w", err)
	}

	serializer := cmn.NewCursorSerializer(cfg.Incremental.PropertyType)
	stateStore := connectors.NewSyncStateStore()
	stateManager := &cmn.CursorStateManager{
		ConnectorID:  connector.ID,
		DatasourceID: datasource.ID,
		Serializer:   serializer,
		StateStore:   stateStore,
	}

	worker := &scanner{
		config:             &cfg,
		connector:          connector,
		datasource:         datasource,
		cursorSerializer:   serializer,
		cursorStateManager: stateManager,
		collectFunc: func(doc core.Document) error {
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

func (cfg *Config) mapping() (*cmn.Mapping, bool) {
	if cfg.FieldMapping.Enabled && cfg.FieldMapping.Mapping != nil {
		return cfg.FieldMapping.Mapping, true
	}
	return nil, false
}
