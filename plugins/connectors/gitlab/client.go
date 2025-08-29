/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package gitlab

import (
	"context"
	"net/http"

	log "github.com/cihub/seelog"
	gitlabv4 "gitlab.com/gitlab-org/api/client-go"
	"infini.sh/framework/core/errors"
)

const (
	ContextDone     errors.ErrorCode = 3
	DefaultPageSize                  = 100
)

// NewGitLabClient creates a new authenticated GitLab client.
func NewGitLabClient(token string, baseURL string) (*gitlabv4.Client, error) {
	var options []gitlabv4.ClientOptionFunc
	if baseURL != "" {
		options = append(options, gitlabv4.WithBaseURL(baseURL))
	}

	client, err := gitlabv4.NewClient(token, options...)
	if err != nil {
		return nil, err
	}
	return client, nil
}

type (
	ProjectProcessor      func([]*gitlabv4.Project) bool
	IssueProcessor        func([]*gitlabv4.Issue) bool
	MergeRequestProcessor func([]*gitlabv4.BasicMergeRequest) bool
	WikiProcessor         func([]*gitlabv4.Wiki) bool
	SnippetProcessor      func([]*gitlabv4.Snippet) bool
	ListProjects          func(context.Context, *gitlabv4.Client, string, ProjectProcessor) error
)

func isGroupOwner(ctx context.Context, client *gitlabv4.Client, ownerID any) (bool, error) {
	_, resp, err := client.Groups.GetGroup(ownerID, nil, gitlabv4.WithContext(ctx))

	if err == nil && resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

func ListGroupProjects(ctx context.Context, client *gitlabv4.Client, owner string, processor ProjectProcessor) error {
	opt := &gitlabv4.ListGroupProjectsOptions{ListOptions: gitlabv4.ListOptions{PerPage: DefaultPageSize}}
	for {
		select {
		case <-ctx.Done():
			return wrapContextDoneError(ctx.Err())
		default:
		}

		projects, resp, err := client.Groups.ListGroupProjects(owner, opt)
		if err != nil {
			return err
		}
		if ok := processor(projects); !ok {
			return nil
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return nil
}

func ListUserProjects(ctx context.Context, client *gitlabv4.Client, owner string, processor ProjectProcessor) error {
	opt := &gitlabv4.ListProjectsOptions{ListOptions: gitlabv4.ListOptions{PerPage: DefaultPageSize}}
	for {
		select {
		case <-ctx.Done():
			return wrapContextDoneError(ctx.Err())
		default:
		}

		projects, resp, err := client.Projects.ListUserProjects(owner, opt)
		if err != nil {
			return err
		}
		if ok := processor(projects); !ok {
			return nil
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return nil
}

// ListIssues lists all issues for a project, processing them page by page.
func ListIssues(ctx context.Context, client *gitlabv4.Client, projectID interface{}, processor IssueProcessor) error {
	opt := &gitlabv4.ListProjectIssuesOptions{
		ListOptions: gitlabv4.ListOptions{PerPage: DefaultPageSize},
	}
	for {
		select {
		case <-ctx.Done():
			return wrapContextDoneError(ctx.Err())
		default:
		}

		issues, resp, err := client.Issues.ListProjectIssues(projectID, opt)
		if err != nil {
			return err
		}
		if ok := processor(issues); !ok {
			return nil
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return nil
}

// ListMergeRequests lists all merge requests for a project, processing them page by page.
func ListMergeRequests(ctx context.Context, client *gitlabv4.Client, projectID interface{}, processor MergeRequestProcessor) error {
	opt := &gitlabv4.ListProjectMergeRequestsOptions{
		ListOptions: gitlabv4.ListOptions{PerPage: DefaultPageSize},
	}
	for {
		select {
		case <-ctx.Done():
			return wrapContextDoneError(ctx.Err())
		default:
		}

		prs, resp, err := client.MergeRequests.ListProjectMergeRequests(projectID, opt)
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

// ListWikiPages lists wiki pages for a project (no pagination supported by GitLab API).
// Note: ListWikisOptions does not support pagination
func ListWikiPages(ctx context.Context, client *gitlabv4.Client, projectID interface{}, processor WikiProcessor) error {
	select {
	case <-ctx.Done():
		return wrapContextDoneError(ctx.Err())
	default:
	}

	withContent := true
	pages, _, err := client.Wikis.ListWikis(projectID, &gitlabv4.ListWikisOptions{
		WithContent: &withContent,
	})
	if err != nil {
		return err
	}
	processor(pages)
	return nil
}

// ListProjectSnippets lists all snippets for a project, processing them page by page.
func ListProjectSnippets(ctx context.Context, client *gitlabv4.Client, projectID interface{}, processor SnippetProcessor) error {
	opt := &gitlabv4.ListProjectSnippetsOptions{
		PerPage: DefaultPageSize,
	}
	for {
		select {
		case <-ctx.Done():
			return wrapContextDoneError(ctx.Err())
		default:
		}

		snippets, resp, err := client.ProjectSnippets.ListSnippets(projectID, opt)
		if err != nil {
			return err
		}
		if ok := processor(snippets); !ok {
			return nil
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return nil
}

// ListComments lists all comments for an issue or merge request, returning a slice.
func ListComments(ctx context.Context, client *gitlabv4.Client, projectID interface{}, issueID int) ([]*gitlabv4.Note, error) {
	opt := &gitlabv4.ListIssueNotesOptions{
		ListOptions: gitlabv4.ListOptions{PerPage: DefaultPageSize},
	}
	var res []*gitlabv4.Note
	for {
		select {
		case <-ctx.Done():
			return nil, wrapContextDoneError(ctx.Err())
		default:
		}

		comments, resp, err := client.Notes.ListIssueNotes(projectID, issueID, opt)
		if err != nil {
			_ = log.Warnf("[%s connector] failed to list comments [projectID=%v, issueID=#%d]: %v", ConnectorGitLab, projectID, issueID, err)
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

func wrapContextDoneError(err error) error {
	return errors.NewWithCode(err, ContextDone, "context canceled")
}
