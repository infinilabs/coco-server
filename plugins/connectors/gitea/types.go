/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package gitea

import (
	"time"

	sdk "code.gitea.io/sdk/gitea"
)

// Config defines the configuration for the Gitea connector.
type Config struct {
	BaseURL           string   `config:"base_url"`
	Token             string   `config:"token"`
	Owner             string   `config:"owner"`
	Repos             []string `config:"repos"`
	IndexIssues       bool     `config:"index_issues"`
	IndexPullRequests bool     `config:"index_pull_requests"`
	IndexWikis        bool     `config:"index_wikis"`
}

// contentable defines an interface for common fields between issues and pull requests.
type contentable interface {
	GetTitle() string
	GetBody() string
	GetHTMLURL() string
	GetLabels() []*sdk.Label
	GetPoster() *sdk.User
	GetID() int64
	GetIndex() int64
	GetState() sdk.StateType
	GetCreated() time.Time
	GetUpdated() time.Time
	GetComments() int
}

// issueWrapper wraps a gitea.Issue to implement contentable
type issueWrapper struct {
	*sdk.Issue
}

func (i *issueWrapper) GetTitle() string        { return i.Title }
func (i *issueWrapper) GetBody() string         { return i.Body }
func (i *issueWrapper) GetHTMLURL() string      { return i.HTMLURL }
func (i *issueWrapper) GetLabels() []*sdk.Label { return i.Labels }
func (i *issueWrapper) GetPoster() *sdk.User    { return i.Poster }
func (i *issueWrapper) GetID() int64            { return i.ID }
func (i *issueWrapper) GetIndex() int64         { return i.Index }
func (i *issueWrapper) GetState() sdk.StateType { return i.State }
func (i *issueWrapper) GetCreated() time.Time   { return i.Created }
func (i *issueWrapper) GetUpdated() time.Time   { return i.Updated }
func (i *issueWrapper) GetComments() int        { return i.Comments }

// prWrapper wraps a gitea.PullRequest to implement contentable
type prWrapper struct {
	*sdk.PullRequest
}

func (p *prWrapper) GetTitle() string        { return p.Title }
func (p *prWrapper) GetBody() string         { return p.Body }
func (p *prWrapper) GetHTMLURL() string      { return p.HTMLURL }
func (p *prWrapper) GetLabels() []*sdk.Label { return p.Labels }
func (p *prWrapper) GetPoster() *sdk.User    { return p.Poster }
func (p *prWrapper) GetID() int64            { return p.ID }
func (p *prWrapper) GetIndex() int64         { return p.Index }
func (p *prWrapper) GetState() sdk.StateType { return p.State }
func (p *prWrapper) GetCreated() time.Time   { return *p.Created }
func (p *prWrapper) GetUpdated() time.Time   { return *p.Updated }
func (p *prWrapper) GetComments() int        { return p.Comments }

type Pageable interface {
	SetNextPage(int)
}

type ListOptions struct {
	Owner   string
	Repo    string
	HasNext bool
	Pageable
}

func (c *ListOptions) OnResponse(resp *sdk.Response, err error) {
	if err != nil || resp == nil || resp.NextPage <= 0 {
		c.HasNext = false
	} else {
		c.SetNextPage(resp.NextPage)
		c.HasNext = true
	}
}

type ListReposCursor struct {
	ListOptions
	Options sdk.ListReposOptions
	IsOrg   bool
}

func NewListReposCursor(owner string, isOrg bool) *ListReposCursor {
	opt := sdk.ListReposOptions{
		ListOptions: withFirstPage(),
	}
	cursor := &ListReposCursor{Options: opt}
	cursor.Owner = owner
	cursor.IsOrg = isOrg
	cursor.Pageable = cursor // Self-reference to implement Pageable
	return cursor
}

func (c *ListReposCursor) SetNextPage(page int) { c.Options.Page = page }

type ListPullRequestsCursor struct {
	ListOptions
	Options sdk.ListPullRequestsOptions
}

func NewListPullRequestsCursor(owner, repo string) *ListPullRequestsCursor {
	opt := sdk.ListPullRequestsOptions{
		ListOptions: withFirstPage(),
		State:       sdk.StateAll,
	}
	cursor := &ListPullRequestsCursor{Options: opt}
	cursor.Owner = owner
	cursor.Repo = repo
	cursor.Pageable = cursor // Self-reference to implement Pageable
	return cursor
}

func (c *ListPullRequestsCursor) SetNextPage(page int) { c.Options.Page = page }

type ListIssuesCursor struct {
	ListOptions
	Options sdk.ListIssueOption
}

func NewListIssuesCursor(owner, repo string) *ListIssuesCursor {
	opt := sdk.ListIssueOption{
		ListOptions: withFirstPage(),
		State:       sdk.StateAll,
		Type:        sdk.IssueTypeIssue,
	}
	cursor := &ListIssuesCursor{Options: opt}
	cursor.Owner = owner
	cursor.Repo = repo
	cursor.Pageable = cursor // Self-reference to implement Pageable
	return cursor
}

func (c *ListIssuesCursor) SetNextPage(page int) { c.Options.Page = page }

type ListCommentsCursor struct {
	ListOptions
	Options sdk.ListIssueCommentOptions
	Index   int64
}

func NewListCommentsCursor(owner, repo string, index int64) *ListCommentsCursor {
	opt := sdk.ListIssueCommentOptions{
		ListOptions: withFirstPage(),
	}
	cursor := &ListCommentsCursor{Options: opt}
	cursor.Owner = owner
	cursor.Repo = repo
	cursor.Index = index
	cursor.Pageable = cursor // Self-reference to implement Pageable
	return cursor
}

func (c *ListCommentsCursor) SetNextPage(page int) { c.Options.Page = page }

func withFirstPage() sdk.ListOptions {
	return sdk.ListOptions{Page: 1, PageSize: DefaultPageSize}
}
