/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package github

import (
	"context"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	githubv3 "github.com/google/go-github/v74/github"
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

func (p *Plugin) processRepos(ctx context.Context, client *githubv3.Client, cfg *Config, datasource *common.DataSource) {
	user, _, err := client.Users.Get(ctx, cfg.Owner)
	if err != nil {
		_ = log.Errorf("[%s connector] failed to get user for [name=%s]: %v", ConnectorGitHub, cfg.Owner, err)
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

	err = ListRepos(ctx, client, user, func(repos []*githubv3.Repository) bool {
		for _, repo := range repos {
			if global.ShuttingDown() {
				return false
			}

			if len(allowedRepos) > 0 && !allowedRepos[strings.ToLower(repo.GetName())] {
				continue
			}

			log.Debugf("[%s connector] processing repo: %s", ConnectorGitHub, repo.GetFullName())

			// Determine what content types will be indexed for this repo
			var contentTypes []string
			if cfg.IndexIssues {
				contentTypes = append(contentTypes, TypeIssue)
			}
			if cfg.IndexPullRequests {
				contentTypes = append(contentTypes, TypePullRequest)
			}

			// Track folders for hierarchy
			folderTracker.TrackGitFolders(cfg.Owner, repo.GetName(), contentTypes)

			// Index repository
			repoDoc := p.transformRepoToDocument(repo, datasource)
			p.pushToQueue(repoDoc)

			// Index issues
			if cfg.IndexIssues {
				p.processIssues(ctx, client, cfg.Owner, repo, datasource)
			}

			// Index pull requests
			if cfg.IndexPullRequests {
				p.processPullRequests(ctx, client, cfg.Owner, repo, datasource)
			}

			processed++

			// if all the repos are processed; then break list repos operation
			if len(allowedRepos) > 0 && len(allowedRepos) == processed {
				return false
			}
		}
		return true
	})
	if err != nil {
		_ = log.Errorf("[%s connector] failed to list repos for owner %s: %v", ConnectorGitHub, cfg.Owner, err)
		return
	}

	// Create all folder documents for the hierarchy
	folderTracker.CreateGitFolderDocuments(datasource, func(doc common.Document) {
		p.pushToQueue(&doc)
	})
}

func (p *Plugin) processIssues(ctx context.Context, client *githubv3.Client, owner string, repo *githubv3.Repository, datasource *common.DataSource) {
	err := ListIssues(ctx, client, owner, repo.GetName(), func(issues []*githubv3.Issue) bool {
		for _, issue := range issues {
			if global.ShuttingDown() {
				return false
			}
			// PRs are returned as issues, so we skip them here.
			if issue.IsPullRequest() {
				continue
			}
			comments, _ := ListComments(ctx, client, owner, repo.GetName(), issue.GetNumber())
			issueDoc := p.transformIssueToDocument(issue, comments, repo, datasource)
			p.pushToQueue(issueDoc)
		}
		return true
	})
	if err != nil {
		_ = log.Errorf("[%s connector] failed to list issues for repo %s/%s: %v", ConnectorGitHub, owner, repo.GetName(), err)
		return
	}

}

func (p *Plugin) processPullRequests(ctx context.Context, client *githubv3.Client, owner string, repo *githubv3.Repository, datasource *common.DataSource) {
	err := ListPullRequests(ctx, client, owner, repo.GetName(), func(prs []*githubv3.PullRequest) bool {
		for _, pr := range prs {
			if global.ShuttingDown() {
				return false
			}
			comments, _ := ListComments(ctx, client, owner, repo.GetName(), pr.GetNumber())
			prDoc := p.transformPullRequestToDocument(pr, comments, repo, datasource)
			p.pushToQueue(prDoc)
		}
		return true
	})
	if err != nil {
		_ = log.Errorf("[%s connector] failed to list pull requests for repo %s/%s: %v", ConnectorGitHub, owner, repo.GetName(), err)
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

func (p *Plugin) transformRepoToDocument(repo *githubv3.Repository, datasource *common.DataSource) *common.Document {
	owner := repo.Owner.GetLogin()

	// Level 3A: Repository info document - belongs to owner category
	categories := connectors.BuildGitRepositoryCategories(owner, repo.GetName())
	idSuffix := fmt.Sprintf("repo-%s-%d", repo.GetFullName(), repo.GetID())

	doc := connectors.CreateDocumentWithHierarchy(TypeRepository, TypeRepository, repo.GetFullName(), repo.GetHTMLURL(), 0, categories, datasource, idSuffix)

	// Add GitHub-specific repository metadata
	doc.Summary = repo.GetDescription()
	doc.Tags = repo.Topics
	doc.Owner = &common.UserInfo{UserID: owner, UserName: owner, UserAvatar: repo.Owner.GetAvatarURL()}

	created := repo.GetCreatedAt().Time
	doc.Created = &created
	updated := repo.GetUpdatedAt().Time
	doc.Updated = &updated

	// Add repository-specific metadata
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}
	doc.Metadata["repository_id"] = repo.GetID()
	doc.Metadata["full_name"] = repo.GetFullName()
	doc.Metadata["private"] = repo.GetPrivate()
	doc.Metadata["fork"] = repo.GetFork()
	doc.Metadata["stargazers_count"] = repo.GetStargazersCount()
	doc.Metadata["watchers_count"] = repo.GetWatchersCount()
	doc.Metadata["forks_count"] = repo.GetForksCount()
	doc.Metadata["open_issues_count"] = repo.GetOpenIssuesCount()
	doc.Metadata["default_branch"] = repo.GetDefaultBranch()
	if repo.GetLanguage() != "" {
		doc.Metadata["language"] = repo.GetLanguage()
	}

	return &doc
}

// Contentable defines an interface for common fields between issues and pull requests.
type Contentable interface {
	GetBody() string
	GetTitle() string
	GetHTMLURL() string
	GetID() int64
	GetNumber() int
	GetUser() *githubv3.User
	GetLabels() []*githubv3.Label
	GetCreatedAt() githubv3.Timestamp
	GetUpdatedAt() githubv3.Timestamp
	GetState() string
	GetAuthorAssociation() string
}

type issueWrapper struct {
	*githubv3.Issue
}

func (i *issueWrapper) GetLabels() []*githubv3.Label {
	return i.Labels
}

type pullRequestWrapper struct {
	*githubv3.PullRequest
}

func (p *pullRequestWrapper) GetLabels() []*githubv3.Label {
	return p.Labels
}

// transformContentableToDocument is a generic function to transform issue-like objects into a document.
func (p *Plugin) transformContentableToDocument(item Contentable, itemType string, comments []*githubv3.IssueComment, repo *githubv3.Repository, datasource *common.DataSource) *common.Document {
	owner := repo.Owner.GetLogin()
	repoName := repo.GetName()

	// Use the item type directly as content type since constants match GitFolder keys
	contentType := itemType

	// Level 4: Content item - belongs to owner/repo/content_type category
	categories := connectors.BuildGitItemCategories(owner, repoName, contentType)
	idSuffix := fmt.Sprintf("%s-%d-%d", itemType, repo.GetID(), item.GetID())

	var content strings.Builder
	content.WriteString(item.GetBody())
	for _, c := range comments {
		content.WriteString("\n\n")
		content.WriteString(c.GetBody())
	}

	doc := connectors.CreateDocumentWithHierarchy(itemType, itemType, item.GetTitle(), item.GetHTMLURL(), len(content.String()), categories, datasource, idSuffix)

	// Add GitHub-specific content metadata
	doc.Content = content.String()

	var tags []string
	for _, label := range item.GetLabels() {
		tags = append(tags, label.GetName())
	}
	doc.Tags = tags

	doc.Owner = &common.UserInfo{
		UserID:     item.GetUser().GetLogin(),
		UserName:   item.GetUser().GetLogin(),
		UserAvatar: item.GetUser().GetAvatarURL(),
	}

	created := item.GetCreatedAt().Time
	doc.Created = &created
	updated := item.GetUpdatedAt().Time
	doc.Updated = &updated

	state := item.GetState()
	if pr, ok := item.(*pullRequestWrapper); ok && pr.GetMerged() {
		state = "merged"
	}

	// Add content-specific metadata
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}
	doc.Metadata["id"] = item.GetID()
	doc.Metadata["state"] = state
	doc.Metadata["number"] = item.GetNumber()
	doc.Metadata["author_association"] = item.GetAuthorAssociation()
	doc.Metadata["repository_id"] = repo.GetID()
	doc.Metadata["repository_full_name"] = repo.GetFullName()

	return &doc
}

func (p *Plugin) transformIssueToDocument(issue *githubv3.Issue, comments []*githubv3.IssueComment, repo *githubv3.Repository, datasource *common.DataSource) *common.Document {
	return p.transformContentableToDocument(&issueWrapper{issue}, TypeIssue, comments, repo, datasource)
}

func (p *Plugin) transformPullRequestToDocument(pr *githubv3.PullRequest, comments []*githubv3.IssueComment, repo *githubv3.Repository, datasource *common.DataSource) *common.Document {
	return p.transformContentableToDocument(&pullRequestWrapper{pr}, TypePullRequest, comments, repo, datasource)
}

func (p *Plugin) pushToQueue(doc *common.Document) {
	data := util.MustToJSONBytes(doc)
	if err := queue.Push(p.Queue, data); err != nil {
		_ = log.Errorf("[%s connector] failed to push document to queue: %v", ConnectorGitHub, err)
	}
}
