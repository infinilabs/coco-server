/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mssql

import (
	"fmt"
	"regexp"
	"strings"
)

// SQLServerDialect provides SQL Server-specific SQL generation functions
type SQLServerDialect struct{}

// BuildIncrementalQuery wraps a base query to add incremental sync conditions
func (d *SQLServerDialect) BuildIncrementalQuery(baseQuery string, lastSyncField string) string {
	return fmt.Sprintf("SELECT * FROM (%s) AS coco_subquery WHERE [%s] > @p1", baseQuery, lastSyncField)
}

// BuildPaginationQuery adds pagination to a SQL Server query using OFFSET/FETCH
func (d *SQLServerDialect) BuildPaginationQuery(baseQuery string, pageSize uint, offset uint) string {
	// SQL Server 2012+ requires ORDER BY for OFFSET/FETCH
	if !d.hasOrderByClause(baseQuery) {
		// Add a default ORDER BY clause if none exists
		// Use (SELECT NULL) as a neutral ordering that works with any query
		baseQuery = fmt.Sprintf("%s ORDER BY (SELECT NULL)", baseQuery)
	}

	return fmt.Sprintf("%s OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", baseQuery, offset, pageSize)
}

// hasOrderByClause checks if the query already contains an ORDER BY clause
func (d *SQLServerDialect) hasOrderByClause(query string) bool {
	// Use regex to check for ORDER BY at the end of the query
	// This handles cases where ORDER BY might appear in subqueries
	orderByRegex := regexp.MustCompile(`(?i)\bORDER\s+BY\b[^)]*$`)
	return orderByRegex.MatchString(strings.TrimSpace(query))
}
