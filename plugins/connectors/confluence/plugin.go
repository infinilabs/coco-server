/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package confluence

import (
	"context"
	"errors"
	"fmt"
	"infini.sh/coco/core"
	"net/url"
	"strings"

	log "github.com/cihub/seelog"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
)

// Config defined Confluence wiki configuration
type Config struct {
	Endpoint          string `config:"endpoint"`
	Username          string `config:"username"`
	Token             string `config:"token"`
	Space             string `config:"space"`
	EnableBlogposts   bool   `config:"enable_blogposts"`
	EnableAttachments bool   `config:"enable_attachments"`
}

const (
	ConnectorConfluence = "confluence"
	PageSize            = 25
	PageExpanded        = "body.view,history,version,metadata,extensions,space" // ancestors
	TypePage            = "page"
	TypeBlogpost        = "blogpost"
	TypeAttachment      = "attachment"
)

func init() {
	pipeline.RegisterProcessorPlugin(ConnectorConfluence, New)
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
	return ConnectorConfluence
}

func (p *Plugin) Fetch(ctx *pipeline.Context, connector *core.Connector, datasource *core.DataSource) error {
	cfg := Config{}
	p.MustParseConfig(datasource, &cfg)
	cfg.Endpoint = strings.TrimRight(cfg.Endpoint, "/")

	log.Debugf("[%s connector] handling datasource: %v", ConnectorConfluence, cfg)

	if cfg.Endpoint == "" || cfg.Space == "" {
		return fmt.Errorf("missing required configuration for datasource [%s]: endpoint or space", datasource.Name)
	}

	handler, err := NewConfluenceHandler(cfg.Endpoint, cfg.Username, cfg.Token)
	if err != nil {
		return fmt.Errorf("failed to init Confluence client for datasource [%s]: %v", datasource.Name, err)
	}

	scanCtx, scanCancel := context.WithCancel(context.Background())
	defer scanCancel()

	// Fetch and process pages
	if err := p.processContent(ctx, scanCtx, handler, connector, datasource, &cfg, cfg.Space, TypePage); err != nil {
		return err
	}

	// Fetch and process blogposts if enabled
	if cfg.EnableBlogposts {
		if err := p.processContent(ctx, scanCtx, handler, connector, datasource, &cfg, cfg.Space, TypeBlogpost); err != nil {
			return err
		}
	}

	// Fetch and process attachments if enabled
	if cfg.EnableAttachments {
		if err := p.processContent(ctx, scanCtx, handler, connector, datasource, &cfg, cfg.Space, TypeAttachment); err != nil {
			return err
		}
	}

	log.Infof("[%s connector] finished fetching datasource [%s]", ConnectorConfluence, datasource.Name)
	return nil
}

func (p *Plugin) processContent(ctx *pipeline.Context, scanCtx context.Context, handler *ConfluenceHandler, connector *core.Connector, datasource *core.DataSource, cfg *Config, space string, typeName string) error {
	req := SearchContentRequest{
		Limit:  PageSize,
		Expand: strings.Split(PageExpanded, ","),
		CQL:    fmt.Sprintf("type = '%s' AND space = '%s'", typeName, space),
		Start:  0,
	}

	var nextURL string

	for {
		// Check for context cancellation
		select {
		case <-scanCtx.Done():
			log.Infof("[%s connector] context cancelled, stopping content fetching for [%s]", ConnectorConfluence, typeName)
			return nil
		default:
		}

		// Check for shutdown signal before making a network request
		if global.ShuttingDown() {
			log.Infof("[%s connector] system is shutting down, stopping content fetching for [%s]", ConnectorConfluence, typeName)
			return nil
		}

		var res *SearchContentResponse
		var err error

		if nextURL != "" {
			res, err = handler.SearchNextContent(scanCtx, nextURL)
		} else {
			res, err = handler.SearchContent(scanCtx, req)
		}

		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Infof("[%s connector] context cancelled, stopping content fetching for [%s]", ConnectorConfluence, typeName)
				return nil
			}
			return fmt.Errorf("fetching content failed: %v", err)
		}

		var docs []core.Document
		for _, content := range res.Results {
			// Check for global shutdown
			if global.ShuttingDown() {
				log.Infof("[%s connector] system is shutting down, stopping processing for [%s]", ConnectorConfluence, typeName)
				return nil
			}

			doc, err := p.transformToDocument(&content, datasource, cfg)
			if err != nil {
				_ = log.Errorf("[%s connector] failed to transform content %s: %v", ConnectorConfluence, content.ID, err)
				continue
			}

			docs = append(docs, *doc)
		}

		if len(docs) > 0 {
			p.BatchCollect(ctx, connector, datasource, docs)
		}

		nextURL = res.Next()
		if nextURL == "" {
			break
		}
	}

	return nil
}

// transformToDocument converts a Confluence Content object to a common.Document.
func (p *Plugin) transformToDocument(content *Content, datasource *core.DataSource, cfg *Config) (*core.Document, error) {
	doc := core.Document{
		Source: core.DataSourceReference{
			ID:   datasource.ID,
			Type: "connector",
			Name: datasource.Name,
		},
		Type: ConnectorConfluence,
		Icon: "default",
	}

	doc.ID = util.MD5digest(fmt.Sprintf("%s-%s", datasource.ID, content.ID))
	doc.Title = content.Title
	doc.System = datasource.System

	doc.Metadata = make(map[string]interface{})
	doc.Metadata["confluence_id"] = content.ID
	if content.Space != nil {
		doc.Category = content.Space.Name
		doc.Metadata["space_key"] = content.Space.Key
		doc.Metadata["space_name"] = content.Space.Name
	}

	baseURL, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL %s: %w", cfg.Endpoint, err)
	}

	// Type-specific fields
	switch content.Type {
	case TypePage, TypeBlogpost:
		convertFromWiki(content, &doc, baseURL)
	case TypeAttachment:
		convertFromAttachment(content, &doc, baseURL)
	default:
		return nil, fmt.Errorf("unsupported content type: %s", content.Type)
	}

	return &doc, nil
}
