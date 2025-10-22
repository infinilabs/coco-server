/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mssql

import (
	"context"

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

	scanCtx := context.Background()
	dialect := &SQLServerDialect{}

	scanner := &cmn.Scanner{
		Name:       ConnectorMSSQL,
		Connector:  connector,
		Datasource: datasource,
		Queue:      p.Queue,
		DriverName: "mssql",
		SqlWithLastModified: func(baseQuery string, lastSyncField string) string {
			return dialect.BuildIncrementalQuery(baseQuery, lastSyncField)
		},
		SqlWithPagination: func(baseQuery string, pageSize uint, offset uint) string {
			return dialect.BuildPaginationQuery(baseQuery, pageSize, offset)
		},
	}
	scanner.Scan(scanCtx)

	log.Infof("[%s connector] finished fetching datasource [%s]", ConnectorMSSQL, datasource.Name)
	return nil
}
