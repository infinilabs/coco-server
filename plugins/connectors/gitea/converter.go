/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package gitea

import (
	"context"
	"fmt"
	"strings"

	sdk "code.gitea.io/sdk/gitea"
	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const (
	TypeIssue       = "issue"
	TypePullRequest = "pull_request"
	TypeRepository  = "repository"
)

func (p *Plugin) processRepos(ctx context.Context, client *sdk.Client, cfg *Config, datasource *common.DataSource) {
	isOrg, err := p.IsOrgUser(client, cfg.Owner)
	if err != nil {
		_ = log.Errorf("[%s connector] failed to get user for [name=%s]: %v", ConnectorGitea, cfg.Owner, err)
		return
	}

	allowedRepos := make(map[string]bool)
	if len(cfg.Repos) > 0 {
		for _, r := range cfg.Repos {
			allowedRepos[strings.ToLower(r)] = true
		}
	}

	// Initialize folder tracker for hierarchical structure
	folderTracker := connectors.NewGitFolderTracker()

	var processed int
	cursor := NewListReposCursor(cfg.Owner, isOrg)

	for {
		if global.ShuttingDown() {
			return
		}

		repos, err := ListRepos(ctx, client, cursor)
		if err != nil {
			_ = log.Errorf("[%s connector] failed to list repos for owner %s: %v", ConnectorGitea, cfg.Owner, err)
			break
		}

		for _, repo := range repos {
			if global.ShuttingDown() {
				break
			}

			if len(allowedRepos) > 0 && !allowedRepos[strings.ToLower(repo.Name)] {
				continue
			}

			log.Debugf("[%s connector] processing repo: %s", ConnectorGitea, repo.FullName)

			// Determine what content types will be indexed for this repo
			var contentTypes []string
			if cfg.IndexIssues {
				contentTypes = append(contentTypes, TypeIssue)
			}
			if cfg.IndexPullRequests {
				contentTypes = append(contentTypes, TypePullRequest)
			}

			// Track folders for hierarchy
			folderTracker.TrackGitFolders(cfg.Owner, repo.Name, contentTypes)

			// Index repository
			repoDoc := p.transformRepoToDocument(repo, datasource)
			p.pushToQueue(repoDoc)

			// Index issues
			if cfg.IndexIssues {
				p.processIssues(ctx, client, repo, datasource)
			}

			// Index pull requests
			if cfg.IndexPullRequests {
				p.processPullRequests(ctx, client, repo, datasource)
			}

			processed++
			if len(allowedRepos) > 0 && len(allowedRepos) == processed {
				break
			}
		}

		if !cursor.HasNext {
			break
		}
	}

	// Create all folder documents for the hierarchy
	folderTracker.CreateGitFolderDocuments(datasource, func(doc common.Document) {
		p.pushToQueue(&doc)
	})
}

func (p *Plugin) processIssues(ctx context.Context, client *sdk.Client, repo *sdk.Repository, datasource *common.DataSource) {
	cursor := NewListIssuesCursor(repo.Owner.UserName, repo.Name)
	for {
		issues, err := ListIssues(ctx, client, cursor)
		if err != nil {
			_ = log.Errorf("[%s connector] failed to list issues for repo %s: %v", ConnectorGitea, repo.FullName, err)
		}

		for _, issue := range issues {
			if global.ShuttingDown() {
				return
			}
			comments, err := ListComments(ctx, client, repo.Owner.UserName, repo.Name, issue.Index)
			if err != nil {
				switch connectors.ResolveCode(err) {
				case connectors.ContextDone:
					_ = log.Warnf("[%s connector] context canceled, stopping list comments for issue [repo=%s, issue=#%d]: %v", ConnectorGitea, repo.FullName, issue.Index, err)
					return
				default:
					_ = log.Warnf("[%s connector] failed to list comments for issue [repo=%s, issue=#%d]: %v", ConnectorGitea, repo.FullName, issue.Index, err)
				}
			}
			issueDoc := p.transformIssueToDocument(issue, comments, repo, datasource)
			p.pushToQueue(issueDoc)
		}

		if !cursor.HasNext {
			break
		}
	}
}

func (p *Plugin) processPullRequests(ctx context.Context, client *sdk.Client, repo *sdk.Repository, datasource *common.DataSource) {
	cursor := NewListPullRequestsCursor(repo.Owner.UserName, repo.Name)
	for {
		prs, err := ListPullRequests(ctx, client, cursor)
		if err != nil {
			_ = log.Errorf("[%s connector] failed to list pull requests for repo %s: %v", ConnectorGitea, repo.FullName, err)
		}
		for _, pr := range prs {
			if global.ShuttingDown() {
				return
			}
			comments, err := ListComments(ctx, client, repo.Owner.UserName, repo.Name, pr.Index)
			if err != nil {
				switch connectors.ResolveCode(err) {
				case connectors.ContextDone:
					_ = log.Warnf("[%s connector] context canceled, stopping list comments for pull requests [repo=%s, pr=#%d]: %v", ConnectorGitea, repo.FullName, pr.Index, err)
					return
				default:
					_ = log.Warnf("[%s connector] failed to list comments for pull requests [repo=%s, pr=#%d]: %v", ConnectorGitea, repo.FullName, pr.Index, err)
				}
			}
			prDoc := p.transformPullRequestToDocument(pr, comments, repo, datasource)
			p.pushToQueue(prDoc)
		}

		if !cursor.HasNext {
			break
		}
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

func (p *Plugin) transformRepoToDocument(repo *sdk.Repository, datasource *common.DataSource) *common.Document {
	owner := repo.Owner.UserName

	// Repository info document - belongs to owner category
	categories := connectors.BuildGitRepositoryCategories(owner, repo.Name)
	idSuffix := fmt.Sprintf("repo-%s-%d", repo.FullName, repo.ID)

	doc := connectors.CreateDocumentWithHierarchy(TypeRepository, TypeRepository, repo.FullName, repo.HTMLURL, 0, categories, datasource, idSuffix)

	// Add Gitea-specific repository metadata
	doc.Summary = repo.Description

	if repo.Owner != nil {
		doc.Owner = &common.UserInfo{
			UserID:     fmt.Sprintf("%d", repo.Owner.ID),
			UserName:   repo.Owner.UserName,
			UserAvatar: repo.Owner.AvatarURL,
		}
	}

	doc.Created = &repo.Created
	doc.Updated = &repo.Updated

	// Add repository-specific metadata
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}
	doc.Metadata["repository_id"] = repo.ID
	doc.Metadata["full_name"] = repo.FullName
	doc.Metadata["private"] = repo.Private
	doc.Metadata["fork"] = repo.Fork
	doc.Metadata["stars_count"] = repo.Stars
	doc.Metadata["watchers_count"] = repo.Watchers
	doc.Metadata["forks_count"] = repo.Forks
	doc.Metadata["open_issues_count"] = repo.OpenIssues
	doc.Metadata["default_branch"] = repo.DefaultBranch
	return &doc
}

// transformContentableToDocument is a generic function to transform issue-like objects into a document.
func (p *Plugin) transformContentableToDocument(item contentable, comments []*sdk.Comment, itemType string, repo *sdk.Repository, datasource *common.DataSource) *common.Document {
	owner := repo.Owner.UserName
	repoName := repo.Name

	// Use the item type directly as content type since constants match GitFolder keys
	contentType := itemType

	// Level 4: Content item - belongs to owner/repo/content_type category
	categories := connectors.BuildGitItemCategories(owner, repoName, contentType)
	idSuffix := fmt.Sprintf("%s-%d-%d", itemType, repo.ID, item.GetID())

	var content strings.Builder
	content.WriteString(item.GetBody())
	for _, c := range comments {
		content.WriteString("\n\n")
		content.WriteString(c.Body)
	}

	doc := connectors.CreateDocumentWithHierarchy(itemType, itemType, item.GetTitle(), item.GetHTMLURL(), len(content.String()), categories, datasource, idSuffix)

	// Add Gitea-specific content metadata
	doc.Content = content.String()

	var tags []string
	for _, label := range item.GetLabels() {
		tags = append(tags, label.Name)
	}
	doc.Tags = tags

	poster := item.GetPoster()
	if poster != nil {
		doc.Owner = &common.UserInfo{
			UserID:     fmt.Sprintf("%d", poster.ID),
			UserName:   poster.UserName,
			UserAvatar: poster.AvatarURL,
		}
	}

	created := item.GetCreated()
	updated := item.GetUpdated()
	doc.Created = &created
	doc.Updated = &updated

	state := string(item.GetState())
	if pr, ok := item.(*prWrapper); ok && pr.HasMerged {
		state = "merged"
	}

	// Add content-specific metadata
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}
	doc.Metadata["id"] = item.GetID()
	doc.Metadata["number"] = item.GetIndex()
	doc.Metadata["state"] = state
	doc.Metadata["comments"] = item.GetComments()
	doc.Metadata["repository_id"] = repo.ID
	doc.Metadata["repository_full_name"] = repo.FullName

	return &doc
}

func (p *Plugin) transformIssueToDocument(issue *sdk.Issue, comments []*sdk.Comment, repo *sdk.Repository, datasource *common.DataSource) *common.Document {
	return p.transformContentableToDocument(&issueWrapper{issue}, comments, TypeIssue, repo, datasource)
}

func (p *Plugin) transformPullRequestToDocument(pr *sdk.PullRequest, comments []*sdk.Comment, repo *sdk.Repository, datasource *common.DataSource) *common.Document {
	return p.transformContentableToDocument(&prWrapper{pr}, comments, TypePullRequest, repo, datasource)
}

func (p *Plugin) pushToQueue(doc *common.Document) {
	data := util.MustToJSONBytes(doc)
	if err := queue.Push(p.Queue, data); err != nil {
		_ = log.Errorf("[%s connector] failed to push document to queue: %v", ConnectorGitea, err)
	}
}
