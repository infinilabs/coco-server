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
	doc := p.newDocument(datasource)
	doc.Title = repo.GetFullName()
	doc.Summary = repo.GetDescription()
	doc.URL = repo.GetHTMLURL()
	doc.Type = TypeRepository
	doc.Icon = TypeRepository
	doc.Tags = repo.Topics
	doc.Owner = &common.UserInfo{UserID: repo.Owner.GetLogin(), UserName: repo.Owner.GetLogin(), UserAvatar: repo.Owner.GetAvatarURL()}

	doc.ID = util.MD5digest(fmt.Sprintf("%s-%d", datasource.ID, repo.GetID()))
	created := repo.GetCreatedAt().Time
	doc.Created = &created
	updated := repo.GetUpdatedAt().Time
	doc.Updated = &updated

	return doc
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
	doc := p.newDocument(datasource)

	var content strings.Builder
	content.WriteString(item.GetBody())
	for _, c := range comments {
		content.WriteString("\n\n")
		content.WriteString(c.GetBody())
	}

	var tags []string
	for _, label := range item.GetLabels() {
		tags = append(tags, label.GetName())
	}

	doc.Title = item.GetTitle()
	doc.Content = content.String()
	doc.URL = item.GetHTMLURL()
	doc.Type = itemType
	doc.Icon = itemType
	doc.Category = repo.GetFullName()
	doc.Tags = tags
	doc.Owner = &common.UserInfo{
		UserID:     item.GetUser().GetLogin(),
		UserName:   item.GetUser().GetLogin(),
		UserAvatar: item.GetUser().GetAvatarURL(),
	}

	doc.ID = util.MD5digest(fmt.Sprintf("%s-%d-%d", datasource.ID, repo.GetID(), item.GetID()))
	created := item.GetCreatedAt().Time
	doc.Created = &created
	updated := item.GetUpdatedAt().Time
	doc.Updated = &updated

	state := item.GetState()
	if pr, ok := item.(*pullRequestWrapper); ok && pr.GetMerged() {
		state = "merged"
	}

	doc.Metadata = map[string]interface{}{
		"id":                 item.GetID(),
		"state":              state,
		"number":             item.GetNumber(),
		"author_association": item.GetAuthorAssociation(),
	}

	return doc
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
