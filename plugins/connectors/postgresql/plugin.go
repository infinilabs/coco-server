/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package postgresql

import (
	"fmt"
	"infini.sh/coco/core"

	log "github.com/cihub/seelog"
	_ "github.com/lib/pq"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
)

const (
	ConnectorPostgreSQL = "postgresql"
)

func init() {
	pipeline.RegisterProcessorPlugin(ConnectorPostgreSQL, New)
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
	return ConnectorPostgreSQL
}

func (p *Plugin) Fetch(ctx *pipeline.Context, connector *core.Connector, datasource *core.DataSource) error {
	log.Debugf("[%s connector] handling datasource: %v", ConnectorPostgreSQL, datasource.Name)

	scanner := &cmn.Scanner{
		Name:       ConnectorPostgreSQL,
		Connector:  connector,
		Datasource: datasource,
		DriverName: "postgres",
		// Use Collect pattern instead of direct queue.Push
		CollectFunc: func(doc core.Document) error {
			p.Collect(ctx, connector, datasource, doc)
			return nil
		},
		SqlWithLastModified: func(baseQuery string, lastSyncField string) string {
			return fmt.Sprintf(`SELECT * FROM (%s) AS coco_subquery WHERE "%s" > $1`, baseQuery, lastSyncField)
		},
		SqlWithPagination: func(baseQuery string, pageSize uint, offset uint) string {
			return fmt.Sprintf(`%s LIMIT %d OFFSET %d`, baseQuery, pageSize, offset)
		},
	}

	if err := scanner.Scan(ctx); err != nil {
		return fmt.Errorf("failed to scan datasource: %w", err)
	}

	log.Infof("[%s connector] finished fetching datasource [%s]", ConnectorPostgreSQL, datasource.Name)
	return nil
}
