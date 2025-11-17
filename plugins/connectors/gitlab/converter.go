/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package gitlab

import (
	"context"
	"fmt"
	"infini.sh/coco/core"
	"net/url"
	"strings"

	log "github.com/cihub/seelog"
	gitlabv4 "gitlab.com/gitlab-org/api/client-go"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
)

const (
	TypeIssue        = "issue"
	TypeMergeRequest = "merge_request"
	TypeRepository   = "repository"
	TypeWiki         = "wiki"
	TypeSnippet      = "snippet"
)

type wikiWrapper struct {
	*gitlabv4.Wiki
	baseURL url.URL
}

// BuildWikiURL construct the wiki path segment using project's path_with_namespace and the wiki page slug.
func (w *wikiWrapper) BuildWikiURL(project *gitlabv4.Project) string {
	// Clear path to ensure we start clean
	w.baseURL.Path = ""
	wikiPath := fmt.Sprintf("/%s/-/wikis/%s", project.PathWithNamespace, w.Slug)

	// Combine the base URL with the wiki path.
	return w.baseURL.JoinPath(wikiPath).String()
}

func (p *Plugin) processProjects(ctx *pipeline.Context, client *gitlabv4.Client, cfg *Config, connector *core.Connector, datasource *core.DataSource) {
	scanCtx := context.Background()
	isGroup, err := isGroupOwner(scanCtx, client, cfg.Owner)
	if err != nil {
		_ = log.Errorf("[%s connector] failed to check whether the owner is a group [Owner=%s]: %v", ConnectorGitLab, cfg.Owner, err)
		return
	}

	var listProjects ListProjects
	if isGroup {
		listProjects = ListGroupProjects
	} else {
		listProjects = ListUserProjects
	}

	allowedRepos := make(map[string]bool)
	if len(cfg.Repos) > 0 {
		for _, r := range cfg.Repos {
			allowedRepos[strings.ToLower(r)] = true
		}
	}

	// Initialize folder tracker for hierarchical structure
	folderTracker := connectors.NewGitFolderTracker()
	// Create all folder documents for the hierarchy
	var folderDocs []core.Document
	folderTracker.CreateGitFolderDocuments(datasource, func(doc core.Document) {
		folderDocs = append(folderDocs, doc)
	})
	if len(folderDocs) > 0 {
		p.BatchCollect(ctx, connector, datasource, folderDocs)
	}

	var processed int

	err = listProjects(scanCtx, client, cfg.Owner, func(projects []*gitlabv4.Project) bool {
		for _, project := range projects {
			if global.ShuttingDown() {
				return false
			}

			if len(allowedRepos) > 0 && !allowedRepos[strings.ToLower(project.Name)] {
				continue
			}

			log.Debugf("[%s connector] processing project: %s", ConnectorGitLab, project.NameWithNamespace)

			// Determine what content types will be indexed for this project
			var contentTypes []string
			if cfg.IndexIssues {
				contentTypes = append(contentTypes, TypeIssue)
			}
			if cfg.IndexMergeRequests {
				contentTypes = append(contentTypes, TypeMergeRequest)
			}
			if cfg.IndexWikis {
				contentTypes = append(contentTypes, TypeWiki)
			}
			if cfg.IndexSnippets {
				contentTypes = append(contentTypes, TypeSnippet)
			}

			// Track folders for hierarchy
			folderTracker.TrackGitFolders(project.Namespace.Name, project.Name, contentTypes)

			// Index repository
			repoDoc := p.transformProjectToDocument(project, datasource)
			p.Collect(ctx, connector, datasource, *repoDoc)

			// Index issues
			if cfg.IndexIssues {
				p.processIssues(ctx, scanCtx, client, project, connector, datasource)
			}

			// Index merge requests
			if cfg.IndexMergeRequests {
				p.processMergeRequests(ctx, scanCtx, client, project, connector, datasource)
			}

			// Index wiki pages
			if cfg.IndexWikis {
				p.processWikis(ctx, scanCtx, client, project, connector, datasource)
			}

			// Index snippets
			if cfg.IndexSnippets {
				p.processSnippets(ctx, scanCtx, client, project, connector, datasource)
			}

			processed++
			if len(allowedRepos) > 0 && len(allowedRepos) == processed {
				return false
			}
		}
		return true
	})
	if err != nil {
		_ = log.Errorf("[%s connector] failed to list projects for owner %s: %v", ConnectorGitLab, cfg.Owner, err)
		return
	}

}

func (p *Plugin) processIssues(ctx *pipeline.Context, scanCtx context.Context, client *gitlabv4.Client, project *gitlabv4.Project, connector *core.Connector, datasource *core.DataSource) {
	err := ListIssues(scanCtx, client, project.ID, func(issues []*gitlabv4.Issue) bool {
		var docs []core.Document
		for _, issue := range issues {
			if global.ShuttingDown() {
				return false
			}
			comments, err := ListComments(scanCtx, client, project.ID, issue.IID)
			if err != nil {
				switch resolveCode(err) {
				case ContextDone:
					_ = log.Warnf("[%s connector] context canceled, stopping list comments for issue [project=%v, issue=#%d]: %v", ConnectorGitLab, project.NameWithNamespace, issue.IID, err)
					return false
				case NotFound:
					log.Debugf("[%s connector] comments not found for issue [project=%v, issue=#%d]: %v", ConnectorGitLab, project.NameWithNamespace, issue.IID, err)
				default:
					_ = log.Warnf("[%s connector] failed to list comments for issue [project=%v, issue=#%d]: %v", ConnectorGitLab, project.NameWithNamespace, issue.IID, err)
				}
			}

			issueDoc := p.transformIssueToDocument(issue, comments, project, datasource)
			docs = append(docs, *issueDoc)
		}
		if len(docs) > 0 {
			p.BatchCollect(ctx, connector, datasource, docs)
		}
		return true
	})
	if err != nil {
		_ = log.Errorf("[%s connector] failed to list issues for project [%s]: %v", ConnectorGitLab, project.NameWithNamespace, err)
		return
	}
}

func (p *Plugin) processMergeRequests(ctx *pipeline.Context, scanCtx context.Context, client *gitlabv4.Client, project *gitlabv4.Project, connector *core.Connector, datasource *core.DataSource) {
	err := ListMergeRequests(scanCtx, client, project.ID, func(mrs []*gitlabv4.BasicMergeRequest) bool {
		var docs []core.Document
		for _, mr := range mrs {
			if global.ShuttingDown() {
				return false
			}
			comments, err := ListComments(scanCtx, client, project.ID, mr.IID)
			if err != nil {
				switch resolveCode(err) {
				case ContextDone:
					_ = log.Warnf("[%s connector] context canceled, stopping list comments for merge request [project=%v, merge_request=#%d]: %v", ConnectorGitLab, project.NameWithNamespace, mr.IID, err)
					return false
				case NotFound:
					log.Debugf("[%s connector] comments not found for merge request [project=%v, merge_request=#%d]: %v", ConnectorGitLab, project.NameWithNamespace, mr.IID, err)
				default:
					_ = log.Warnf("[%s connector] failed to list comments for merge request [project=%v, merge_request=#%d]: %v", ConnectorGitLab, project.NameWithNamespace, mr.IID, err)
				}
			}
			mrDoc := p.transformMergeRequestToDocument(mr, comments, project, datasource)
			docs = append(docs, *mrDoc)
		}
		if len(docs) > 0 {
			p.BatchCollect(ctx, connector, datasource, docs)
		}
		return true
	})
	if err != nil {
		_ = log.Errorf("[%s connector] failed to list merge requests for project %s: %v", ConnectorGitLab, project.NameWithNamespace, err)
		return
	}
}

func (p *Plugin) processWikis(ctx *pipeline.Context, scanCtx context.Context, client *gitlabv4.Client, project *gitlabv4.Project, connector *core.Connector, datasource *core.DataSource) {
	err := ListWikiPages(scanCtx, client, project.ID, func(wikis []*gitlabv4.Wiki) bool {
		var docs []core.Document
		for _, wiki := range wikis {
			if global.ShuttingDown() {
				return false
			}
			wikiDoc := p.transformWikiToDocument(&wikiWrapper{wiki, *client.BaseURL()}, project, datasource)
			docs = append(docs, *wikiDoc)
		}
		if len(docs) > 0 {
			p.BatchCollect(ctx, connector, datasource, docs)
		}
		return true
	})
	if err != nil {
		_ = log.Errorf("[%s connector] failed to list wiki pages for project [%s]: %v", ConnectorGitLab, project.NameWithNamespace, err)
		return
	}
}

func (p *Plugin) processSnippets(ctx *pipeline.Context, scanCtx context.Context, client *gitlabv4.Client, project *gitlabv4.Project, connector *core.Connector, datasource *core.DataSource) {
	err := ListProjectSnippets(scanCtx, client, project.ID, func(snippets []*gitlabv4.Snippet) bool {
		var docs []core.Document
		for _, sn := range snippets {
			if global.ShuttingDown() {
				return false
			}
			snippetDoc := p.transformSnippetToDocument(sn, project, datasource)
			docs = append(docs, *snippetDoc)
		}
		if len(docs) > 0 {
			p.BatchCollect(ctx, connector, datasource, docs)
		}
		return true
	})
	if err != nil {
		_ = log.Errorf("[%s connector] failed to list snippets for project [%s]: %v", ConnectorGitLab, project.NameWithNamespace, err)
		return
	}
}

func (p *Plugin) transformProjectToDocument(project *gitlabv4.Project, datasource *core.DataSource) *core.Document {
	owner := project.Namespace.Name

	// Repository info document - belongs to owner category
	categories := connectors.BuildGitRepositoryCategories(owner, project.Name)
	idSuffix := fmt.Sprintf("repo-%s-%d", project.NameWithNamespace, project.ID)

	doc := connectors.CreateDocumentWithHierarchy(TypeRepository, TypeRepository, project.NameWithNamespace, project.WebURL, 0, categories, datasource, idSuffix)

	// Add GitLab-specific repository metadata
	doc.Summary = project.Description
	doc.Tags = project.Topics
	doc.Cover = project.AvatarURL

	if project.Owner != nil {
		doc.Owner = &core.UserInfo{
			UserID:     project.Owner.Username,
			UserName:   project.Owner.Name,
			UserAvatar: project.Owner.AvatarURL,
		}
	}

	doc.Created = project.CreatedAt
	doc.Updated = project.UpdatedAt

	// Add repository-specific metadata
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}
	doc.Metadata["project_id"] = project.ID
	doc.Metadata["name_with_namespace"] = project.NameWithNamespace
	doc.Metadata["path_with_namespace"] = project.PathWithNamespace
	doc.Metadata["visibility"] = project.Visibility
	doc.Metadata["fork"] = project.ForkedFromProject != nil
	doc.Metadata["merge_method"] = project.MergeMethod
	doc.Metadata["star_count"] = project.StarCount
	doc.Metadata["forks_count"] = project.ForksCount
	if project.DefaultBranch != "" {
		doc.Metadata["default_branch"] = project.DefaultBranch
	}

	// Store additional payload data
	doc.Payload = map[string]interface{}{
		"ssh_url_to_repo":  project.SSHURLToRepo,
		"http_url_to_repo": project.HTTPURLToRepo,
		"readme_url":       project.ReadmeURL,
		"links":            project.Links,
	}

	return &doc
}

func (p *Plugin) transformIssueToDocument(issue *gitlabv4.Issue, comments []*gitlabv4.Note, project *gitlabv4.Project, datasource *core.DataSource) *core.Document {
	owner := project.Namespace.Name
	repoName := project.Name

	// Level 4: Content item - belongs to owner/repo/content_type category
	categories := connectors.BuildGitItemCategories(owner, repoName, TypeIssue)
	idSuffix := fmt.Sprintf("issue-%d-%d", project.ID, issue.ID)

	var content strings.Builder
	content.WriteString(issue.Description)
	for _, c := range comments {
		content.WriteString("\n\n")
		content.WriteString(c.Body)
	}

	doc := connectors.CreateDocumentWithHierarchy(TypeIssue, TypeIssue, issue.Title, issue.WebURL, len(content.String()), categories, datasource, idSuffix)

	// Add GitLab-specific issue metadata
	doc.Content = content.String()

	var tags []string
	for _, label := range issue.Labels {
		tags = append(tags, label)
	}
	doc.Tags = tags

	if issue.Author != nil {
		doc.Owner = &core.UserInfo{
			UserID:     issue.Author.Username,
			UserName:   issue.Author.Name,
			UserAvatar: issue.Author.AvatarURL,
		}
	}

	doc.Created = issue.CreatedAt
	doc.Updated = issue.UpdatedAt

	// Add content-specific metadata
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}
	doc.Metadata["project_id"] = project.ID
	doc.Metadata["id"] = issue.ID
	doc.Metadata["iid"] = issue.IID
	doc.Metadata["state"] = issue.State
	doc.Metadata["upvotes"] = issue.Upvotes
	doc.Metadata["downvotes"] = issue.Downvotes

	return &doc
}

func (p *Plugin) transformMergeRequestToDocument(mr *gitlabv4.BasicMergeRequest, comments []*gitlabv4.Note, project *gitlabv4.Project, datasource *core.DataSource) *core.Document {
	owner := project.Namespace.Name
	repoName := project.Name

	// Level 4: Content item - belongs to owner/repo/content_type category
	categories := connectors.BuildGitItemCategories(owner, repoName, TypeMergeRequest)
	idSuffix := fmt.Sprintf("merge_request-%d-%d", project.ID, mr.ID)

	var content strings.Builder
	content.WriteString(mr.Description)
	for _, c := range comments {
		content.WriteString("\n\n")
		content.WriteString(c.Body)
	}

	doc := connectors.CreateDocumentWithHierarchy(TypeMergeRequest, TypeMergeRequest, mr.Title, mr.WebURL, len(content.String()), categories, datasource, idSuffix)

	// Add GitLab-specific merge request metadata
	doc.Content = content.String()

	var tags []string
	for _, label := range mr.Labels {
		tags = append(tags, label)
	}
	doc.Tags = tags

	if mr.Author != nil {
		doc.Owner = &core.UserInfo{
			UserID:     mr.Author.Username,
			UserName:   mr.Author.Name,
			UserAvatar: mr.Author.AvatarURL,
		}
	}

	doc.Created = mr.CreatedAt
	doc.Updated = mr.UpdatedAt

	// Add content-specific metadata
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}
	doc.Metadata["project_id"] = project.ID
	doc.Metadata["id"] = mr.ID
	doc.Metadata["iid"] = mr.IID
	doc.Metadata["state"] = mr.State
	doc.Metadata["upvotes"] = mr.Upvotes
	doc.Metadata["downvotes"] = mr.Downvotes

	return &doc
}

func (p *Plugin) transformWikiToDocument(wiki *wikiWrapper, project *gitlabv4.Project, datasource *core.DataSource) *core.Document {
	owner := project.Namespace.Name
	repoName := project.Name

	// Level 4: Content item - belongs to owner/repo/content_type category
	categories := connectors.BuildGitItemCategories(owner, repoName, TypeWiki)
	idSuffix := fmt.Sprintf("wiki-%d-%s", project.ID, wiki.Slug)

	doc := connectors.CreateDocumentWithHierarchy(TypeWiki, TypeWiki, wiki.Title, wiki.BuildWikiURL(project), len(wiki.Content), categories, datasource, idSuffix)

	// Add GitLab-specific wiki metadata
	doc.Content = wiki.Content
	doc.Subcategory = wiki.Slug

	// GitLab wiki pages do not expose created/updated in list; fallback to project times?
	// doc.Created = project.CreatedAt
	// doc.Updated = project.LastActivityAt

	// Add content-specific metadata
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}
	doc.Metadata["project_id"] = project.ID
	doc.Metadata["encoding"] = wiki.Encoding
	doc.Metadata["format"] = wiki.Format
	doc.Metadata["slug"] = wiki.Slug

	return &doc
}

func (p *Plugin) transformSnippetToDocument(sn *gitlabv4.Snippet, project *gitlabv4.Project, datasource *core.DataSource) *core.Document {
	owner := project.Namespace.Name
	repoName := project.Name

	// Level 4: Content item - belongs to owner/repo/content_type category
	categories := connectors.BuildGitItemCategories(owner, repoName, TypeSnippet)
	idSuffix := fmt.Sprintf("snippet-%d-%d", project.ID, sn.ID)

	doc := connectors.CreateDocumentWithHierarchy(TypeSnippet, TypeSnippet, sn.Title, sn.WebURL, len(sn.Description), categories, datasource, idSuffix)

	// Add GitLab-specific snippet metadata
	doc.Content = sn.Description

	doc.Owner = &core.UserInfo{
		UserID:   sn.Author.Username,
		UserName: sn.Author.Name,
	}

	doc.Created = sn.CreatedAt
	doc.Updated = sn.UpdatedAt

	// Add content-specific metadata
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}
	doc.Metadata["project_id"] = sn.ProjectID

	// Store additional payload data
	doc.Payload = map[string]interface{}{
		"file_name":          sn.FileName,
		"visibility":         sn.Visibility,
		"raw_url":            sn.RawURL,
		"files":              sn.Files,
		"repository_storage": sn.RepositoryStorage,
	}

	return &doc
}
