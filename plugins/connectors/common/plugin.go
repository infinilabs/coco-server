/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"context"
	"database/sql"
	"fmt"
	"infini.sh/coco/core"
	"time"

	"infini.sh/framework/core/orm"

	log "github.com/cihub/seelog"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
)

const (
	DefaultPageSize = 500
)

type Scanner struct {
	Name       string
	DriverName string
	Connector  *core.Connector
	Datasource *core.DataSource

	// CollectFunc is called to collect each document (replaces direct queue.Push)
	// This allows using the ConnectorProcessorBase.Collect() method
	CollectFunc func(doc core.Document) error

	// SqlWithLastModified used to query incremental data
	SqlWithLastModified func(baseQuery string, lastSyncField string) string

	// SqlWithPagination used to append pagination condition
	SqlWithPagination func(baseQuery string, pageSize uint, offset uint) string
}

type FieldMapping struct {
	Enabled bool     `config:"enabled"`
	Mapping *Mapping `config:"mapping"`
}

type Mapping struct {
	ID            string     `config:"id"`
	Hashed        bool       `config:"hashed"`
	Title         string     `config:"title"`
	URL           string     `config:"url"`
	Summary       string     `config:"summary"`
	Content       string     `config:"content"`
	Icon          string     `config:"icon"`
	Category      string     `config:"category"`
	Subcategory   string     `config:"subcategory"`
	Created       string     `config:"created"`
	Updated       string     `config:"updated"`
	Cover         string     `config:"cover"`
	Type          string     `config:"type"`
	Lang          string     `config:"lang"`
	Thumbnail     string     `config:"thumbnail"`
	Tags          string     `config:"tags"`
	Size          string     `config:"size"`
	Owner         UserInfo   `config:"owner"`
	Metadata      []KVPair   `config:"metadata"`
	Payload       []KVPair   `config:"payload"`
	LastUpdatedBy EditorInfo `config:"last_updated_by"`
}

type KVPair struct {
	Name  string `config:"name"`
	Value string `config:"value"`
}

type UserInfo struct {
	Avatar   string `config:"avatar"`
	UserName string `config:"username"`
	UserID   string `config:"userid"`
}

type EditorInfo struct {
	UserInfo  UserInfo `config:"user"`
	Timestamp string   `config:"timestamp"`
}

// Config defines the full configuration for the PostgreSQL connector,
// aligning with issue #457 to be query-centric.
type Config struct {
	ConnectionURI     string       `config:"connection_uri"`
	SQL               string       `config:"sql"`
	FieldMapping      FieldMapping `config:"field_mapping"`
	Pagination        bool         `config:"pagination"`
	PageSize          uint         `config:"page_size"`
	LastModifiedField string       `config:"last_modified_field"`
}

func (s *Scanner) Scan(ctx context.Context) error {
	cfg := Config{}
	if err := connectors.ParseConnectorConfigure(s.Connector, s.Datasource, &cfg); err != nil {
		_ = log.Errorf("[%s connector] parsing connector configuration failed for datasource [%s]: %v", s.Name, s.Datasource.Name, err)
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	if cfg.ConnectionURI == "" {
		_ = log.Errorf("[%s connector] ConnectionURI is required for datasource [%s]", s.Name, s.Datasource.Name)
		return fmt.Errorf("connection_uri is required")
	}
	if cfg.SQL == "" {
		_ = log.Errorf("[%s connector] SQL query is required for datasource [%s]", s.Name, s.Datasource.Name)
		return fmt.Errorf("sql query is required")
	}

	db, err := sql.Open(s.DriverName, cfg.ConnectionURI)
	if err != nil {
		_ = log.Errorf("[%s connector] failed to open database connection for datasource [%s]: %v", s.Name, s.Datasource.Name, err)
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	pingCtx, pingCancel := context.WithTimeout(ctx, connectors.DefaultConnectionTimeout)
	defer pingCancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = log.Errorf("[%s connector] failed to connect to database for datasource [%s]: %v", s.Name, s.Datasource.Name, err)
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Debugf("[%s connector] successfully connected to %s for datasource [%s]", s.Name, s.DriverName, s.Datasource.Name)

	scanCtx, scanCancel := context.WithCancel(ctx)
	defer scanCancel()

	if err := s.processQuery(scanCtx, db, &cfg); err != nil {
		return fmt.Errorf("failed to process query: %w", err)
	}

	log.Debugf("[%s connector] finished scanning datasource [%s]", s.Name, s.Datasource.Name)
	return nil
}

// processQuery fetches and indexes rows from the user-defined query.
func (s *Scanner) processQuery(ctx context.Context, db *sql.DB, cfg *Config) error {
	baseQuery := cfg.SQL
	var args []interface{}
	// Handle incremental updates
	if cfg.LastModifiedField != "" {
		lastSyncValue, err := s.GetLastSyncValue(s.Datasource.ID)
		if err != nil {
			_ = log.Errorf("[%s connector] failed to get last sync value for datasource [%s]: %v", s.Name, s.Datasource.Name, err)
			return fmt.Errorf("failed to get last sync value: %w", err)
		}
		if lastSyncValue != nil {
			// Wrap the user's query to safely add a WHERE clause.
			baseQuery = s.SqlWithLastModified(baseQuery, cfg.LastModifiedField)
			args = append(args, lastSyncValue)
			log.Debugf("[%s connector] performing incremental sync for datasource [%s] where %s > %s", s.Name, s.Datasource.Name, cfg.LastModifiedField, lastSyncValue)
		} else {
			log.Debugf("[%s connector] performing full sync for datasource [%s] (no last sync value found)", s.Name, s.Datasource.Name)
		}
	}

	// Pagination parameter
	pageSize := cfg.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	var offset uint = 0

	for {
		query := baseQuery

		// Handle pagination
		if cfg.Pagination {
			query = s.SqlWithPagination(baseQuery, pageSize, offset)
		}

		rows, err := db.QueryContext(ctx, query, args...)

		if global.Env().IsDebug {
			log.Debugf("[%s connector] execute query [%s] for datasource [%s]", s.Name, query, s.Datasource.Name)
		}

		if err != nil {
			_ = log.Errorf("[%s connector] failed to execute query [%s] for datasource [%s]: %v", s.Name, query, s.Datasource.Name, err)
			return fmt.Errorf("failed to execute query: %w", err)
		}

		rowsProcessed, err := s.processRows(ctx, rows, cfg)
		if err != nil {
			_ = log.Errorf("[%s connector] error processing rows for datasource [%s]: %v", s.Name, s.Datasource.Name, err)
			_ = rows.Close()
			return fmt.Errorf("failed to process rows: %w", err)
		}
		_ = rows.Close()

		if !cfg.Pagination || rowsProcessed < int(pageSize) {
			break // Last page or not pagination
		}
		offset += pageSize
	}
	return nil
}

// processRows iterates through query results, transforms them, and pushes them to the queue.
func (s *Scanner) processRows(ctx context.Context, rows *sql.Rows, cfg *Config) (int, error) {
	columns, err := rows.Columns()
	if err != nil {
		return 0, fmt.Errorf("failed to get columns: %w", err)
	}

	var rowsProcessed int

	for rows.Next() {
		select {
		case <-ctx.Done():
			log.Infof("[%s connector] context cancelled, stopping row processing for datasource [%s]", s.Name, s.Datasource.Name)
			return rowsProcessed, ctx.Err()
		default:
		}

		if global.ShuttingDown() {
			log.Infof("[%s connector] system is shutting down, stopping row processing for datasource [%s]", s.Name, s.Datasource.Name)
			return rowsProcessed, fmt.Errorf("system shutting down")
		}

		rowMap, err := s.scanRowToMap(rows, columns)
		if err != nil {
			_ = log.Errorf("[%s connector] failed to scan row for datasource [%s]: %v", s.Name, s.Datasource.Name, err)
			continue
		}

		doc, err := s.transformRowToDocument(rowMap, cfg)
		if err != nil {
			_ = log.Errorf("[%s connector] transforming row to doc failed for datasource [%s]: %v", s.Name, s.Datasource.Name, err)
			continue
		}

		if err = s.CollectFunc(*doc); err != nil {
			_ = log.Errorf("[%s connector] failed to collect document for datasource [%s]: %v", s.Name, s.Datasource.Name, err)
		}
		rowsProcessed++
	}
	return rowsProcessed, nil
}

// scanRowToMap scans the current row into a map of column name to value.
func (s *Scanner) scanRowToMap(rows *sql.Rows, columns []string) (map[string]interface{}, error) {
	// create a slice to temporarily hold the column values.
	values := make([]interface{}, len(columns))

	// create a slice of pointers for rows.Scan() to write into.
	scanArgs := make([]interface{}, len(columns))

	// populate scanArgs with pointers to the values slice.
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// scan the current row from the database into our values slice.
	if err := rows.Scan(scanArgs...); err != nil {
		return nil, err
	}

	// create the final map to store the results.
	rowMap := make(map[string]interface{})
	for i, col := range columns {
		if values[i] != nil {
			rowMap[col] = values[i]
		}
	}
	return rowMap, nil
}

// transformRowToDocument converts a database row (as a map) to a common.Document using field mappings.
func (s *Scanner) transformRowToDocument(rowMap map[string]interface{}, cfg *Config) (*core.Document, error) {
	doc := &core.Document{
		Source: core.DataSourceReference{
			ID:   s.Datasource.ID,
			Type: "connector",
			Name: s.Datasource.Name,
		},
	}
	doc.System = s.Datasource.System

	if m, ok := cfg.GetValidMapping(); ok {
		t := Transformer{
			Payload: rowMap,
			Visited: make(map[string]bool),
		}
		t.Transform(doc, m)
	}

	// Enabled field mapping and doc's ID present
	if cfg.FieldMapping.Enabled && doc.ID != "" {
		// Append datasource ID to doc ID
		doc.ID = fmt.Sprintf("%s-%s", s.Datasource.ID, doc.ID)

		// ID hash
		if cfg.FieldMapping.IDHashable() {
			doc.ID = util.MD5digest(doc.ID)
		}
	}
	return doc, nil
}

func (s *Scanner) GetLastSyncValue(datasourceID string) (*time.Time, error) {
	var err error

	q := orm.Query{}
	q.Size = 1
	q.AddSort("updated", orm.DESC)
	q.Filter = &orm.Cond{
		BoolType:  orm.Filter,
		QueryType: orm.QueryTerms,
		Field:     "source.id",
		Value:     []interface{}{datasourceID},
	}

	var results []core.Document

	err, _ = orm.SearchWithJSONMapper(&results, &q)
	if err != nil {
		_ = log.Errorf("[%s connector] fetch last updated doc failed, datasource id: [%s]: %v", s.Name, datasourceID, err)
	}
	if len(results) > 0 {
		return results[0].Updated, nil
	}
	return nil, nil
}

func (p *KVPair) GetValue() string {
	if p.Value == "" {
		return p.Name
	}
	return p.Value
}

func (f *FieldMapping) IDHashable() bool {
	if !f.Enabled || f.Mapping == nil {
		return true
	}
	return f.Mapping.Hashed
}

func (c *Config) GetValidMapping() (*Mapping, bool) {
	if c.FieldMapping.Enabled {
		return c.FieldMapping.Mapping, c.FieldMapping.Mapping != nil
	}
	return nil, false
}
