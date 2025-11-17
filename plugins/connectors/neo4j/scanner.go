package neo4j

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"infini.sh/framework/core/pipeline"

	"infini.sh/coco/core"
	"infini.sh/coco/plugins/connectors"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
)

const (
	paramLimit          = "__coco_limit"
	paramSkip           = "__coco_skip"
	paramCursorProperty = "__coco_cursor_property"
	paramCursorTie      = "__coco_cursor_tie"
)

const (
	tieAlias = "coco_tie"
)

type scanner struct {
	config             *Config
	connector          *core.Connector
	datasource         *core.DataSource
	cursorSerializer   *cmn.CursorSerializer
	cursorStateManager *cmn.CursorStateManager
	collectFunc        func(doc core.Document) error
}

func (cfg *Config) validate() error {
	if cfg.ConnectionURI == "" {
		return errors.New("connection_uri is required")
	}
	if cfg.Cypher == "" {
		return errors.New("cypher is required")
	}

	if cfg.Parameters == nil {
		cfg.Parameters = map[string]interface{}{}
	}

	if cfg.PageSize == 0 {
		cfg.PageSize = uint(cmn.DefaultPageSize)
	}

	// Validate incremental configuration
	if err := cfg.Incremental.Validate(); err != nil {
		return err
	}

	return nil
}

func (s *scanner) Scan(ctx *pipeline.Context) error {
	if err := connectors.CheckContextDone(ctx); err != nil {
		_ = log.Warnf("[%s connector] context cancelled before scan for datasource [%s]: %v", ConnectorNeo4j, s.datasource.Name, err)
		return fmt.Errorf("context cancelled: %w", err)
	}

	cfg := Config{}
	if err := connectors.ParseConnectorConfigure(s.connector, s.datasource, &cfg); err != nil {
		_ = log.Errorf("[%s connector] parsing connector configuration failed for datasource [%s]: %v", ConnectorNeo4j, s.datasource.Name, err)
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	if err := cfg.validate(); err != nil {
		_ = log.Errorf("[%s connector] invalid configuration for datasource [%s]: %v", ConnectorNeo4j, s.datasource.Name, err)
		return fmt.Errorf("invalid configuration: %w", err)
	}

	driver, err := s.newDriver(&cfg)
	if err != nil {
		_ = log.Errorf("[%s connector] failed to create driver for datasource [%s]: %v", ConnectorNeo4j, s.datasource.Name, err)
		return fmt.Errorf("failed to create driver: %w", err)
	}
	defer func() {
		if closeErr := driver.Close(ctx); closeErr != nil {
			_ = log.Errorf("[%s connector] error closing driver for datasource [%s]: %v", ConnectorNeo4j, s.datasource.Name, closeErr)
		}
	}()

	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = log.Errorf("[%s connector] failed to verify connectivity for datasource [%s]: %v", ConnectorNeo4j, s.datasource.Name, err)
		return fmt.Errorf("failed to verify connectivity: %w", err)
	}

	sessionConfig := neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead}
	if cfg.Database != "" {
		sessionConfig.DatabaseName = cfg.Database
	}
	if cfg.Pagination {
		sessionConfig.FetchSize = int(cfg.PageSize)
	}

	session := driver.NewSession(ctx, sessionConfig)
	defer func() {
		if closeErr := session.Close(ctx); closeErr != nil {
			_ = log.Errorf("[%s connector] error closing session for datasource [%s]: %v", ConnectorNeo4j, s.datasource.Name, closeErr)
		}
	}()

	var cursor *cmn.CursorWatermark
	if cfg.Incremental.Enabled {
		cursor, err = s.cursorStateManager.LoadWithFallback(ctx, cfg.Incremental)
		if err != nil {
			_ = log.Errorf("[%s connector] failed to load cursor for datasource [%s]: %v", ConnectorNeo4j, s.datasource.Name, err)
			return fmt.Errorf("failed to load cursor: %w", err)
		}
	}

	if cursor != nil {
		log.Infof("[%s connector] resuming from cursor: property=%v, tie=%v", ConnectorNeo4j, cursor.Property, cursor.Tie)
	} else {
		log.Infof("[%s connector] no cursor found, starting full scan", ConnectorNeo4j)
	}

	offset := 0
	page := 0
	totalProcessed := 0

	for {
		if err := connectors.CheckContextDone(ctx); err != nil {
			log.Infof("[%s connector] context cancelled during scan for datasource [%s]: %v", ConnectorNeo4j, s.datasource.Name, err)
			return fmt.Errorf("context cancelled during scan: %w", err)
		}

		query, params, err := s.buildQuery(&cfg, cursor, offset)
		if err != nil {
			_ = log.Errorf("[%s connector] failed to build query for datasource [%s]: %v", ConnectorNeo4j, s.datasource.Name, err)
			return fmt.Errorf("failed to build query: %w", err)
		}

		page++
		if global.Env().IsDebug {
			log.Debugf("[%s connector] executing cypher for datasource [%s], page=%d, query=%s, params=%s", ConnectorNeo4j, s.datasource.Name, page, query, util.MustToJSON(paramsForLogging(params)))
		}

		result, err := session.Run(ctx, query, params)
		if err != nil {
			_ = log.Errorf("[%s connector] cypher execution failed for datasource [%s]: %v. query=%s params=%s", ConnectorNeo4j, s.datasource.Name, err, query, util.MustToJSON(paramsForLogging(params)))
			return fmt.Errorf("cypher execution failed: %w", err)
		}

		processed, lastCursor, err := s.processResult(ctx, result, &cfg, s.cursorSerializer)
		if err != nil {
			_ = log.Errorf("[%s connector] failed processing rows for datasource [%s]: %v", ConnectorNeo4j, s.datasource.Name, err)
			return fmt.Errorf("failed processing rows: %w", err)
		}

		totalProcessed += processed

		if cfg.Incremental.Enabled {
			if lastCursor == nil {
				_ = log.Warnf("[%s connector] incremental property %s missing in page for datasource [%s]; stopping incremental scan", ConnectorNeo4j, cfg.Incremental.Property, s.datasource.Name)
				break
			}
			if cursor != nil {
				diff := cmn.CompareCursors(lastCursor, cursor, cfg.Incremental.PropertyType)
				if diff == 0 {
					_ = log.Warnf("[%s connector] incremental cursor did not advance for datasource [%s]; tie breaker expression may be invalid", ConnectorNeo4j, s.datasource.Name)
				}
			}
			cursor = lastCursor
			if err := s.cursorStateManager.Save(ctx, cfg.Incremental.Property, cursor); err != nil {
				_ = log.Errorf("[%s connector] failed to persist cursor for datasource [%s]: %v", ConnectorNeo4j, s.datasource.Name, err)
				return fmt.Errorf("failed to persist cursor: %w", err)
			}

			if processed == 0 {
				break
			}
			if !cfg.Pagination || processed < int(cfg.PageSize) {
				break
			}
			continue
		}

		if processed == 0 {
			break
		}

		if !cfg.Pagination {
			break
		}
		offset += processed
	}

	log.Infof("[%s connector] finished scanning datasource [%s], total=%v documents processed", ConnectorNeo4j, s.datasource.Name, totalProcessed)
	return nil
}

func (s *scanner) newDriver(cfg *Config) (neo4j.DriverWithContext, error) {
	auth := neo4j.NoAuth()
	if cfg.AuthToken != "" {
		auth = neo4j.BearerAuth(cfg.AuthToken)
	} else if cfg.Username != "" || cfg.Password != "" {
		auth = neo4j.BasicAuth(cfg.Username, cfg.Password, "")
	}

	return neo4j.NewDriverWithContext(cfg.ConnectionURI, auth)
}

func (s *scanner) buildQuery(cfg *Config, cursor *cmn.CursorWatermark, offset int) (string, map[string]interface{}, error) {
	if cfg.Incremental.Enabled {
		return buildIncrementalQuery(cfg, cursor, offset)
	}

	base := strings.TrimSpace(cfg.Cypher)
	params := cloneParameters(cfg.Parameters)

	builder := strings.Builder{}

	if cfg.Pagination {
		builder.WriteString("CALL () { ")
		builder.WriteString(base)
		builder.WriteString(" } WITH * ")
		builder.WriteString("SKIP $")
		builder.WriteString(paramSkip)
		builder.WriteString(" LIMIT $")
		builder.WriteString(paramLimit)
		builder.WriteString(" RETURN *")
		params[paramLimit] = int(cfg.PageSize)
		params[paramSkip] = offset
		return builder.String(), params, nil
	}

	return base, params, nil
}

func buildIncrementalQuery(cfg *Config, cursor *cmn.CursorWatermark, _ int) (string, map[string]interface{}, error) {
	base := strings.TrimSpace(cfg.Cypher)
	params := cloneParameters(cfg.Parameters)
	tieExpr := strings.TrimSpace(cfg.Incremental.TieBreaker)
	if tieExpr == "" {
		return "", nil, errors.New("incremental.tie_breaker is required")
	}

	builder := strings.Builder{}

	builder.WriteString("CALL () { ")
	builder.WriteString(base)
	builder.WriteString(" } WITH *, ")
	builder.WriteString(cfg.Incremental.Property)
	builder.WriteString(" AS coco_property, ")
	builder.WriteString(tieExpr)
	builder.WriteString(" AS ")
	builder.WriteString(tieAlias)

	propExpr := "$" + paramCursorProperty
	if cfg.Incremental.PropertyType == "datetime" {
		propExpr = fmt.Sprintf("datetime($%s)", paramCursorProperty)
	}

	if cursor != nil && cursor.Property != nil {
		var propParam interface{}
		if cfg.Incremental.PropertyType == "datetime" {
			if cursor.Stored != nil {
				propParam = cursor.Stored.Property.Value
			} else if ts, ok := cursor.Property.(time.Time); ok {
				propParam = ts.UTC().Format(time.RFC3339Nano)
			} else {
				propParam = fmt.Sprintf("%v", cursor.Property)
			}
		} else {
			propParam = cursor.Property
		}
		builder.WriteString(" WHERE coco_property > ")
		builder.WriteString(propExpr)
		params[paramCursorProperty] = propParam
		if cursor.Tie != nil {
			builder.WriteString(" OR (coco_property = ")
			builder.WriteString(propExpr)
			builder.WriteString(" AND ")
			builder.WriteString(tieAlias)
			builder.WriteString(" > $")
			builder.WriteString(paramCursorTie)
			builder.WriteString(")")
			params[paramCursorTie] = cursor.Tie
		}
	}

	builder.WriteString(" ORDER BY coco_property ASC, ")
	builder.WriteString(tieAlias)
	builder.WriteString(" ASC")

	if cfg.Pagination {
		builder.WriteString(" LIMIT $")
		builder.WriteString(paramLimit)
		builder.WriteString(" ")
		params[paramLimit] = int(cfg.PageSize)
	}

	builder.WriteString(" RETURN *")
	return builder.String(), params, nil
}

func (s *scanner) processResult(ctx context.Context, result neo4j.ResultWithContext, cfg *Config, factory *cmn.CursorSerializer) (int, *cmn.CursorWatermark, error) {
	processed := 0
	var lastCursor *cmn.CursorWatermark

	for result.Next(ctx) {
		if err := connectors.CheckContextDone(ctx); err != nil {
			return processed, lastCursor, err
		}

		record := result.Record()
		payload := recordToMap(record)

		if cfg.Incremental.Enabled {
			propertyValue, ok := payload[cfg.Incremental.Property]
			if !ok {
				_ = log.Warnf("[%s connector] incremental property '%s' missing in row for datasource [%s]", ConnectorNeo4j, cfg.Incremental.Property, s.datasource.Name)
				continue
			}
			tieValue, ok := payload[tieAlias]
			if !ok {
				_ = log.Warnf("[%s connector] incremental tie value missing in row for datasource [%s]", ConnectorNeo4j, s.datasource.Name)
				continue
			}

			snapshotCandidate, err := factory.FromValue(propertyValue, tieValue)
			if err != nil {
				_ = log.Errorf("[%s connector] failed to normalize cursor value for datasource [%s]: %v", ConnectorNeo4j, s.datasource.Name, err)
				continue
			}
			if lastCursor == nil || cmn.CompareCursors(snapshotCandidate, lastCursor, cfg.Incremental.PropertyType) > 0 {
				lastCursor = snapshotCandidate
			}
		}

		// Remove internal tie alias before mapping to document fields
		delete(payload, tieAlias)

		doc, err := s.transform(payload, cfg)
		if err != nil {
			_ = log.Errorf("[%s connector] transform failed for datasource [%s]: %v", ConnectorNeo4j, s.datasource.Name, err)
			continue
		}

		if err := s.collectFunc(*doc); err != nil {
			_ = log.Errorf("[%s connector] failed to collect document for datasource [%s]: %v", ConnectorNeo4j, s.datasource.Name, err)
		}

		processed++
	}

	if err := result.Err(); err != nil {
		return processed, lastCursor, err
	}

	return processed, lastCursor, nil
}

func (s *scanner) transform(payload map[string]interface{}, cfg *Config) (*core.Document, error) {
	doc := &core.Document{
		Source: core.DataSourceReference{
			ID:   s.datasource.ID,
			Type: "connector",
			Name: s.datasource.Name,
		},
	}
	doc.System = s.datasource.System

	if mapping, ok := cfg.mapping(); ok {
		transformer := cmn.Transformer{Payload: payload, Visited: make(map[string]bool)}
		transformer.Transform(doc, mapping)
	}

	if cfg.FieldMapping.Enabled && doc.ID != "" {
		doc.ID = fmt.Sprintf("%s-%s", s.datasource.ID, doc.ID)
		if cfg.FieldMapping.IDHashable() {
			doc.ID = util.MD5digest(doc.ID)
		}
	}

	return doc, nil
}
