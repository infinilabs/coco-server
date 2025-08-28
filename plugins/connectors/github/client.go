/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package github

import (
	"context"

	log "github.com/cihub/seelog"
	githubv3 "github.com/google/go-github/v74/github"
	"golang.org/x/oauth2"
)

const DefaultPageSize = 100

// NewGitHubClient creates a new authenticated GitHub client.
func NewGitHubClient(token string) *githubv3.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	return githubv3.NewClient(tc)
}

type (
	RepoProcessor        func([]*githubv3.Repository) bool
	IssueProcessor       func([]*githubv3.Issue) bool
	PullRequestProcessor func([]*githubv3.PullRequest) bool
)

// ListRepos lists repositories for a user or organization, processing them page by page.
func ListRepos(ctx context.Context, client *githubv3.Client, user *githubv3.User, processor RepoProcessor) error {
	if user.GetType() == "Organization" {
		orgOpt := &githubv3.RepositoryListByOrgOptions{ListOptions: githubv3.ListOptions{PerPage: DefaultPageSize}}
		for {
			repos, resp, err := client.Repositories.ListByOrg(ctx, user.GetLogin(), orgOpt)
			if err != nil {
				return err
			}
			if ok := processor(repos); !ok {
				return nil
			}
			if resp.NextPage == 0 {
				break
			}
			orgOpt.Page = resp.NextPage
		}
		return nil
	}

	// Default to user repositories
	userOpt := &githubv3.RepositoryListByUserOptions{ListOptions: githubv3.ListOptions{PerPage: DefaultPageSize}}
	for {
		repos, resp, err := client.Repositories.ListByUser(ctx, user.GetLogin(), userOpt)
		if err != nil {
			return err
		}
		if ok := processor(repos); !ok {
			return nil
		}
		if resp.NextPage == 0 {
			break
		}
		userOpt.Page = resp.NextPage
	}
	return nil
}

// ListIssues lists all issues for a repository, processing them page by page.
func ListIssues(ctx context.Context, client *githubv3.Client, owner, repo string, processor IssueProcessor) error {
	opt := &githubv3.IssueListByRepoOptions{
		State:       "all",
		ListOptions: githubv3.ListOptions{PerPage: DefaultPageSize},
	}
	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, owner, repo, opt)
		if err != nil {
			return err
		}
		if ok := processor(issues); !ok {
			return nil
		}
		if resp.NextPage == 0 {
			break
		}
		opt.ListOptions.Page = resp.NextPage
	}
	return nil
}

// ListPullRequests lists all pull requests for a repository, processing them page by page.
func ListPullRequests(ctx context.Context, client *githubv3.Client, owner, repo string, processor PullRequestProcessor) error {
	opt := &githubv3.PullRequestListOptions{
		State:       "all",
		ListOptions: githubv3.ListOptions{PerPage: DefaultPageSize},
	}
	for {
		prs, resp, err := client.PullRequests.List(ctx, owner, repo, opt)
		if err != nil {
			return err
		}
		if ok := processor(prs); !ok {
			return nil
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return nil
}

// ListComments lists all comments for an issue or pull request, returning a slice.
func ListComments(ctx context.Context, client *githubv3.Client, owner, repo string, number int) ([]*githubv3.IssueComment, error) {
	opt := &githubv3.IssueListCommentsOptions{
		ListOptions: githubv3.ListOptions{PerPage: DefaultPageSize},
	}
	var res []*githubv3.IssueComment
	for {
		comments, resp, err := client.Issues.ListComments(ctx, owner, repo, number, opt)
		if err != nil {
			_ = log.Warnf("[%s connector] failed to list comments [repo=%s/%s, number=#%d]: %v", ConnectorGitHub, owner, repo, number, err)
			return res, err
		}
		res = append(res, comments...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return res, nil
}
