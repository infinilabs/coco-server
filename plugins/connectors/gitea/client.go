/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package gitea

import (
	"context"
	"strings"

	sdk "code.gitea.io/sdk/gitea"
	"infini.sh/coco/plugins/connectors"
)

const (
	DefaultPageSize = 50
	DefaultBaseURL  = "https://gitea.com"
)

// NewGiteaClient creates a new authenticated Gitea client.
func NewGiteaClient(baseURL string, token string) (*sdk.Client, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return sdk.NewClient(baseURL, sdk.SetToken(token))
}

func ListRepos(ctx context.Context, client *sdk.Client, cursor *ListReposCursor) ([]*sdk.Repository, error) {
	var repos []*sdk.Repository
	var resp *sdk.Response
	var err error

	if cursor.IsOrg {
		repos, resp, err = client.ListOrgRepos(cursor.Owner, sdk.ListOrgReposOptions(cursor.Options))
	} else {
		repos, resp, err = client.ListUserRepos(cursor.Owner, cursor.Options)
	}
	cursor.OnResponse(resp, err)
	return repos, err
}

// ListIssues lists all issues for a repository.
func ListIssues(ctx context.Context, client *sdk.Client, cursor *ListIssuesCursor) ([]*sdk.Issue, error) {
	if err := connectors.CheckContextDone(ctx); err != nil {
		return nil, err
	}

	issues, resp, err := client.ListRepoIssues(cursor.Owner, cursor.Repo, cursor.Options)
	cursor.OnResponse(resp, err)
	return issues, err
}

// ListPullRequests lists all pull requests for a repository.
func ListPullRequests(ctx context.Context, client *sdk.Client, cursor *ListPullRequestsCursor) ([]*sdk.PullRequest, error) {
	if err := connectors.CheckContextDone(ctx); err != nil {
		return nil, err
	}

	prs, resp, err := client.ListRepoPullRequests(cursor.Owner, cursor.Repo, cursor.Options)
	cursor.OnResponse(resp, err)
	return prs, err
}

// ListComments lists all comments for an issue or pull request.
func ListComments(ctx context.Context, client *sdk.Client, owner string, repo string, index int64) ([]*sdk.Comment, error) {
	cursor := NewListCommentsCursor(owner, repo, index)
	var res []*sdk.Comment
	for {
		err := connectors.CheckContextDone(ctx)
		if err != nil {
			return nil, err
		}

		comments, resp, err := client.ListIssueComments(cursor.Owner, cursor.Repo, cursor.Index, cursor.Options)
		if err != nil {
			return res, err
		}
		res = append(res, comments...)
		cursor.OnResponse(resp, err)

		// check whether it has next page
		if !cursor.HasNext {
			break
		}
	}
	return res, nil
}
