/* Copyright © INFINI LTD. All rights reserved.  
 * Web: https://infinilabs.com  
 * Email: hello#infini.ltd */  
  
 package jira  
  
 import (  
	 "context"  
	 "fmt"  
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
   
 const (  
	 ConnectorJira    = "jira"  
	 DefaultPageSize  = 50  
	 MaxRetries      = 3  
	 SyncInterval    = time.Minute * 30  
 )  
   
 func init() {  
	 module.RegisterUserPlugin(&Plugin{})  
 }  
   
 type Plugin struct {  
	 connectors.BasePlugin  
	 mu     sync.Mutex  
	 ctx    context.Context  
	 cancel context.CancelFunc  
 }  
   
 func (p *Plugin) Name() string {  
	 return ConnectorJira  
 }  
   
 func (p *Plugin) Setup() {  
	 p.BasePlugin.Init("connector.jira", "indexing jira issues", p)  
 }  
   
 func (p *Plugin) Start() error {  
	 p.mu.Lock()  
	 defer p.mu.Unlock()  
	 p.ctx, p.cancel = context.WithCancel(context.Background())  
	 return p.BasePlugin.Start(SyncInterval)  
 }  
   
 func (p *Plugin) Stop() error {  
	 p.mu.Lock()  
	 defer p.mu.Unlock()  
   
	 if p.cancel != nil {  
		 log.Infof("[jira connector] received stop signal, cancelling current scan")  
		 p.cancel()  
		 p.ctx = nil  
		 p.cancel = nil  
	 }  
	 return nil  
 }  
   
 func (p *Plugin) Scan(connector *common.Connector, datasource *common.DataSource) {  
	 p.mu.Lock()  
	 parentCtx := p.ctx  
	 p.mu.Unlock()  
   
	 if parentCtx == nil {  
		 _ = log.Warnf("[jira connector] plugin is stopped, skipping scan for datasource [%s]", datasource.Name)  
		 return  
	 }  
   
	 cfg := Config{}  
	 err := connectors.ParseConnectorConfigure(connector, datasource, &cfg)  
	 if err != nil {  
		 _ = log.Errorf("[jira connector] parsing connector configuration failed: %v", err)  
		 return  
	 }  
   
	 log.Debugf("[jira connector] handling datasource: %v", cfg)  
   
	 if cfg.BaseURL == "" {  
		 _ = log.Errorf("[jira connector] missing required configuration for datasource [%s]: base_url", datasource.Name)  
		 return  
	 }  
   
	 client, err := NewJiraClient(&cfg)  
	 if err != nil {  
		 _ = log.Errorf("[jira connector] failed to init Jira client for datasource [%s]: %v", datasource.Name, err)  
		 return  
	 }  
   
	 scanCtx, scanCancel := context.WithCancel(parentCtx)  
	 defer scanCancel()  
   
	 p.scanIssues(scanCtx, client, connector, datasource, &cfg)  
 }  
   
 func (p *Plugin) scanIssues(ctx context.Context, client *JiraClient, connector *common.Connector, datasource *common.DataSource, cfg *Config) {  
	 jql := p.buildJQL(cfg, datasource)  
	 log.Debugf("[jira connector] using JQL: %s", jql)  
   
	 startAt := 0  
	 maxResults := cfg.MaxResults  
	 if maxResults <= 0 {  
		 maxResults = DefaultPageSize  
	 }  
   
	 for {  
		 select {  
		 case <-ctx.Done():  
			 log.Debugf("[jira connector] context cancelled, stopping scan")  
			 return  
		 default:  
		 }  
   
		 if global.ShuttingDown() {  
			 log.Infof("[jira connector] system is shutting down, stopping scan")  
			 return  
		 }  
   
		 searchResult, err := client.SearchIssues(ctx, jql, startAt, maxResults)  
		 if err != nil {  
			 _ = log.Errorf("[jira connector] failed to search issues: %v", err)  
			 return  
		 }  
   
		 if len(searchResult.Issues) == 0 {  
			 break  
		 }  
   
		 for _, issue := range searchResult.Issues {  
			 select {  
			 case <-ctx.Done():  
				 return  
			 default:  
			 }  
   
			 if global.ShuttingDown() {  
				 return  
			 }  
   
			 doc, err := p.transformIssueToDocument(&issue, datasource, cfg)  
			 if err != nil {  
				 _ = log.Errorf("[jira connector] failed to transform issue %s: %v", issue.Key, err)  
				 continue  
			 }  
   
			 // 获取评论（如果配置启用）  
			 if cfg.IncludeComments {  
				 comments, err := client.GetIssueComments(ctx, issue.Key)  
				 if err != nil {  
					 _ = log.Warnf("[jira connector] failed to get comments for issue %s: %v", issue.Key, err)  
				 } else {  
					 p.processComments(comments, datasource, cfg, connector)  
				 }  
			 }  
   
			 data := util.MustToJSONBytes(doc)  
			 if err := queue.Push(p.Queue, data); err != nil {  
				 _ = log.Errorf("[jira connector] failed to push document to queue: %v", err)  
				 continue  
			 }  
		 }  
   
		 log.Infof("[jira connector] processed %d issues (startAt: %d)", len(searchResult.Issues), startAt)  
   
		 if startAt+maxResults >= searchResult.Total {  
			 break  
		 }  
		 startAt += maxResults  
	 }  
 }  
   
 func (p *Plugin) buildJQL(cfg *Config, datasource *common.DataSource) string {  
	 jql := ""  
   
	 // 项目过滤  
	 if len(cfg.Projects) > 0 {  
		 projectList := ""  
		 for i, project := range cfg.Projects {  
			 if i > 0 {  
				 projectList += ", "  
			 }  
			 projectList += fmt.Sprintf("\"%s\"", project)  
		 }  
		 jql += fmt.Sprintf("project in (%s)", projectList)  
	 }  
   
	 // 问题类型过滤  
	 if len(cfg.IssueTypes) > 0 {  
		 if jql != "" {  
			 jql += " AND "  
		 }  
		 typeList := ""  
		 for i, issueType := range cfg.IssueTypes {  
			 if i > 0 {  
				 typeList += ", "  
			 }  
			 typeList += fmt.Sprintf("\"%s\"", issueType)  
		 }  
		 jql += fmt.Sprintf("issuetype in (%s)", typeList)  
	 }  
   
	 // 增量同步  
	 lastSync, err := connectors.GetLastSyncValue(datasource.ID)  
	 if err == nil && lastSync != "" {  
		 if jql != "" {  
			 jql += " AND "  
		 }  
		 jql += fmt.Sprintf("updated >= \"%s\"", lastSync)  
	 }  
   
	 // 自定义 JQL 过滤器  
	 if cfg.JQLFilter != "" {  
		 if jql != "" {  
			 jql += " AND "  
		 }  
		 jql += fmt.Sprintf("(%s)", cfg.JQLFilter)  
	 }  
   
	 // 默认排序  
	 if jql != "" {  
		 jql += " ORDER BY updated DESC"  
	 } else {  
		 jql = "ORDER BY updated DESC"  
	 }  
   
	 return jql  
 }  
   
 func (p *Plugin) transformIssueToDocument(issue *Issue, datasource *common.DataSource, cfg *Config) (*common.Document, error) {  
	 doc := &common.Document{  
		 Source: common.DataSourceReference{  
			 ID:   datasource.ID,  
			 Type: "connector",  
			 Name: datasource.Name,  
		 },  
		 Type:   ConnectorJira,  
		 Icon:   "jira",  
		 System: datasource.System,  
	 }  
   
	 doc.ID = util.MD5digest(fmt.Sprintf("%s-%s", datasource.ID, issue.Key))  
	 doc.Title = issue.Fields.Summary  
	 doc.Content = issue.Fields.Description  
	 doc.URL = fmt.Sprintf("%s/browse/%s", cfg.BaseURL, issue.Key)  
   
	 if issue.Fields.Project != nil {  
		 doc.Category = issue.Fields.Project.Name  
	 }  
   
	 if issue.Fields.Created != nil {  
		 doc.Created = issue.Fields.Created  
	 }  
	 if issue.Fields.Updated != nil {  
		 doc.Updated = issue.Fields.Updated  
	 }  
   
	 if issue.Fields.Reporter != nil {  
		 doc.Owner = &common.UserInfo{  
			 UserName: issue.Fields.Reporter.DisplayName,  
			 UserID:   issue.Fields.Reporter.AccountID,  
		 }  
	 }  
   
	 // 添加 Jira 特有的元数据  
	 doc.Metadata = map[string]interface{}{  
		 "jira_key":     issue.Key,  
		 "issue_type":   issue.Fields.IssueType.Name,  
		 "status":       issue.Fields.Status.Name,  
		 "priority":     issue.Fields.Priority.Name,  
		 "labels":       issue.Fields.Labels,  
		 "components":   issue.Fields.Components,  
	 }  
   
	 if issue.Fields.Assignee != nil {  
		 doc.Metadata["assignee"] = map[string]interface{}{  
			 "display_name": issue.Fields.Assignee.DisplayName,  
			 "account_id":   issue.Fields.Assignee.AccountID,  
		 }  
	 }  
   
	 return doc, nil  
 }  
   
 func (p *Plugin) processComments(comments *CommentsResponse, datasource *common.DataSource, cfg *Config, connector *common.Connector) {  
	 for _, comment := range comments.Comments {  
		 doc := &common.Document{  
			 Source: common.DataSourceReference{  
				 ID:   datasource.ID,  
				 Type: "connector",  
				 Name: datasource.Name,  
			 },  
			 Type:    "jira_comment",  
			 Icon:    "comment",  
			 System:  datasource.System,  
			 Title:   fmt.Sprintf("Comment on %s", comment.IssueKey),  
			 Content: comment.Body,  
			 URL:     fmt.Sprintf("%s/browse/%s?focusedCommentId=%s", cfg.BaseURL, comment.IssueKey, comment.ID),  
		 }  
   
		 doc.ID = util.MD5digest(fmt.Sprintf("%s-comment-%s", datasource.ID, comment.ID))  
   
		 if comment.Created != nil {  
			 doc.Created = comment.Created  
		 }  
		 if comment.Updated != nil {  
			 doc.Updated = comment.Updated  
		 }  
   
		 if comment.Author != nil {  
			 doc.Owner = &common.UserInfo{  
				 UserName: comment.Author.DisplayName,  
				 UserID:   comment.Author.AccountID,  
			 }  
		 }  
   
		 doc.Metadata = map[string]interface{}{  
			 "comment_id": comment.ID,  
			 "issue_key":  comment.IssueKey,  
		 }  
   
		 data := util.MustToJSONBytes(doc)  
		 if err := queue.Push(p.Queue, data); err != nil {  
			 _ = log.Errorf("[jira connector] failed to push comment to queue: %v", err)  
		 }  
	 }  
 }