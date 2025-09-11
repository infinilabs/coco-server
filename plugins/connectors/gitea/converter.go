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
	doc := p.newDocument(datasource)
	doc.Title = repo.FullName
	doc.Summary = repo.Description
	doc.URL = repo.HTMLURL
	doc.Type = TypeRepository
	doc.Icon = TypeRepository
	if repo.Owner != nil {
		doc.Owner = &common.UserInfo{
			UserID:     fmt.Sprintf("%d", repo.Owner.ID),
			UserName:   repo.Owner.UserName,
			UserAvatar: repo.Owner.AvatarURL,
		}
	}
	doc.ID = util.MD5digest(fmt.Sprintf("%s-%d", datasource.ID, repo.ID))
	doc.Created = &repo.Created
	doc.Updated = &repo.Updated
	return doc
}

// transformContentableToDocument is a generic function to transform issue-like objects into a document.
func (p *Plugin) transformContentableToDocument(item contentable, comments []*sdk.Comment, itemType string, repo *sdk.Repository, datasource *common.DataSource) *common.Document {
	doc := p.newDocument(datasource)
	var tags []string
	for _, label := range item.GetLabels() {
		tags = append(tags, label.Name)
	}

	var content strings.Builder
	content.WriteString(item.GetBody())
	for _, c := range comments {
		content.WriteString("\n\n")
		content.WriteString(c.Body)
	}

	doc.Title = item.GetTitle()
	doc.Content = content.String()
	doc.Size = len(doc.Content)
	doc.URL = item.GetHTMLURL()
	doc.Type = itemType
	doc.Icon = itemType
	doc.Category = repo.FullName
	doc.Tags = tags

	poster := item.GetPoster()
	if poster != nil {
		doc.Owner = &common.UserInfo{
			UserID:     fmt.Sprintf("%d", poster.ID),
			UserName:   poster.UserName,
			UserAvatar: poster.AvatarURL,
		}
	}

	doc.ID = util.MD5digest(fmt.Sprintf("%s-%d-%d", datasource.ID, repo.ID, item.GetID()))
	created := item.GetCreated()
	updated := item.GetUpdated()
	doc.Created = &created
	doc.Updated = &updated

	state := string(item.GetState())
	if pr, ok := item.(*prWrapper); ok && pr.HasMerged {
		state = "merged"
	}

	doc.Metadata = map[string]interface{}{
		"id":       item.GetID(),
		"number":   item.GetIndex(),
		"state":    state,
		"comments": item.GetComments(),
	}
	return doc
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
