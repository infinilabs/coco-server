/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package oracle

import (
	"fmt"
	"regexp"
	"strings"

	log "github.com/cihub/seelog"
	_ "github.com/sijms/go-ora/v2" // Import the Oracle driver
	"infini.sh/coco/modules/common"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
)

const (
	ConnectorOracle = "oracle"
)

func init() {
	pipeline.RegisterProcessorPlugin(ConnectorOracle, New)
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
	return ConnectorOracle
}

func (p *Plugin) Fetch(ctx *pipeline.Context, connector *common.Connector, datasource *common.DataSource) error {
	log.Debugf("[%s connector] handling datasource: %v", ConnectorOracle, datasource.Name)

	scanner := &cmn.Scanner{
		Name:       ConnectorOracle,
		Connector:  connector,
		Datasource: datasource,
		DriverName: "oracle",
		// Use Collect pattern instead of direct queue.Push
		CollectFunc: func(doc common.Document) error {
			p.Collect(ctx, connector, datasource, doc)
			return nil
		},
		SqlWithLastModified: func(baseQuery string, lastSyncField string) string {
			// Use :1 as the placeholder for Oracle (go-ora driver uses numbered parameters)
			return fmt.Sprintf(`SELECT * FROM (%s) WHERE %s > :1`, baseQuery, lastSyncField)
		},
		SqlWithPagination: func(baseQuery string, pageSize uint, offset uint) string {
			// Oracle 12c+ requires ORDER BY for OFFSET FETCH.
			if !hasOrderByClause(strings.ToUpper(baseQuery)) {
				// This is a simple check; complex queries might need manual ORDER BY.
				_ = log.Warnf("[%s connector] pagination is enabled but no ORDER BY clause was found in the query for datasource [%s]. Stability is not guaranteed.", ConnectorOracle, datasource.Name)
			}
			return fmt.Sprintf(`%s OFFSET %d ROWS FETCH NEXT %d ROWS ONLY`, baseQuery, offset, pageSize)
		},
	}

	if err := scanner.Scan(ctx); err != nil {
		return fmt.Errorf("failed to scan datasource: %w", err)
	}

	log.Infof("[%s connector] finished fetching datasource [%s]", ConnectorOracle, datasource.Name)
	return nil
}

// hasOrderByClause checks if the query already contains an ORDER BY clause
func hasOrderByClause(query string) bool {
	// Look for ORDER BY anywhere in the query, not just at the end
	orderByRegex := regexp.MustCompile(`(?i)\bORDER\s+BY\b`)
	return orderByRegex.MatchString(strings.TrimSpace(query))
}
