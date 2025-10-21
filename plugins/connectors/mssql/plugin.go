/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mssql

import (
	"fmt"

	log "github.com/cihub/seelog"
	_ "github.com/microsoft/go-mssqldb" // Import the MSSQL driver
	"infini.sh/coco/modules/common"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
)

const (
	ConnectorMSSQL = "mssql"
)

func init() {
	pipeline.RegisterProcessorPlugin(ConnectorMSSQL, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	runner := Plugin{}
	runner.Init(c, &runner)
	return &runner, nil
}

type Plugin struct {
	cmn.ConnectorProcessorBase
}

func (p *Plugin) Name() string {
	return ConnectorMSSQL
}

func (p *Plugin) Fetch(ctx *pipeline.Context, connector *common.Connector, datasource *common.DataSource) error {
	log.Debugf("[%s connector] handling datasource: %v", ConnectorMSSQL, datasource.Name)

	dialect := &SQLServerDialect{}

	scanner := &cmn.Scanner{
		Name:       ConnectorMSSQL,
		Connector:  connector,
		Datasource: datasource,
		DriverName: "mssql",
		// Use Collect pattern instead of direct queue.Push
		CollectFunc: func(doc common.Document) error {
			p.Collect(ctx, connector, datasource, doc)
			return nil
		},
		SqlWithLastModified: func(baseQuery string, lastSyncField string) string {
			return dialect.BuildIncrementalQuery(baseQuery, lastSyncField)
		},
		SqlWithPagination: func(baseQuery string, pageSize uint, offset uint) string {
			return dialect.BuildPaginationQuery(baseQuery, pageSize, offset)
		},
	}

	if err := scanner.Scan(ctx); err != nil {
		return fmt.Errorf("failed to scan datasource: %w", err)
	}

	log.Infof("[%s connector] finished fetching datasource [%s]", ConnectorMSSQL, datasource.Name)
	return nil
}
