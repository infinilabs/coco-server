/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package confluence

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/queue"
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
	SyncInterval        = time.Minute * 5
)

func init() {
	module.RegisterUserPlugin(&Plugin{})
}

type Plugin struct {
	connectors.BasePlugin
	// mu protects the cancel function below.
	mu sync.Mutex
	// ctx is the root context for the plugin, created on Start and cancelled on Stop.
	ctx context.Context
	// cancel is the function to call to cancel a running scan.
	cancel context.CancelFunc
}

func (p *Plugin) Name() string {
	return ConnectorConfluence
}

func (p *Plugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	// Create a root context that lives for the duration of the plugin
	p.ctx, p.cancel = context.WithCancel(context.Background())
	return p.BasePlugin.Start(SyncInterval)
}

func (p *Plugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancel != nil {
		log.Infof("[confluence connector] received stop signal, cancelling current scan")
		p.cancel()
		p.ctx = nil
		p.cancel = nil
	}
	return nil
}

func (p *Plugin) Setup() {
	p.BasePlugin.Init("connector.confluence", "indexing confluence wiki", p)
}

func (p *Plugin) Scan(connector *common.Connector, datasource *common.DataSource) {
	// Get the parent context
	p.mu.Lock()
	parentCtx := p.ctx
	p.mu.Unlock()

	// Check if the plugin has been stopped before proceeding.
	if parentCtx == nil {
		_ = log.Warnf("[confluence connector] plugin is stopped, skipping scan for datasource [%s]", datasource.Name)
		return
	}

	cfg := Config{}
	err := connectors.ParseConnectorConfigure(connector, datasource, &cfg)
	if err != nil {
		_ = log.Errorf("[confluence connector] parsing connector configuration failed: %v", err)
		panic(err)
	}
	cfg.Endpoint = strings.TrimRight(cfg.Endpoint, "/")

	log.Debugf("[confluence connector] handling datasource: %v", cfg)

	if cfg.Endpoint == "" || cfg.Space == "" {
		_ = log.Errorf("[confluence connector] missing required configuration for datasource [%s]: endpoint or space", datasource.Name)
		return
	}

	handler, err := NewConfluenceHandler(cfg.Endpoint, cfg.Username, cfg.Token)
	if err != nil {
		_ = log.Errorf("[confluence connector] failed to init Confluence client for datasource [%s]: %v", datasource.Name, err)
		panic(err)
	}

	scanCtx, scanCancel := context.WithCancel(parentCtx)
	// Ensure this scan's resources are cleaned up when it finishes.
	defer scanCancel()

	var wg sync.WaitGroup

	// processChan defines the logic to process a channel of search results.
	processChan := func(resultChan <-chan *SearchContentResponse, contentType string) {
		defer wg.Done()

		for {
			select {
			case <-scanCtx.Done():
				log.Debugf("[confluence connector] context cancelled, stopping process for [%s]", contentType)
				return
			case res, ok := <-resultChan:
				if !ok {
					log.Debugf("[confluence connector] channel for %s closed", contentType)
					return
				}

				for _, content := range res.Results {
					// Check for global shutdown or scan cancellation
					select {
					case <-scanCtx.Done():
						log.Debugf("[confluence connector] context cancelled during item processing for [%s]", contentType)
						return
					default:
					}
					if global.ShuttingDown() {
						log.Infof("[confluence connector] system is shutting down, stopping scan for [%s]", contentType)
						return
					}

					doc, err := p.transformToDocument(&content, datasource, &cfg)
					if err != nil {
						_ = log.Errorf("[confluence connector] failed to transform content %s: %v", content.ID, err)
						continue
					}

					data := util.MustToJSONBytes(doc)
					if err := queue.Push(p.Queue, data); err != nil {
						_ = log.Errorf("[confluence connector] failed to push document to queue for datasource [%s]: %v", datasource.Name, err)
						continue // Continue processing other documents instead of panicking
					}
				}
			}
		}
	}

	// Fetch and process pages
	wg.Add(1)
	pagesChan := p.fetchContent(scanCtx, handler, cfg.Space, TypePage)
	go processChan(pagesChan, TypePage)

	// Fetch and process blogposts if enabled
	if cfg.EnableBlogposts {
		wg.Add(1)
		blogpostsChan := p.fetchContent(scanCtx, handler, cfg.Space, TypeBlogpost)
		go processChan(blogpostsChan, TypeBlogpost)
	}

	// Fetch and process attachments if enabled
	if cfg.EnableAttachments {
		wg.Add(1)
		attachmentsChan := p.fetchContent(scanCtx, handler, cfg.Space, TypeAttachment)
		go processChan(attachmentsChan, TypeAttachment)
	}

	wg.Wait()
	log.Infof("[confluence connector] finished scanning datasource [%s]", datasource.Name)
}

type requestWrapper struct {
	SearchContentRequest
	Next string
}

func (p *Plugin) fetchContent(ctx context.Context, handler *ConfluenceHandler, space string, typeName string) <-chan *SearchContentResponse {
	resultChan := make(chan *SearchContentResponse)
	go func() {
		defer close(resultChan)

		var fn func(requestWrapper) (*SearchContentResponse, error)
		fn = func(req requestWrapper) (*SearchContentResponse, error) {
			if req.Next != "" {
				return handler.SearchNextContent(ctx, req.Next)
			}
			return handler.SearchContent(ctx, req.SearchContentRequest)
		}

		req := SearchContentRequest{
			Limit:  PageSize,
			Expand: strings.Split(PageExpanded, ","),
			CQL:    fmt.Sprintf("type = '%s' AND space = '%s'", typeName, space),
			Start:  0,
		}
		wrapper := requestWrapper{SearchContentRequest: req}

		for {
			select {
			case <-ctx.Done():
				log.Info("[confluence connector] context cancelled, stopping content fetching for [%s]", typeName)
				return
			default:
			}

			// Check for shutdown signal before making a network request
			if global.ShuttingDown() {
				log.Infof("[confluence connector] system is shutting down, stopping content fetching for [%s]", typeName)
				return
			}

			res, err := fn(wrapper)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					log.Info("[confluence connector] context cancelled, stopping content fetching for [%s]", typeName)
				} else {
					_ = log.Errorf("[confluence connector] fetching content failed: %v", err)
				}
				return
			}

			select {
			case resultChan <- res:
			case <-ctx.Done():
				log.Info("[confluence connector] context cancelled, stopping content fetching for [%s]", typeName)
				return
			}

			wrapper.Next = res.Next()
			if wrapper.Next == "" {
				return
			}
		}
	}()

	return resultChan
}

// transformToDocument converts a Confluence Content object to a common.Document.
func (p *Plugin) transformToDocument(content *Content, datasource *common.DataSource, cfg *Config) (*common.Document, error) {
	doc := common.Document{
		Source: common.DataSourceReference{
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
