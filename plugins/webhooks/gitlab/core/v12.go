/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

type MergeRequestV12 struct {
	ID                          int           `json:"id"`
	IID                         int           `json:"iid"`
	ProjectID                   int           `json:"project_id"`
	Title                       string        `json:"title"`
	Description                 string        `json:"description"`
	State                       string        `json:"state"`
	CreatedAt                   string        `json:"created_at"`
	UpdatedAt                   string        `json:"updated_at"`
	MergedBy                    interface{}   `json:"merged_by"`
	MergedAt                    interface{}   `json:"merged_at"`
	ClosedBy                    interface{}   `json:"closed_by"`
	ClosedAt                    interface{}   `json:"closed_at"`
	TargetBranch                string        `json:"target_branch"`
	SourceBranch                string        `json:"source_branch"`
	UserNotesCount              int           `json:"user_notes_count"`
	Upvotes                     int           `json:"upvotes"`
	Downvotes                   int           `json:"downvotes"`
	Assignee                    interface{}   `json:"assignee"`
	Author                      UserV12       `json:"author"`
	Assignees                   []UserV12     `json:"assignees"`
	SourceProjectID             int           `json:"source_project_id"`
	TargetProjectID             int           `json:"target_project_id"`
	Labels                      []string      `json:"labels"`
	WorkInProgress              bool          `json:"work_in_progress"`
	Milestone                   interface{}   `json:"milestone"`
	MergeWhenPipelineSucceeds   bool          `json:"merge_when_pipeline_succeeds"`
	MergeStatus                 string        `json:"merge_status"`
	SHA                         string        `json:"sha"`
	MergeCommitSHA              interface{}   `json:"merge_commit_sha"`
	DiscussionLocked            interface{}   `json:"discussion_locked"`
	ShouldRemoveSourceBranch    interface{}   `json:"should_remove_source_branch"`
	ForceRemoveSourceBranch     bool          `json:"force_remove_source_branch"`
	Reference                   string        `json:"reference"`
	WebURL                      string        `json:"web_url"`
	TimeStats                   TimeStatsV12  `json:"time_stats"`
	Squash                      bool          `json:"squash"`
	TaskCompletionStatus        TaskStatusV12 `json:"task_completion_status"`
	Subscribed                  bool          `json:"subscribed"`
	ChangesCount                string        `json:"changes_count"`
	LatestBuildStartedAt        interface{}   `json:"latest_build_started_at"`
	LatestBuildFinishedAt       interface{}   `json:"latest_build_finished_at"`
	FirstDeployedToProductionAt interface{}   `json:"first_deployed_to_production_at"`
	Pipeline                    interface{}   `json:"pipeline"`
	HeadPipeline                *PipelineV12  `json:"head_pipeline"`
	DiffRefs                    DiffRefsV12   `json:"diff_refs"`
	MergeError                  interface{}   `json:"merge_error"`
	User                        CanMergeV12   `json:"user"`
	Changes                     []ChangeV12   `json:"changes"`
	ApprovalsBeforeMerge        interface{}   `json:"approvals_before_merge"`
}

type UserV12 struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	State     string `json:"state"`
	AvatarURL string `json:"avatar_url"`
	WebURL    string `json:"web_url"`
}

type TimeStatsV12 struct {
	TimeEstimate        int         `json:"time_estimate"`
	TotalTimeSpent      int         `json:"total_time_spent"`
	HumanTimeEstimate   interface{} `json:"human_time_estimate"`
	HumanTotalTimeSpent interface{} `json:"human_total_time_spent"`
}

type TaskStatusV12 struct {
	Count          int `json:"count"`
	CompletedCount int `json:"completed_count"`
}

type PipelineV12 struct {
	ID          int               `json:"id"`
	SHA         string            `json:"sha"`
	Ref         string            `json:"ref"`
	Status      string            `json:"status"`
	WebURL      string            `json:"web_url"`
	BeforeSHA   string            `json:"before_sha"`
	Tag         bool              `json:"tag"`
	YamlErrors  interface{}       `json:"yaml_errors"`
	User        UserV12           `json:"user"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
	StartedAt   interface{}       `json:"started_at"`
	FinishedAt  interface{}       `json:"finished_at"`
	CommittedAt interface{}       `json:"committed_at"`
	Duration    interface{}       `json:"duration"`
	Coverage    interface{}       `json:"coverage"`
	StatusInfo  DetailedStatusV12 `json:"detailed_status"`
}

type DetailedStatusV12 struct {
	Icon         string      `json:"icon"`
	Text         string      `json:"text"`
	Label        string      `json:"label"`
	Group        string      `json:"group"`
	Tooltip      string      `json:"tooltip"`
	HasDetails   bool        `json:"has_details"`
	DetailsPath  string      `json:"details_path"`
	Illustration interface{} `json:"illustration"`
	Favicon      string      `json:"favicon"`
}

type DiffRefsV12 struct {
	BaseSHA  string `json:"base_sha"`
	HeadSHA  string `json:"head_sha"`
	StartSHA string `json:"start_sha"`
}

type CanMergeV12 struct {
	CanMerge bool `json:"can_merge"`
}

type ChangeV12 struct {
	OldPath     string `json:"old_path"`
	NewPath     string `json:"new_path"`
	AMode       string `json:"a_mode"`
	BMode       string `json:"b_mode"`
	NewFile     bool   `json:"new_file"`
	RenamedFile bool   `json:"renamed_file"`
	DeletedFile bool   `json:"deleted_file"`
	Diff        string `json:"diff"`
}
