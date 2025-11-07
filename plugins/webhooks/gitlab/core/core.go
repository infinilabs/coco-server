/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

type MergeRequestEvent struct {
	ObjectKind       string            `json:"object_kind"`
	EventType        string            `json:"event_type"`
	User             UserInfo          `json:"user"`
	Project          ProjectInfo       `json:"project"`
	Repository       RepositoryInfo    `json:"repository"`
	ObjectAttributes MergeRequestAttrs `json:"object_attributes"`
	Labels           []LabelInfo       `json:"labels"`
	Changes          map[string]Change `json:"changes,omitempty"`
}

type UserInfo struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	WebURL    string `json:"web_url"`
	Email     string `json:"email"`
	State     string `json:"state"`
}

type ProjectInfo struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	Namespace         string  `json:"namespace"`
	PathWithNamespace string  `json:"path_with_namespace"`
	WebURL            string  `json:"web_url"`
	GitSSHURL         string  `json:"git_ssh_url"`
	GitHTTPURL        string  `json:"git_http_url"`
	DefaultBranch     string  `json:"default_branch"`
	Description       *string `json:"description"`
}

type RepositoryInfo struct {
	Name        string  `json:"name"`
	URL         string  `json:"url"`
	Description *string `json:"description"`
	Homepage    string  `json:"homepage"`
}

type MergeRequestAttrs struct {
	ID                          int64          `json:"id"`
	IID                         int64          `json:"iid"`
	TargetBranch                string         `json:"target_branch"`
	SourceBranch                string         `json:"source_branch"`
	Title                       string         `json:"title"`
	Description                 string         `json:"description"`
	Action                      string         `json:"action"`
	State                       string         `json:"state"`
	StateID                     int64          `json:"state_id"`
	MergeStatus                 string         `json:"merge_status"`
	DetailedMergeStatus         string         `json:"detailed_merge_status"`
	URL                         string         `json:"url"`
	AuthorID                    int64          `json:"author_id"`
	AssigneeIDs                 []int64        `json:"assignee_ids"`
	ReviewerIDs                 []int64        `json:"reviewer_ids"`
	SourceProjectID             int64          `json:"source_project_id"`
	TargetProjectID             int64          `json:"target_project_id"`
	CreatedAt                   string         `json:"created_at"`
	UpdatedAt                   string         `json:"updated_at"`
	LastCommit                  CommitInfo     `json:"last_commit"`
	MergeParams                 map[string]any `json:"merge_params"`
	WorkInProgress              bool           `json:"work_in_progress"`
	Draft                       bool           `json:"draft"`
	BlockingDiscussionsResolved bool           `json:"blocking_discussions_resolved"`
	FirstContribution           bool           `json:"first_contribution"`
	Source                      ProjectInfo    `json:"source"`
	Target                      ProjectInfo    `json:"target"`
}

type CommitInfo struct {
	ID        string       `json:"id"`
	Message   string       `json:"message"`
	Title     string       `json:"title"`
	Timestamp string       `json:"timestamp"`
	URL       string       `json:"url"`
	Author    CommitAuthor `json:"author"`
}

type CommitAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type LabelInfo struct {
	ID    int64  `json:"id,omitempty"`
	Title string `json:"title,omitempty"`
}

type Change struct {
	Previous any `json:"previous,omitempty"`
	Current  any `json:"current,omitempty"`
}

type MergeRequestDetail struct {
	ID                          int64                `json:"id"`
	IID                         int64                `json:"iid"`
	ProjectID                   int64                `json:"project_id"`
	Title                       string               `json:"title"`
	Description                 string               `json:"description"`
	State                       string               `json:"state"`
	CreatedAt                   string               `json:"created_at"`
	UpdatedAt                   string               `json:"updated_at"`
	MergedBy                    *UserInfo            `json:"merged_by"`
	MergeUser                   *UserInfo            `json:"merge_user"`
	MergedAt                    *string              `json:"merged_at"`
	ClosedBy                    *UserInfo            `json:"closed_by"`
	ClosedAt                    *string              `json:"closed_at"`
	TargetBranch                string               `json:"target_branch"`
	SourceBranch                string               `json:"source_branch"`
	UserNotesCount              int64                `json:"user_notes_count"`
	Upvotes                     int64                `json:"upvotes"`
	Downvotes                   int64                `json:"downvotes"`
	Author                      UserInfo             `json:"author"`
	Assignees                   []UserInfo           `json:"assignees"`
	Assignee                    *UserInfo            `json:"assignee"`
	Reviewers                   []UserInfo           `json:"reviewers"`
	SourceProjectID             int64                `json:"source_project_id"`
	TargetProjectID             int64                `json:"target_project_id"`
	Labels                      []string             `json:"labels"`
	Draft                       bool                 `json:"draft"`
	WorkInProgress              bool                 `json:"work_in_progress"`
	Milestone                   *MilestoneInfo       `json:"milestone"`
	MergeWhenPipelineSucceeds   bool                 `json:"merge_when_pipeline_succeeds"`
	MergeStatus                 string               `json:"merge_status"`
	DetailedMergeStatus         string               `json:"detailed_merge_status"`
	SHA                         string               `json:"sha"`
	MergeCommitSHA              *string              `json:"merge_commit_sha"`
	SquashCommitSHA             *string              `json:"squash_commit_sha"`
	DiscussionLocked            *bool                `json:"discussion_locked"`
	ShouldRemoveSourceBranch    *bool                `json:"should_remove_source_branch"`
	ForceRemoveSourceBranch     bool                 `json:"force_remove_source_branch"`
	PreparedAt                  string               `json:"prepared_at"`
	Reference                   string               `json:"reference"`
	References                  MRReferences         `json:"references"`
	WebURL                      string               `json:"web_url"`
	TimeStats                   MRTimeStats          `json:"time_stats"`
	Squash                      bool                 `json:"squash"`
	SquashOnMerge               bool                 `json:"squash_on_merge"`
	TaskCompletionStatus        TaskCompletionStatus `json:"task_completion_status"`
	HasConflicts                bool                 `json:"has_conflicts"`
	BlockingDiscussionsResolved bool                 `json:"blocking_discussions_resolved"`
	ApprovalsBeforeMerge        *int                 `json:"approvals_before_merge"`
	Subscribed                  bool                 `json:"subscribed"`
	ChangesCount                string               `json:"changes_count"`
	LatestBuildStartedAt        *string              `json:"latest_build_started_at"`
	LatestBuildFinishedAt       *string              `json:"latest_build_finished_at"`
	FirstDeployedToProductionAt *string              `json:"first_deployed_to_production_at"`
	Pipeline                    *PipelineInfo        `json:"pipeline"`
	HeadPipeline                *PipelineInfo        `json:"head_pipeline"`
	DiffRefs                    DiffRefs             `json:"diff_refs"`
	MergeError                  *string              `json:"merge_error"`
	FirstContribution           bool                 `json:"first_contribution"`
	User                        MRUserPermission     `json:"user"`
}

type MilestoneInfo struct {
	ID          int64   `json:"id"`
	IID         int64   `json:"iid"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	DueDate     *string `json:"due_date"`
	StartDate   *string `json:"start_date"`
	State       string  `json:"state"`
}

type MRReferences struct {
	Short    string `json:"short"`
	Relative string `json:"relative"`
	Full     string `json:"full"`
}

type MRTimeStats struct {
	TimeEstimate        int64   `json:"time_estimate"`
	TotalTimeSpent      int64   `json:"total_time_spent"`
	HumanTimeEstimate   *string `json:"human_time_estimate"`
	HumanTotalTimeSpent *string `json:"human_total_time_spent"`
}

type TaskCompletionStatus struct {
	Count          int64 `json:"count"`
	CompletedCount int64 `json:"completed_count"`
}

type PipelineInfo struct {
	ID     int64  `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
	Ref    string `json:"ref,omitempty"`
	SHA    string `json:"sha,omitempty"`
	WebURL string `json:"web_url,omitempty"`
}

type DiffRefs struct {
	BaseSHA  string `json:"base_sha"`
	HeadSHA  string `json:"head_sha"`
	StartSHA string `json:"start_sha"`
}

type MRUserPermission struct {
	CanMerge bool `json:"can_merge"`
}

type MRCommit struct {
	ID               string            `json:"id"`
	ShortID          string            `json:"short_id"`
	CreatedAt        string            `json:"created_at"`
	ParentIDs        []string          `json:"parent_ids"`
	Title            string            `json:"title"`
	Message          string            `json:"message"`
	AuthorName       string            `json:"author_name"`
	AuthorEmail      string            `json:"author_email"`
	AuthoredDate     string            `json:"authored_date"`
	CommitterName    string            `json:"committer_name"`
	CommitterEmail   string            `json:"committer_email"`
	CommittedDate    string            `json:"committed_date"`
	Trailers         map[string]string `json:"trailers"`
	ExtendedTrailers map[string]string `json:"extended_trailers"`
	WebURL           string            `json:"web_url"`
}

type MRDiff struct {
	Diff          string `json:"diff"`
	Collapsed     bool   `json:"collapsed"`
	TooLarge      bool   `json:"too_large"`
	NewPath       string `json:"new_path"`
	OldPath       string `json:"old_path"`
	AMode         string `json:"a_mode"`
	BMode         string `json:"b_mode"`
	NewFile       bool   `json:"new_file"`
	RenamedFile   bool   `json:"renamed_file"`
	DeletedFile   bool   `json:"deleted_file"`
	GeneratedFile bool   `json:"generated_file"`
}

type FileContent struct {
	FileName        string `json:"file_name"`
	FilePath        string `json:"file_path"`
	Size            int    `json:"size"`
	Encoding        string `json:"encoding"`
	ContentSHA256   string `json:"content_sha256"`
	Ref             string `json:"ref"`
	BlobID          string `json:"blob_id"`
	CommitID        string `json:"commit_id"`
	LastCommitID    string `json:"last_commit_id"`
	ExecuteFilemode bool   `json:"execute_filemode"`
	Content         string `json:"content"`
}

type MergeRequestNote struct {
	ID             int64                  `json:"id"`
	Type           *string                `json:"type"` // nullable
	Body           string                 `json:"body"`
	Author         NoteAuthor             `json:"author"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
	System         bool                   `json:"system"`
	NoteableID     int64                  `json:"noteable_id"`
	NoteableType   string                 `json:"noteable_type"`
	ProjectID      int64                  `json:"project_id"`
	Resolvable     bool                   `json:"resolvable"`
	Confidential   bool                   `json:"confidential"`
	Internal       bool                   `json:"internal"`
	Imported       bool                   `json:"imported"`
	ImportedFrom   string                 `json:"imported_from"`
	NoteableIID    int64                  `json:"noteable_iid"`
	CommandsChange map[string]interface{} `json:"commands_changes"`
}

type NoteAuthor struct {
	ID          int64   `json:"id"`
	Username    string  `json:"username"`
	PublicEmail *string `json:"public_email"` // nullable
	Name        string  `json:"name"`
	State       string  `json:"state"`
	Locked      bool    `json:"locked"`
	AvatarURL   string  `json:"avatar_url"`
	WebURL      string  `json:"web_url"`
}
