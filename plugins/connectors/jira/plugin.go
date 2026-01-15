package jira

import (
	"fmt"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
)

const (
	ConnectorJira = "jira"
)

// Plugin implements the Jira connector
type Plugin struct {
	cmn.ConnectorProcessorBase
}

func init() {
	// Register the Jira connector as a pipeline processor
	pipeline.RegisterProcessorPlugin(ConnectorJira, New)
}

// New creates a new instance of the Jira connector plugin
func New(c *config.Config) (pipeline.Processor, error) {
	runner := Plugin{}
	runner.Init(c, &runner)
	return &runner, nil
}

// Name returns the processor name
func (p *Plugin) Name() string {
	return ConnectorJira
}

// Fetch implements the main data fetching logic for Jira
func (p *Plugin) Fetch(ctx *pipeline.Context, connector *core.Connector, datasource *core.DataSource) error {
	scanCtx := ctx

	// Parse configuration
	cfg := &Config{}
	p.MustParseConfig(datasource, cfg)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		_ = log.Errorf("[jira] [%s] invalid config: %v", datasource.Name, err)
		return fmt.Errorf("invalid config: %w", err)
	}

	log.Infof("[jira] [%s] starting fetch - endpoint: %s, project: %s", datasource.Name, cfg.Endpoint, cfg.ProjectKey)

	// Create Jira client
	client, err := NewJiraClient(cfg.Endpoint, cfg.Username, cfg.Token, datasource.Name)
	if err != nil {
		_ = log.Errorf("[jira] [%s] failed to create client: %v", datasource.Name, err)
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Test connection and get project info
	project, err := client.GetProject(scanCtx, cfg.ProjectKey)
	if err != nil {
		_ = log.Errorf("[jira] [%s] failed to get project %s: %v", datasource.Name, cfg.ProjectKey, err)
		return fmt.Errorf("failed to get project: %w", err)
	}

	log.Infof("[jira] [%s] connected to project: %s (%s)", datasource.Name, project.Name, project.Key)

	// Build JQL query for the project
	jql := fmt.Sprintf("project = %s ORDER BY updated DESC", cfg.ProjectKey)
	log.Debugf("[jira] [%s] JQL query: %s", datasource.Name, jql)

	// Fetch issues with pagination
	startAt := 0
	pageSize := DefaultPageSize
	totalFetched := 0
	pageNum := 1

	for {
		// Check for shutdown or context cancellation
		if global.ShuttingDown() {
			log.Infof("[jira] [%s] shutting down, stopping fetch", datasource.Name)
			return nil
		}

		select {
		case <-ctx.Done():
			log.Infof("[jira] [%s] context cancelled, stopping fetch", datasource.Name)
			return ctx.Err()
		default:
		}

		// Fetch page of issues
		log.Debugf("[jira] [%s] fetching page %d (startAt: %d, maxResults: %d)", datasource.Name, pageNum, startAt, pageSize)
		issues, total, err := client.SearchIssues(scanCtx, jql, startAt, pageSize)
		if err != nil {
			_ = log.Errorf("[jira] [%s] failed to search issues (page %d): %v", datasource.Name, pageNum, err)
			return fmt.Errorf("failed to search issues: %w", err)
		}

		// Check if we got any issues
		if len(issues) == 0 {
			log.Infof("[jira] [%s] no more issues to fetch (total fetched: %d)", datasource.Name, totalFetched)
			break
		}

		log.Infof("[jira] [%s] processing page %d: %d issues (total: %d)", datasource.Name, pageNum, len(issues), total)

		// Process each issue
		for i, issue := range issues {
			// Check for shutdown
			if global.ShuttingDown() {
				log.Infof("[jira] [%s] shutting down, stopping issue processing", datasource.Name)
				return nil
			}

			// Transform issue to document
			doc, err := transformToDocument(&issue, datasource, cfg, cfg.IndexComments)
			if err != nil {
				_ = log.Warnf("[jira] [%s] failed to transform issue %s: %v", datasource.Name, issue.Key, err)
				continue
			}

			// Collect the document
			p.BatchCollect(ctx, connector, datasource, []core.Document{*doc})

			totalFetched++

			// Process attachments if enabled
			if cfg.IndexAttachments && issue.Fields.Attachments != nil {
				for _, attachment := range issue.Fields.Attachments {
					if global.ShuttingDown() {
						return nil
					}

					attachDoc, err := transformAttachmentToDocument(&issue, attachment, datasource, cfg)
					if err != nil {
						_ = log.Warnf("[jira] [%s] failed to transform attachment %s: %v", datasource.Name, attachment.Filename, err)
						continue
					}

					p.BatchCollect(ctx, connector, datasource, []core.Document{*attachDoc})
				}
			}

			// Log progress every 10 issues
			if (i+1)%10 == 0 {
				log.Debugf("[jira] [%s] processed %d/%d issues in page %d", datasource.Name, i+1, len(issues), pageNum)
			}
		}

		// Move to next page
		startAt += len(issues)
		pageNum++

		// Check if we've fetched all issues
		if startAt >= total {
			log.Infof("[jira] [%s] all issues fetched: %d total", datasource.Name, totalFetched)
			break
		}
	}

	log.Infof("[jira] [%s] fetch completed: %d issues indexed", datasource.Name, totalFetched)
	return nil
}
