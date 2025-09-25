/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package gitlab

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	log "github.com/cihub/seelog"
	gitlabv4 "gitlab.com/gitlab-org/api/client-go"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
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

func (p *Plugin) processProjects(ctx context.Context, client *gitlabv4.Client, cfg *Config, datasource *common.DataSource) {
	isGroup, err := isGroupOwner(ctx, client, cfg.Owner)
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

	var processed int

	err = listProjects(ctx, client, cfg.Owner, func(projects []*gitlabv4.Project) bool {
		for _, project := range projects {
			if global.ShuttingDown() {
				return false
			}

			if len(allowedRepos) > 0 && !allowedRepos[strings.ToLower(project.Name)] {
				continue
			}

			log.Debugf("[%s connector] processing project: %s", ConnectorGitLab, project.NameWithNamespace)

			// Index repository
			repoDoc := p.transformProjectToDocument(project, datasource)
			p.pushToQueue(repoDoc)

			// Index issues
			if cfg.IndexIssues {
				p.processIssues(ctx, client, project, datasource)
			}

			// Index merge requests
			if cfg.IndexMergeRequests {
				p.processMergeRequests(ctx, client, project, datasource)
			}

			// Index wiki pages
			if cfg.IndexWikis {
				p.processWikis(ctx, client, project, datasource)
			}

			// Index snippets
			if cfg.IndexSnippets {
				p.processSnippets(ctx, client, project, datasource)
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

func (p *Plugin) processIssues(ctx context.Context, client *gitlabv4.Client, project *gitlabv4.Project, datasource *common.DataSource) {
	err := ListIssues(ctx, client, project.ID, func(issues []*gitlabv4.Issue) bool {
		for _, issue := range issues {
			if global.ShuttingDown() {
				return false
			}
			comments, err := ListComments(ctx, client, project.ID, issue.IID)
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
			p.pushToQueue(issueDoc)
		}
		return true
	})
	if err != nil {
		_ = log.Errorf("[%s connector] failed to list issues for project [%s]: %v", ConnectorGitLab, project.NameWithNamespace, err)
		return
	}
}

func (p *Plugin) processMergeRequests(ctx context.Context, client *gitlabv4.Client, project *gitlabv4.Project, datasource *common.DataSource) {
	err := ListMergeRequests(ctx, client, project.ID, func(mrs []*gitlabv4.BasicMergeRequest) bool {
		for _, mr := range mrs {
			if global.ShuttingDown() {
				return false
			}
			comments, err := ListComments(ctx, client, project.ID, mr.IID)
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
			p.pushToQueue(mrDoc)
		}
		return true
	})
	if err != nil {
		_ = log.Errorf("[%s connector] failed to list merge requests for project %s: %v", ConnectorGitLab, project.NameWithNamespace, err)
		return
	}
}

func (p *Plugin) processWikis(ctx context.Context, client *gitlabv4.Client, project *gitlabv4.Project, datasource *common.DataSource) {
	err := ListWikiPages(ctx, client, project.ID, func(wikis []*gitlabv4.Wiki) bool {
		for _, wiki := range wikis {
			if global.ShuttingDown() {
				return false
			}
			wikiDoc := p.transformWikiToDocument(&wikiWrapper{wiki, *client.BaseURL()}, project, datasource)
			p.pushToQueue(wikiDoc)
		}
		return true
	})
	if err != nil {
		_ = log.Errorf("[%s connector] failed to list wiki pages for project [%s]: %v", ConnectorGitLab, project.NameWithNamespace, err)
		return
	}
}

func (p *Plugin) processSnippets(ctx context.Context, client *gitlabv4.Client, project *gitlabv4.Project, datasource *common.DataSource) {
	err := ListProjectSnippets(ctx, client, project.ID, func(snippets []*gitlabv4.Snippet) bool {
		for _, sn := range snippets {
			if global.ShuttingDown() {
				return false
			}
			snippetDoc := p.transformSnippetToDocument(sn, project, datasource)
			p.pushToQueue(snippetDoc)
		}
		return true
	})
	if err != nil {
		_ = log.Errorf("[%s connector] failed to list snippets for project [%s]: %v", ConnectorGitLab, project.NameWithNamespace, err)
		return
	}
}

func (p *Plugin) newDocument(datasource *common.DataSource) *common.Document {
	doc := &common.Document{
		Source: common.DataSourceReference{
			ID:   datasource.ID,
			Type: "connector",
			Name: datasource.Name,
		},
	}
	doc.System = datasource.System
	return doc
}

func (p *Plugin) transformProjectToDocument(project *gitlabv4.Project, datasource *common.DataSource) *common.Document {
	doc := p.newDocument(datasource)
	doc.Title = project.NameWithNamespace
	doc.Summary = project.Description
	doc.URL = project.WebURL
	doc.Type = TypeRepository
	doc.Icon = TypeRepository
	doc.Tags = project.Topics
	doc.Cover = project.AvatarURL

	if project.Owner != nil {
		doc.Owner = &common.UserInfo{
			UserID:     project.Owner.Username,
			UserName:   project.Owner.Name,
			UserAvatar: project.Owner.AvatarURL,
		}
	}

	doc.ID = util.MD5digest(fmt.Sprintf("%s-%d", datasource.ID, project.ID))
	doc.Created = project.CreatedAt
	doc.Updated = project.UpdatedAt

	doc.Payload = map[string]interface{}{
		"project_id":          project.ID,
		"ssh_url_to_repo":     project.SSHURLToRepo,
		"http_url_to_repo":    project.HTTPURLToRepo,
		"readme_url":          project.ReadmeURL,
		"name":                project.Name,
		"name_with_namespace": project.NameWithNamespace,
		"path_with_namespace": project.PathWithNamespace,
		"merge_method":        project.MergeMethod,
		"links":               project.Links,
	}

	return doc
}

func (p *Plugin) transformIssueToDocument(issue *gitlabv4.Issue, comments []*gitlabv4.Note, project *gitlabv4.Project, datasource *common.DataSource) *common.Document {
	doc := p.newDocument(datasource)

	var content strings.Builder
	content.WriteString(issue.Description)
	for _, c := range comments {
		content.WriteString("\n\n")
		content.WriteString(c.Body)
	}

	doc.Title = issue.Title
	doc.Content = content.String()
	doc.Size = len(doc.Content)
	doc.URL = issue.WebURL
	doc.Type = TypeIssue
	doc.Icon = TypeIssue
	doc.Category = project.NameWithNamespace
	doc.Tags = issue.Labels

	if issue.Author != nil {
		doc.Owner = &common.UserInfo{
			UserID:     issue.Author.Username,
			UserName:   issue.Author.Name,
			UserAvatar: issue.Author.AvatarURL,
		}
	}

	doc.ID = util.MD5digest(fmt.Sprintf("%s-%d-%d", datasource.ID, project.ID, issue.ID))
	doc.Created = issue.CreatedAt
	doc.Updated = issue.UpdatedAt

	doc.Metadata = map[string]interface{}{
		"project_id": project.ID,
		"id":         issue.ID,
		"iid":        issue.IID,
		"state":      issue.State,
		"upvotes":    issue.Upvotes,
		"downvotes":  issue.Downvotes,
	}

	return doc
}

func (p *Plugin) transformMergeRequestToDocument(mr *gitlabv4.BasicMergeRequest, comments []*gitlabv4.Note, project *gitlabv4.Project, datasource *common.DataSource) *common.Document {
	doc := p.newDocument(datasource)

	var content strings.Builder
	content.WriteString(mr.Description)
	for _, c := range comments {
		content.WriteString("\n\n")
		content.WriteString(c.Body)
	}

	doc.Title = mr.Title
	doc.Content = content.String()
	doc.Size = len(doc.Content)
	doc.URL = mr.WebURL
	doc.Type = TypeMergeRequest
	doc.Icon = TypeMergeRequest
	doc.Category = project.NameWithNamespace
	doc.Tags = mr.Labels

	if mr.Author != nil {
		doc.Owner = &common.UserInfo{
			UserID:     mr.Author.Username,
			UserName:   mr.Author.Name,
			UserAvatar: mr.Author.AvatarURL,
		}
	}

	doc.ID = util.MD5digest(fmt.Sprintf("%s-%d-%d", datasource.ID, project.ID, mr.ID))
	doc.Created = mr.CreatedAt
	doc.Updated = mr.UpdatedAt

	doc.Metadata = map[string]interface{}{
		"project_id": project.ID,
		"id":         mr.ID,
		"iid":        mr.IID,
		"state":      mr.State,
		"upvotes":    mr.Upvotes,
		"downvotes":  mr.Downvotes,
	}

	return doc
}

func (p *Plugin) transformWikiToDocument(wiki *wikiWrapper, project *gitlabv4.Project, datasource *common.DataSource) *common.Document {
	doc := p.newDocument(datasource)
	doc.Title = wiki.Title
	doc.Content = wiki.Content

	doc.URL = wiki.BuildWikiURL(project)
	doc.Type = TypeWiki
	doc.Icon = TypeWiki
	doc.Category = project.NameWithNamespace
	doc.Subcategory = wiki.Slug
	doc.Size = len(doc.Content)

	// GitLab wiki pages do not expose created/updated in list; fallback to project times?
	// doc.Created = project.CreatedAt
	// doc.Updated = project.LastActivityAt

	doc.Metadata = map[string]interface{}{
		"project_id": project.ID,
		"encoding":   wiki.Encoding,
		"format":     wiki.Format,
		"slug":       wiki.Slug,
	}

	// ID: combine datasource, project, and wiki slug
	doc.ID = util.MD5digest(fmt.Sprintf("%s-%d-%s", datasource.ID, project.ID, wiki.Slug))
	return doc
}

func (p *Plugin) transformSnippetToDocument(sn *gitlabv4.Snippet, project *gitlabv4.Project, datasource *common.DataSource) *common.Document {
	doc := p.newDocument(datasource)
	doc.Title = sn.Title
	doc.Content = sn.Description
	doc.URL = sn.WebURL
	doc.Type = TypeSnippet
	doc.Icon = TypeSnippet
	doc.Category = project.NameWithNamespace
	doc.Owner = &common.UserInfo{
		UserID:   sn.Author.Username,
		UserName: sn.Author.Name,
	}
	doc.Created = sn.CreatedAt
	doc.Updated = sn.UpdatedAt
	doc.Metadata = map[string]interface{}{
		"project_id": sn.ProjectID,
	}
	doc.Payload = map[string]interface{}{
		"file_name":          sn.FileName,
		"visibility":         sn.Visibility,
		"raw_url":            sn.RawURL,
		"files":              sn.Files,
		"repository_storage": sn.RepositoryStorage,
	}
	doc.ID = util.MD5digest(fmt.Sprintf("%s-%d-%d", datasource.ID, project.ID, sn.ID))
	return doc
}

func (p *Plugin) pushToQueue(doc *common.Document) {
	data := util.MustToJSONBytes(doc)
	if err := queue.Push(p.Queue, data); err != nil {
		_ = log.Errorf("[%s connector] failed to push document to queue: %v", ConnectorGitLab, err)
	}
}
