package neo4j

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	rdbms "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const (
	modePropertyWatermark = "property_watermark"

	paramLimit          = "__coco_limit"
	paramSkip           = "__coco_skip"
	paramCursorProperty = "__coco_cursor_property"
	paramCursorTie      = "__coco_cursor_tie"
)

const (
	tieAlias = "coco_tie"
)

type scanner struct {
	name       string
	connector  *common.Connector
	datasource *common.DataSource
	queue      *queue.QueueConfig
	stateStore *connectors.SyncStateStore
}

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
	Incremental   IncrementalConfig      `config:"incremental"`
	FieldMapping  rdbms.FieldMapping     `config:"field_mapping"`
}

func (cfg *Config) mapping() (*rdbms.Mapping, bool) {
	if cfg.FieldMapping.Enabled && cfg.FieldMapping.Mapping != nil {
		return cfg.FieldMapping.Mapping, true
	}
	return nil, false
}

type IncrementalConfig struct {
	Enabled      bool   `config:"enabled"`
	Mode         string `config:"mode"`
	Property     string `config:"property"`
	PropertyType string `config:"property_type"`
	TieBreaker   string `config:"tie_breaker"`
	ResumeFrom   string `config:"resume_from"`
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
		cfg.PageSize = uint(rdbms.DefaultPageSize)
	}

	if cfg.Incremental.Mode == "" {
		cfg.Incremental.Mode = modePropertyWatermark
	}

	if cfg.Incremental.Enabled {
		if cfg.Incremental.Mode != modePropertyWatermark {
			return fmt.Errorf("unsupported incremental mode %q", cfg.Incremental.Mode)
		}
		if strings.TrimSpace(cfg.Incremental.Property) == "" {
			return errors.New("incremental.property is required when incremental sync is enabled")
		}
		if strings.TrimSpace(cfg.Incremental.TieBreaker) == "" {
			return errors.New("incremental.tie_breaker is required when incremental sync is enabled")
		}
		cfg.Incremental.PropertyType = normalizePropertyType(cfg.Incremental.PropertyType)
	}

	return nil
}

func (s *scanner) Scan(ctx context.Context) {
	if err := connectors.CheckContextDone(ctx); err != nil {
		_ = log.Warnf("[%s connector] context cancelled before scan for datasource [%s]: %v", s.name, s.datasource.Name, err)
		return
	}

	cfg := Config{}
	if err := connectors.ParseConnectorConfigure(s.connector, s.datasource, &cfg); err != nil {
		_ = log.Errorf("[%s connector] parsing connector configuration failed for datasource [%s]: %v", s.name, s.datasource.Name, err)
		return
	}

	if err := cfg.validate(); err != nil {
		_ = log.Errorf("[%s connector] invalid configuration for datasource [%s]: %v", s.name, s.datasource.Name, err)
		return
	}

	driver, err := s.newDriver(&cfg)
	if err != nil {
		_ = log.Errorf("[%s connector] failed to create driver for datasource [%s]: %v", s.name, s.datasource.Name, err)
		return
	}
	defer func() {
		if closeErr := driver.Close(ctx); closeErr != nil {
			_ = log.Errorf("[%s connector] error closing driver for datasource [%s]: %v", s.name, s.datasource.Name, closeErr)
		}
	}()

	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = log.Errorf("[%s connector] failed to verify connectivity for datasource [%s]: %v", s.name, s.datasource.Name, err)
		return
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
			_ = log.Errorf("[%s connector] error closing session for datasource [%s]: %v", s.name, s.datasource.Name, closeErr)
		}
	}()

	var cursor *cursorSnapshot
	factory := cursorFactory{propertyType: cfg.Incremental.PropertyType}

	if cfg.Incremental.Enabled {
		stored, err := s.loadCursor(ctx, &cfg)
		if err != nil {
			_ = log.Errorf("[%s connector] failed to load sync cursor for datasource [%s]: %v", s.name, s.datasource.Name, err)
			return
		}
		cursor = stored
		if cursor == nil && cfg.Incremental.ResumeFrom != "" {
			snapshot, err := factory.fromResume(cfg.Incremental.ResumeFrom)
			if err != nil {
				_ = log.Errorf("[%s connector] invalid resume_from value for datasource [%s]: %v", s.name, s.datasource.Name, err)
				return
			}
			cursor = snapshot
		}
	}

	offset := 0
	page := 0
	totalProcessed := 0

	for {
		if err := connectors.CheckContextDone(ctx); err != nil {
			log.Infof("[%s connector] context cancelled during scan for datasource [%s]: %v", s.name, s.datasource.Name, err)
			return
		}

		query, params, err := s.buildQuery(&cfg, cursor, offset)
		if err != nil {
			_ = log.Errorf("[%s connector] failed to build query for datasource [%s]: %v", s.name, s.datasource.Name, err)
			return
		}

		page++
		if global.Env().IsDebug {
			log.Debugf("[%s connector] executing cypher for datasource [%s], page=%d, query=%s, params=%s", s.name, s.datasource.Name, page, query, util.MustToJSON(paramsForLogging(params)))
		}

		result, err := session.Run(ctx, query, params)
		if err != nil {
			_ = log.Errorf("[%s connector] cypher execution failed for datasource [%s]: %v. query=%s params=%s", s.name, s.datasource.Name, err, query, util.MustToJSON(paramsForLogging(params)))
			return
		}

		processed, lastCursor, err := s.processResult(ctx, result, &cfg, factory)
		if err != nil {
			_ = log.Errorf("[%s connector] failed processing rows for datasource [%s]: %v", s.name, s.datasource.Name, err)
			return
		}

		totalProcessed += processed

		if cfg.Incremental.Enabled {
			if lastCursor == nil {
				_ = log.Warnf("[%s connector] incremental property %s missing in page for datasource [%s]; stopping incremental scan", s.name, cfg.Incremental.Property, s.datasource.Name)
				break
			}
			cmp := 1
			if cursor != nil {
				cmp = compareCursor(lastCursor, cursor, cfg.Incremental.PropertyType)
			}
			switch {
			case cursor == nil || cmp > 0:
				cursor = lastCursor
			case cmp == 0:
				_ = log.Warnf("[%s connector] incremental cursor did not advance for datasource [%s]; tie breaker expression may be invalid", s.name, s.datasource.Name)
				cursor = lastCursor
				if err := s.saveCursor(ctx, &cfg, cursor); err != nil {
					_ = log.Errorf("[%s connector] failed to persist cursor for datasource [%s]: %v", s.name, s.datasource.Name, err)
				}
				break
			default:
				cursor = lastCursor
			}
			if err := s.saveCursor(ctx, &cfg, cursor); err != nil {
				_ = log.Errorf("[%s connector] failed to persist cursor for datasource [%s]: %v", s.name, s.datasource.Name, err)
				return
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

	log.Infof("[%s connector] finished scanning datasource [%s], total=%v documents processed", s.name, s.datasource.Name, totalProcessed)
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

func (s *scanner) buildQuery(cfg *Config, cursor *cursorSnapshot, offset int) (string, map[string]interface{}, error) {
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

func buildIncrementalQuery(cfg *Config, cursor *cursorSnapshot, _ int) (string, map[string]interface{}, error) {
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

	if cursor != nil && cursor.property != nil {
		var propParam interface{}
		if cfg.Incremental.PropertyType == "datetime" {
			if cursor.stored != nil {
				propParam = cursor.stored.Property.Value
			} else if ts, ok := cursor.property.(time.Time); ok {
				propParam = ts.UTC().Format(time.RFC3339Nano)
			} else {
				propParam = fmt.Sprintf("%v", cursor.property)
			}
		} else {
			propParam = cursor.property
		}
		builder.WriteString(" WHERE coco_property > ")
		builder.WriteString(propExpr)
		params[paramCursorProperty] = propParam
		if cursor.tie != nil {
			builder.WriteString(" OR (coco_property = ")
			builder.WriteString(propExpr)
			builder.WriteString(" AND ")
			builder.WriteString(tieAlias)
			builder.WriteString(" > $")
			builder.WriteString(paramCursorTie)
			builder.WriteString(")")
			params[paramCursorTie] = cursor.tie
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

func (s *scanner) processResult(ctx context.Context, result neo4j.ResultWithContext, cfg *Config, factory cursorFactory) (int, *cursorSnapshot, error) {
	processed := 0
	var lastCursor *cursorSnapshot

	for result.Next(ctx) {
		if err := connectors.CheckContextDone(ctx); err != nil {
			return processed, lastCursor, err
		}

		record := result.Record()
		payload := recordToMap(record)

		if cfg.Incremental.Enabled {
			propertyValue, ok := payload[cfg.Incremental.Property]
			if !ok {
				_ = log.Warnf("[%s connector] incremental property '%s' missing in row for datasource [%s]", s.name, cfg.Incremental.Property, s.datasource.Name)
				continue
			}
			tieValue, ok := payload[tieAlias]
			if !ok {
				_ = log.Warnf("[%s connector] incremental tie value missing in row for datasource [%s]", s.name, s.datasource.Name)
				continue
			}

			snapshotCandidate, err := factory.fromValue(propertyValue, tieValue)
			if err != nil {
				_ = log.Errorf("[%s connector] failed to normalize cursor value for datasource [%s]: %v", s.name, s.datasource.Name, err)
				continue
			}
			if lastCursor == nil || compareCursor(snapshotCandidate, lastCursor, cfg.Incremental.PropertyType) > 0 {
				lastCursor = snapshotCandidate
			}
		}

		// Remove internal tie alias before mapping to document fields
		delete(payload, tieAlias)

		doc, err := s.transform(payload, cfg)
		if err != nil {
			_ = log.Errorf("[%s connector] transform failed for datasource [%s]: %v", s.name, s.datasource.Name, err)
			continue
		}

		data := util.MustToJSONBytes(doc)
		if global.Env().IsDebug {
			log.Debugf("[%s connector] transformed data: %s", s.name, data)
		}
		if err := queue.Push(s.queue, data); err != nil {
			_ = log.Errorf("[%s connector] failed to push document to queue for datasource [%s]: %v", s.name, s.datasource.Name, err)
			return processed, lastCursor, err
		}

		processed++
	}

	if err := result.Err(); err != nil {
		return processed, lastCursor, err
	}

	return processed, lastCursor, nil
}

func (s *scanner) transform(payload map[string]interface{}, cfg *Config) (*common.Document, error) {
	doc := &common.Document{
		Source: common.DataSourceReference{
			ID:   s.datasource.ID,
			Type: "connector",
			Name: s.datasource.Name,
		},
	}
	doc.System = s.datasource.System

	if mapping, ok := cfg.mapping(); ok {
		transformer := rdbms.Transformer{Payload: payload, Visited: make(map[string]bool)}
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

func (s *scanner) loadCursor(ctx context.Context, cfg *Config) (*cursorSnapshot, error) {
	state, err := s.stateStore.Load(ctx, s.connector.ID, s.datasource.ID)
	if err != nil {
		if err.Error() == "record not found" {
			log.Info("[%s connector] cursor not found for datasource [%s]: %v", s.name, s.datasource.Name, err)
			return nil, nil
		}
		return nil, err
	}
	if state == nil || state.Cursor == nil {
		return nil, nil
	}

	savedProperty := strings.TrimSpace(state.Property)
	currentProperty := strings.TrimSpace(cfg.Incremental.Property)
	if currentProperty == "" || savedProperty == "" {
		return nil, nil
	}
	if savedProperty != currentProperty {
		log.Infof("[%s connector] incremental property changed for datasource [%s]: stored=%s current=%s, resetting cursor", s.name, s.datasource.Name, savedProperty, currentProperty)
		return nil, nil
	}
	factory := cursorFactory{propertyType: cfg.Incremental.PropertyType}
	return factory.fromStored(state.Cursor)
}

func (s *scanner) saveCursor(ctx context.Context, cfg *Config, snapshot *cursorSnapshot) error {
	state := &connectors.SyncState{
		ConnectorID:  s.connector.ID,
		DatasourceID: s.datasource.ID,
		Mode:         modePropertyWatermark,
		Property:     cfg.Incremental.Property,
		Cursor:       snapshot.stored,
	}
	return s.stateStore.Save(ctx, state)
}
