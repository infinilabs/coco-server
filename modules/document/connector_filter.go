/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"net/http"
	"strings"

	"infini.sh/coco/modules/common"
)

// The connector facet filters on a field the document index does not carry:
// the connector reference (github, yuque, …) lives on the datasource, and
// documents only reference the datasource (source.id). The connector
// reference shown in the UI is resolved at read time by RefineDocument, so a
// connector filter must translate into datasource terms at query time — this
// way it also matches documents indexed before the connector was ever
// surfaced, with no reindex.

const (
	connectorFilterField  = "source.connector_id"
	datasourceFilterField = "source.id"
	// a connector with no bound datasource must match nothing; an empty any()
	// parses as zero terms (a no-op), so the translated clause carries a
	// sentinel term no datasource can ever have
	noDatasourceSentinel = "__no_datasource__"
)

// indirection keeps the translation unit-testable without an engine behind
// the datasource lookup
var datasourceIDsByConnector = common.GetDatasourceIDsByConnector

// translateConnectorFilters rewrites `filter=source.connector_id:any(github)`
// request params into `filter=source.id:any(ds1,ds2)` in place. It must run
// before any query builder reads the request, so every search route —
// keyword/semantic legs, the hybrid RRF routes and the aggregation request —
// shares the behavior. Other clauses pass through untouched.
func translateConnectorFilters(req *http.Request) {
	q := req.URL.Query()
	filters := q["filter"]
	if len(filters) == 0 {
		return
	}

	changed := false
	for i, clause := range filters {
		next, ok := translateConnectorFilterClause(clause)
		if !ok {
			continue
		}
		filters[i] = next
		changed = true
	}

	if !changed {
		return
	}

	q["filter"] = filters
	req.URL.RawQuery = q.Encode()
}

// translateConnectorFilterClause translates one filter clause. Supported
// syntax mirrors the framework parser (query_args_parser.go): optional `-`/`!`
// negation and `field:any(v1,v2)` terms lists. Anything else — other fields,
// other operators — passes through unmarked.
func translateConnectorFilterClause(clause string) (string, bool) {
	body := strings.TrimSpace(clause)

	negate := false
	if strings.HasPrefix(body, "-") || strings.HasPrefix(body, "!") {
		negate = true
		body = strings.TrimSpace(body[1:])
	}

	if !strings.HasPrefix(body, connectorFilterField+":any(") || !strings.HasSuffix(body, ")") {
		return clause, false
	}

	valueStr := body[len(connectorFilterField)+len(":any(") : len(body)-1]

	datasourceIDs := []string{}
	for _, item := range strings.Split(valueStr, ",") {
		connectorID := strings.TrimSpace(item)
		if connectorID == "" {
			continue
		}
		datasourceIDs = append(datasourceIDs, datasourceIDsByConnector(connectorID)...)
	}

	if len(datasourceIDs) == 0 {
		datasourceIDs = []string{noDatasourceSentinel}
	}

	prefix := ""
	if negate {
		prefix = "!"
	}
	return prefix + datasourceFilterField + ":any(" + strings.Join(datasourceIDs, ",") + ")", true
}
